package jumpi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

var (
	signingMethod = jwt.SigningMethodHS256
	signingKey    = []byte{}
	globalStore   *Store

	ErrInvalidToken   = errors.New("Invalid Authentication Token")
	ErrInvalidObject  = errors.New("invalid object")
	ErrInvalidRequest = errors.New("unrecognized request")

	InternalServerError   = ErrorResponse{Status: http.StatusInternalServerError, Code: "err_internal", Description: "An internal server error has occured!"}
	UnprocessableEntity   = ErrorResponse{Status: 422, Code: "err_unprocessable", Description: "Unable to process given entity!"}
	AuthorizationRequired = ErrorResponse{Status: http.StatusUnauthorized, Code: "err_authorization_required", Description: "Please provide a valid Authorization Token to access this resource!"}
	BadRequest            = ErrorResponse{Status: http.StatusBadRequest, Code: "err_bad_request"}
)

func utcnow() time.Time {
	return time.Now().UTC()
}

// inspired by github.com/gorilla/securecookie
func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func init() {
	// override timefunc of jwt and always use UTC times
	jwt.TimeFunc = utcnow

	// create a new signing key
	signingKey = generateRandomKey(64)
}

type session struct {
	store *Store
	user  string
}

func (session *session) Login() (string, error) {
	username := strings.ToLower(session.user)
	token := jwt.New(signingMethod)
	token.Claims["usr"] = username
	token.Claims["nbf"] = jwt.TimeFunc().Unix()                    // not-before time
	token.Claims["exp"] = jwt.TimeFunc().Add(time.Hour * 2).Unix() // expiration time

	result, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	// store this session into session store for that user
	if err := session.store.Set(BucketSessions, "user~"+username, result); err != nil {
		return "", err
	}

	return result, err
}

func (session *session) Logout() error {
	username := strings.ToLower(session.user)
	return session.store.Delete(BucketSessions, "user~"+username)
}

func (session *session) Validate(rawToken string) (bool, *jwt.Token) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		// validate signing method (known vulnerability of JWTs)
		if token.Method.Alg() != signingMethod.Alg() {
			return nil, ErrInvalidToken
		}

		return signingKey, nil
	})

	if err == nil {
		if _, ok := token.Claims["usr"]; !ok {
			return false, nil
		}

		username, ok := token.Claims["usr"].(string)
		if !ok {
			return false, nil
		}

		// token is valid if JWT is valid and the stored session for this
		// user is the same as the JWT
		tkn, _ := session.store.Get(BucketSessions, "user~"+username)
		if tkn != rawToken {
			return false, nil
		}

		return token.Valid, token
	}
	return false, nil
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
		// altough token.Signature should hold the parsed signature, it is
		// always empty, so we parse the token ourselves. Note that at this
		// point we know we have a valid token, so this is save
		token := ses.(*jwt.Token)
		parts := strings.Split(token.Raw, ".")
		session = parts[2]
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

type Response struct {
	Status  int         `json:"status"`
	Content interface{} `json:"response"`
}

type ErrorResponse struct {
	Status      int    `json:"status"`
	Code        string `json:"error"`
	Description string `json:"description"`
}

func (r ErrorResponse) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(r.Status)

	if err := json.NewEncoder(w).Encode(r); err != nil {
		log.Fatal(err)
	}
}

func (r Response) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(r.Status)

	if err := json.NewEncoder(w).Encode(r); err != nil {
		InternalServerError.Write(w)
		log.Printf("unable to write answer: %s\n", err.Error())
	}
}

func StackMiddleware(handler http.HandlerFunc, mid ...func(http.Handler) http.HandlerFunc) http.HandlerFunc {
	for _, m := range mid {
		handler = m(handler)
	}
	return handler
}

func ContextMiddleware(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// load authenticated session and add user to context (if found)
		context.Set(r, "user", nil)
		context.Set(r, "session", nil)
		if tokens, ok := r.Header["Authorization"]; ok {
			for _, t := range tokens {
				// separate from first space, we want a Bearer token
				splits := strings.Split(t, " ")
				if len(splits) < 2 && splits[0] != "Bearer" {
					continue
				}

				session := &session{store: globalStore}
				if ok, token := session.Validate(splits[1]); ok {
					username, _ := token.Claims["usr"].(string)
					context.Set(r, "user", username)
					context.Set(r, "session", token)
					break
				}
			}
		}

		// handle request and clear context afterwards
		handler.ServeHTTP(w, r)
	}
}

func CheckRequestObject(r *http.Request, definition interface{}) error {
	if r == nil {
		return ErrInvalidRequest
	}

	// definition needs to be a pointer to a struct
	st := reflect.TypeOf(definition)
	if st.Kind() != reflect.Ptr {
		return ErrInvalidObject
	}
	st = st.Elem()
	if st.Kind() != reflect.Struct {
		return ErrInvalidObject
	}

	// parse json object from http request
	if err := json.NewDecoder(r.Body).Decode(definition); err != nil {
		return err
	}

	// iterate over all fields and check if the format is correct, if nothing is
	// specified then we just accept the json decoding
	for i := 0; i < st.NumField(); i += 1 {
		field := st.Field(i)
		format := field.Tag.Get("format")
		json_name := field.Tag.Get("json")

		if len(json_name) == 0 {
			json_name = field.Name
		}

		if len(format) > 0 && field.Type.Kind() == reflect.String {
			val := reflect.ValueOf(definition).Elem().Field(i).Interface()
			value, ok := val.(string)
			if !ok {
				return errors.New("invalid field \"" + json_name + "\"")
			}

			if matched, err := regexp.MatchString(format, value); err != nil || !matched {
				return errors.New("invalid field \"" + json_name + "\"")
			}
		}
	}

	return nil
}

func LoginRequired(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := context.Get(r, "user"); u != nil {
			handler.ServeHTTP(w, r)
		} else {
			AuthorizationRequired.Write(w)
		}
	}
}

func authLogin(w http.ResponseWriter, r *http.Request) {
	type _json struct {
		Username string `json:"username" format:"[a-z][a-z0-9\_\-]{2,}"`
		Password string `json:"password" format:".+"`
	}
	var cred _json

	if err := CheckRequestObject(r, &cred); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	// check username and password
	//TODO
	if false {
		AuthLoginFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_login_failed"}
		//AuthLoginFailed.Description = err.Error()
		AuthLoginFailed.Write(w)
		return
	}

	session := &session{user: cred.Username, store: globalStore}
	AuthLoginSuccessful := Response{Status: http.StatusOK}
	AuthLoginSuccessful.Content, _ = session.Login()
	AuthLoginSuccessful.Write(w)
}

func authLogout(w http.ResponseWriter, r *http.Request) {
	if username, ok := context.Get(r, "user").(string); ok {
		session := &session{store: globalStore, user: username}
		if err := session.Logout(); err != nil {
			AuthLogoutFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_logout_failed"}
			AuthLogoutFailed.Description = err.Error()
			AuthLogoutFailed.Write(w)
			return
		}
	}

	AuthLogoutSuccessful := Response{Status: http.StatusOK}
	AuthLogoutSuccessful.Write(w)
}

func authValidate(w http.ResponseWriter, r *http.Request) {
	// get authorization header and check validity of token
	if tokens, ok := r.Header["Authorization"]; ok {
		for _, t := range tokens {
			// separate from first space, we want a Bearer token
			splits := strings.Split(t, " ")
			if len(splits) < 2 && splits[0] != "Bearer" {
				continue
			}

			session := &session{store: globalStore}
			if ok, _ := session.Validate(splits[1]); ok {
				AuthTokenValid := Response{Status: http.StatusOK, Content: "Authorization Token Valid"}
				AuthTokenValid.Write(w)
				return
			}
		}
	}

	AuthTokenInvalid := Response{Status: http.StatusForbidden, Content: "Authorization Token Invalid"}
	AuthTokenInvalid.Write(w)
}

func StartAPIServer(root string, store *Store) {
	globalStore = store
	go func() {
		router := mux.NewRouter()
		router.KeepContext = true // we clean context ourselves in logger

		if strings.HasSuffix(root, "/") {
			root = root[:len(root)-1]
		}
		api := router.PathPrefix(root + "/api/").Subrouter()
		api = api.StrictSlash(true)

		api.Path("/auth/login").Methods("POST").HandlerFunc(authLogin)
		api.Path("/auth/logout").Methods("GET").HandlerFunc(StackMiddleware(authLogout, LoginRequired))
		api.Path("/auth/validate").Methods("GET").HandlerFunc(authValidate)

		logger := &logger{log.New(os.Stdout, "", 0)}
		n := negroni.New(negroni.NewRecovery(), logger)
		n.UseHandler(StackMiddleware(router.ServeHTTP, ContextMiddleware))
		n.Run("127.0.0.1:4200")
	}()
}