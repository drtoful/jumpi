package jumpi

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
)

var (
	signingMethod = jwt.SigningMethodHS256
	signingKey    = []byte{}

	v1routes = Routes{
		Route{Method: "POST", Pattern: "/auth/login", HandlerFunc: authLogin},
		Route{Method: "GET", Pattern: "/auth/logout", HandlerFunc: StackMiddleware(authLogout, LoginRequired)},
		Route{Method: "GET", Pattern: "/auth/validate", HandlerFunc: authValidate},

		Route{Method: "POST", Pattern: "/store/unlock", HandlerFunc: StackMiddleware(storeUnlock, LoginRequired)},
		Route{Method: "POST", Pattern: "/store/lock", HandlerFunc: StackMiddleware(storeLock, LoginRequired)},
		Route{Method: "GET", Pattern: "/store/status", HandlerFunc: StackMiddleware(storeStatus, LoginRequired)},

		Route{Method: "GET", Pattern: "/secrets/list", HandlerFunc: StackMiddleware(secretList, StoreUnlockRequired, LoginRequired)},
		Route{Method: "POST", Pattern: "/secrets", HandlerFunc: StackMiddleware(secretSet, StoreUnlockRequired, LoginRequired)},
		Route{Method: "DELETE", Pattern: "/secrets/{id}", HandlerFunc: StackMiddleware(secretDelete, StoreUnlockRequired, LoginRequired)},

		Route{Method: "GET", Pattern: "/targets/list", HandlerFunc: StackMiddleware(targetList, StoreUnlockRequired, LoginRequired)},
		Route{Method: "POST", Pattern: "/targets", HandlerFunc: StackMiddleware(targetSet, StoreUnlockRequired, LoginRequired)},
		Route{Method: "DELETE", Pattern: "/targets/{id}", HandlerFunc: StackMiddleware(targetDelete, StoreUnlockRequired, LoginRequired)},
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

	if username, ok := r.Context().Value("user").(string); ok {
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

/******************************************
 * STORE
 ******************************************/
func storeUnlock(w http.ResponseWriter, r *http.Request) {
	type _request struct {
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

	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.Unlock([]byte(request.Password)); err != nil {
		ResponseError(w, http.StatusForbidden, err)
		return
	}

	log.Printf("audit: %v unlocked store successfully\n", r.Context().Value("user"))
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

func storeLock(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.Lock(); err != nil {
		ResponseError(w, http.StatusForbidden, err)
		return
	}

	log.Printf("audit: %v locked store successfully\n", r.Context().Value("user"))
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

func storeStatus(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	c := make(map[string]interface{})
	c["locked"] = store.IsLocked()

	response := JSONResponse{
		Status:  http.StatusOK,
		Content: c,
	}
	response.Write(w)
}

/******************************************
 * SECRETS
 ******************************************/
func secretList(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	entries, err := store.Scan(BucketSecrets, "", 0, 0, true)
	if err != nil {
		ResponseError(w, http.StatusForbidden, err)
		return
	}

	type _response struct {
		Name        string     `json:"name"`
		Type        TypeSecret `json:"type"`
		Fingerprint string     `json:"fingerprint,omitempty"`
	}

	c := make([]_response, len(entries))
	i := 0
	for _, entry := range entries {
		// parse secret for more information
		secret := &Secret{
			ID: entry.Key,
		}
		if err := secret.Load(store); err != nil {
			continue
		}

		c[i] = _response{
			Name:        entry.Key,
			Type:        secret.Type,
			Fingerprint: secret.Fingerprint(),
		}
		i += 1
	}

	response := JSONResponse{
		Status:  http.StatusOK,
		Content: c,
	}
	response.Write(w)
}

func secretSet(w http.ResponseWriter, r *http.Request) {
	type _request struct {
		ID   string `json:"id" valid:"[^~\/]{3,}"`
		Type int    `json:"type"`
		Data string `json:"data"`
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

	secret := &Secret{
		ID: request.ID,
	}
	switch TypeSecret(request.Type) {
	case Password:
		secret.Secret = request.Data
		break
	case PKey:
		block, _ := pem.Decode([]byte(request.Data))
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			ResponseError(w, http.StatusBadRequest, err)
			return
		}

		pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			ResponseError(w, http.StatusBadRequest, err)
			return
		}

		secret.Secret = pkey
		break
	default:
		ResponseError(w, http.StatusBadRequest, errors.New("unknown type"))
		return
	}

	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if err := secret.Store(store); err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	log.Printf("audit: %v added secret '%s'\n", r.Context().Value("user"), request.ID)
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

func secretDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		ResponseError(w, http.StatusBadRequest, errors.New("id missing"))
		return
	}

	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	secret := &Secret{ID: id}
	if err := secret.Delete(store); err != nil {
		ResponseError(w, http.StatusBadRequest, err)
		return
	}

	log.Printf("audit: %v added secret '%s'\n", r.Context().Value("user"), id)
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

/******************************************
 * TARGETS
 ******************************************/
func targetList(w http.ResponseWriter, r *http.Request) {
	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	entries, err := store.Scan(BucketTargets, "", 0, 0, true)
	if err != nil {
		ResponseError(w, http.StatusForbidden, err)
		return
	}

	type _response struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}

	c := make([]_response, len(entries))
	i := 0
	for _, entry := range entries {
		c[i] = _response{
			Name:   entry.Key,
			Secret: entry.Value,
		}
		i += 1
	}

	response := JSONResponse{
		Status:  http.StatusOK,
		Content: c,
	}
	response.Write(w)
}

func targetSet(w http.ResponseWriter, r *http.Request) {
	type _request struct {
		Username string `json:"user" format:"\w+"`
		Hostname string `json:"host" format:".+"`
		Port     int    `json:"port"`
		Secret   string `json:"secret" format:".+"`
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

	if request.Port < 1 || request.Port > 65535 {
		ResponseError(w, http.StatusBadRequest, errors.New("port number out of range"))
		return
	}

	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	target := &Target{
		Username: request.Username,
		Hostname: request.Hostname,
		Port:     request.Port,
		Secret:   &Secret{ID: request.Secret},
	}

	if err := target.Store(store); err != nil {
		ResponseError(w, http.StatusBadRequest, err)
		return
	}

	log.Printf("audit: %v added target '%s' referencing secret '%s'\n", r.Context().Value("user"), target.ID(), request.Secret)
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

func targetDelete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		ResponseError(w, http.StatusBadRequest, errors.New("id missing"))
		return
	}

	store, err := GetStore(r)
	if err != nil {
		ResponseError(w, http.StatusInternalServerError, err)
		return
	}

	if err := store.Delete(BucketTargets, id); err != nil {
		ResponseError(w, http.StatusBadRequest, err)
		return
	}

	log.Printf("audit: %v removed target '%s'\n", r.Context().Value("user"), id)
	response := JSONResponse{
		Status: http.StatusOK,
	}
	response.Write(w)
}

// Main Router
func NewAPIv1Router(router *mux.Router) (*mux.Router, error) {
	router.StrictSlash(true)
	v1routes.Attach(router)
	return router, nil
}
