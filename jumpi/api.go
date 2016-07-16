package jumpi

import (
	"crypto/rand"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
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
	store         *Store

	ErrInvalidToken = errors.New("Invalid Authentication Token")
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

				session := &session{store: store}
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

func StartAPIServer(root string, store *Store) {
	store = store
	go func() {
		router := mux.NewRouter()
		router.KeepContext = true // we clean context ourselves in logger

		api := router.PathPrefix(root + "/api/").Subrouter()
		api = api.StrictSlash(true)

		logger := &logger{log.New(os.Stdout, "", 0)}
		n := negroni.New(negroni.NewRecovery(), logger)
		n.UseHandler(StackMiddleware(router.ServeHTTP, ContextMiddleware))
		n.Run("127.0.0.1:4200")
	}()
}
