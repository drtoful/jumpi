package jumpi

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/negroni"
	"github.com/drtoful/jumpi/jumpi/api/v1"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

var (
	globalStore *Store
)

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

func StartAPIServer(root string, store *Store) {
	globalStore = store
	go func() {
		router := mux.NewRouter()
		router.KeepContext = true // we clean context ourselves in logger

		if strings.HasSuffix(root, "/") {
			root = root[:len(root)-1]
		}

		// attach APIv1
		api := router.PathPrefix(root + "/api/v1/").Subrouter()
		api = api.StrictSlash(true)

		api, err := apiv1.NewRouter(api)
		if err != nil {
			log.Fatal(err)
		}

		logger := &logger{log.New(os.Stdout, "", 0)}
		n := negroni.New(negroni.NewRecovery(), logger)
		n.UseHandler(router)
		//n.UseHandler(StackMiddleware(router.ServeHTTP, ContextMiddleware))
		n.Run("127.0.0.1:4200")
	}()
}
