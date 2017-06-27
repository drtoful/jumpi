package jumpi

import (
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

var (
	signingMethod = jwt.SigningMethodHS256
	signingKey    = []byte{}

	v1routes = Routes{
		Route{Name: "AuthLogin", Method: "POST", Pattern: "/auth/login", HandlerFunc: authLogin},
		Route{Name: "AuthLogout", Method: "GET", Pattern: "/auth/logout", HandlerFunc: StackMiddleware(authLogout, LoginRequired)},
		Route{Name: "AuthVal", Method: "GET", Pattern: "/auth/validate", HandlerFunc: authValidate},
	}
)

func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

func init() {
	jwt.TimeFunc = func() time.Time {
		return time.Now().UTC()
	}
	signingKey = generateRandomKey(64)
}

type session struct {
	store *Store
	user  string
}

func (session *session) Login() (string, error) {
	// create new JWT token
	username := strings.ToLower(session.user)
	token := jwt.New(signingMethod)
	token.Claims["usr"] = username
	token.Claims["nbf"] = jwt.TimeFunc().Unix()
	token.Claims["exp"] = jwt.TimeFunc().Add(time.Hour * 2).Unix()

	result, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	if err := session.store.SetRaw(BucketSessions, "user~v1~"+username, []byte(result)); err != nil {
		return "", err
	}

	return result, err
}

func (session *session) Logout() error {
	username := strings.ToLower(session.user)
	return session.store.Delete(BucketSessions, "user~v1~"+username)
}

func (session *session) Validate(rawToken string) (bool, *jwt.Token) {
	token, err := jwt.Parse(rawToken, func(token *jwt.Token) (interface{}, error) {
		// validate signing method (known vulnerability of JWTs)
		if token.Method.Alg() != signingMethod.Alg() {
			return nil, errors.New("Invalid Authentication Token")
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

		// token is valid if JWT is valid and the stored session id for
		// this user is the same as the JWT
		username = strings.ToLower(username)
		tkn, _ := session.store.GetRaw(BucketSessions, "user~v1~"+username)
		if string(tkn) != rawToken {
			return false, nil
		}

		return token.Valid, token
	}

	return false, nil
}

/******************************************
 * AUTHENTICATION
 ******************************************/
func authLogin(w http.ResponseWriter, r *http.Request) {
	type _request struct {
		Username string `json:"username" valid:"^[a-z][a-z0-9\\-\\_]{2,}$"`
		Password string `json:"password"`
	}
	var request _request

	jreq, err := ParseJsonRequest(r, &request)
	if err != nil {
		ResponseError(w, 422, err)
		return
	}

	if err := jreq.Validate(); err != nil {
		ResponseError(w, http.StatusBadRequest, err)
		return
	}

	// check username and password
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if ok := store.Auth(request.Username, []byte(request.Password)); !ok {
		ResponseError(w, http.StatusForbidden, errors.New("invalid username/password"))
		return
	}

	// create a new session
	session := &session{user: request.Username, store: store}
	bearer, err := session.Login()
	if err != nil {
		ResponseError(w, http.StatusForbidden, err)
		return
	}

	response := JSONResponse{
		Status:  http.StatusOK,
		Content: bearer,
	}
	response.Write(w)
}

func authLogout(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if username, ok := context.Get(r, "user").(string); ok {
		session := &session{store: store, user: username}
		if err := session.Logout(); err != nil {
			ResponseError(w, http.StatusForbidden, err)
			return
		}
	}

	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

func authValidate(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	// get authorization header and check validity of token
	if tokens, ok := r.Header["Authorization"]; ok {
		for _, t := range tokens {
			// search for bearer tokens
			splits := strings.Split(t, " ")
			if len(splits) < 2 || splits[0] != "Bearer" {
				continue
			}

			session := &session{store: store}
			if ok, _ := session.Validate(splits[1]); ok {
				response := JSONResponse{
					Status: http.StatusOK,
				}
				response.Write(w)
				return
			}
		}
	}

	ResponseError(w, http.StatusForbidden, errors.New("Invalid Authorization Token"))
}

// Main Router
func NewAPIv1Router(router *mux.Router) (*mux.Router, error) {
	router.StrictSlash(true)
	v1routes.Attach(router)
	return router, nil
}
