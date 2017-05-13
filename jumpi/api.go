package jumpi

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

var (
	globalStore *Store
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

type Routes []Route

func (r Route) Attach(router *mux.Router) {
	log.Printf("api_server: attaching route: %s %s\n", r.Method, r.Pattern)

	router.Methods(r.Method).
		Path(r.Pattern).
		Name(r.Name).
		Handler(r.HandlerFunc)
}

func (r Routes) Attach(router *mux.Router) {
	for _, route := range r {
		route.Attach(router)
	}
}

type logger struct {
	*log.Logger
}

func (l *logger) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	start := time.Now()
	next(w, r)
	res := w.(negroni.ResponseWriter)

	session := "-"
	if ses := context.Get(r, "session"); ses != nil {
		session = ses.(string)
	}

	user := "-"
	if usr := context.Get(r, "user"); usr != nil {
		user = usr.(string)
	}

	useragent := r.Header.Get("User-Agent")
	referer := r.Header.Get("Referer")
	duration := int64(time.Since(start) / time.Millisecond)
	remote := "-"
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		remote = host
	}

	l.Printf("%s %s %s \"%s %s %s\" %d %d \"%s\" \"%s\" %s %dms\n",
		time.Now().UTC().Format(time.RFC3339),
		remote,
		user,
		r.Method,
		r.URL,
		r.Proto,
		res.Status(),
		res.Size(),
		useragent,
		referer,
		session,
		duration,
	)

	// logging is the last action, so here we can clear the context
	context.Clear(r)
}

type JSONRequest struct {
	obj interface{}
}

type JSONResponse struct {
	Status  int         `json:"status"`
	Content interface{} `json:"response"`
}

type JSONResponseError JSONResponse

var (
	LimitRequest int64 = 4096
)

func ParseJsonRequest(r *http.Request, v interface{}) (*JSONRequest, error) {
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, LimitRequest))
	if err != nil {
		return nil, err
	}

	if err := r.Body.Close(); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, v); err != nil {
		return nil, err
	}

	return &JSONRequest{obj: v}, nil
}

func (jr JSONRequest) Validate() error {
	// object needs to be a pointer to a struct
	st := reflect.TypeOf(jr.obj)
	if st.Kind() != reflect.Ptr {
		return errors.New("invalid request")
	}

	st = st.Elem()
	if st.Kind() != reflect.Struct {
		return errors.New("invalid request")
	}

	// iterate over all fields and check if the format is correct, if nothing
	// is specified then we just accept the json decoding
	for i := 0; i < st.NumField(); i += 1 {
		field := st.Field(i)
		format := field.Tag.Get("valid")
		json_name := field.Tag.Get("json")

		if len(json_name) == 0 {
			json_name = field.Name
		}

		if len(format) > 0 && field.Type.Kind() == reflect.String {
			val := reflect.ValueOf(jr.obj).Elem().Field(i).Interface()
			value, ok := val.(string)
			if !ok {
				return errors.New("invalid field \"" + json_name + "\"")
			}

			// check if value complies to format
			if matched, err := regexp.MatchString(format, value); err != nil || !matched {
				return errors.New("invalid field \"" + json_name + "\"")
			}
		}
	}

	return nil
}

func (jr JSONResponse) Write(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(jr.Status)

	if err := json.NewEncoder(w).Encode(jr); err != nil {
		return err
	}

	return nil
}

func ResponseError(w http.ResponseWriter, status int, e error) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(e.Error()); err != nil {
		log.Fatalf("api_server: unable to send error response: %s\n", err.Error())
	}
}

func ContextMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// load authenticated session and add user to context (if found)
		context.Set(r, "user", nil)
		context.Set(r, "session", nil)

		//if tokens, ok := r.Header["Authorization"]; ok {
		//}

		// pass request to next handler
		handler.ServeHTTP(w, r)
	}
}

func StackMiddleware(handler http.HandlerFunc, mid ...func(http.Handler) http.HandlerFunc) http.HandlerFunc {
	for _, m := range mid {
		handler = m(handler)
	}
	return handler
}

func StartAPIServer(root string, store *Store) {
	globalStore = store
	go func() {
		router := mux.NewRouter()
		router.KeepContext = true // we clean context ourselves in logger

		if strings.HasSuffix(root, "/") {
			root = root[:len(root)-1]
		}

		// attach APIv1
		apiv1 := router.PathPrefix(root + "/api/v1/").Subrouter()
		apiv1.StrictSlash(true)
		if _, err := NewAPIv1Router(apiv1); err != nil {
			log.Fatal(err)
		}

		logger := &logger{log.New(os.Stdout, "", 0)}
		n := negroni.New(negroni.NewRecovery(), logger)
		n.UseHandler(StackMiddleware(router.ServeHTTP, ContextMiddleware))
		n.Run("127.0.0.1:4200")
	}()
}
