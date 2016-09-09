package jumpi

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
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
	StoreUnlockedRequired = ErrorResponse{Status: http.StatusInternalServerError, Code: "store_locked", Description: "This operation needs unlocked store"}
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
	if err := session.store.SetRaw(BucketSessions, "user~"+username, []byte(result)); err != nil {
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
		tkn, _ := session.store.GetRaw(BucketSessions, "user~"+username)
		if string(tkn) != rawToken {
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
				if len(splits) < 2 || splits[0] != "Bearer" {
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

func StoreUnlockRequired(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if globalStore.IsLocked() {
			StoreUnlockedRequired.Write(w)
		} else {
			handler.ServeHTTP(w, r)
		}
	}
}

func validate(username, password string) error {
	if ok := globalStore.Auth(username, []byte(password)); !ok {
		return errors.New("invalid username/password")
	}

	return nil
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
	if err := validate(cred.Username, cred.Password); err != nil {
		AuthLoginFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_login_failed"}
		AuthLoginFailed.Description = err.Error()
		AuthLoginFailed.Write(w)
		return
	}

	session := &session{user: cred.Username, store: globalStore}
	bearer, err := session.Login()
	if err != nil {
		AuthLoginFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_login_failed"}
		AuthLoginFailed.Description = err.Error()
		AuthLoginFailed.Write(w)
		return
	}
	AuthLoginSuccessful := Response{Status: http.StatusOK}
	AuthLoginSuccessful.Content = bearer
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

func storeUnlock(w http.ResponseWriter, r *http.Request) {
	type _json struct {
		Password string `json:"password" format:".+"`
	}
	var pwd _json

	if err := CheckRequestObject(r, &pwd); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if err := globalStore.Unlock([]byte(pwd.Password)); err != nil {
		UnlockFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_unlock_failed"}
		UnlockFailed.Description = err.Error()
		UnlockFailed.Write(w)
		return
	}

	log.Printf("audit: %v unlocked store successfully\n", context.Get(r, "user"))
	UnlockSuccessful := Response{Status: http.StatusOK}
	UnlockSuccessful.Write(w)
}

func storeLock(w http.ResponseWriter, r *http.Request) {
	if err := globalStore.Lock(); err != nil {
		LockFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_lock_failed"}
		LockFailed.Description = err.Error()
		LockFailed.Write(w)
		return
	}

	log.Printf("audit: %v locked store successfully\n", context.Get(r, "user"))
	LockSuccessful := Response{Status: http.StatusOK}
	LockSuccessful.Write(w)
}

func storeStatus(w http.ResponseWriter, r *http.Request) {
	Status := Response{Status: http.StatusOK}
	Status.Content = globalStore.IsLocked()
	Status.Write(w)
}

func secretList(w http.ResponseWriter, r *http.Request) {
	skip := 0
	limit := 0

	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if vals, ok := r.Form["skip"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			skip = int(i)
		}
	}
	if vals, ok := r.Form["limit"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			limit = int(i)
		}
	}

	keys, err := globalStore.Scan(BucketSecrets, "", skip, limit)
	if err != nil {
		SecretListFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_list_failed"}
		SecretListFailed.Description = err.Error()
		SecretListFailed.Write(w)
		return
	}

	SecretList := Response{Status: http.StatusOK}
	SecretList.Content = keys
	SecretList.Write(w)
}

func secretSet(w http.ResponseWriter, r *http.Request) {
	type _json struct {
		ID   string `json:"id" format:"[^\/]{3,}"`
		Type int    `json:"type"`
		Data string `json:"data"`
	}
	var req _json

	if err := CheckRequestObject(r, &req); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	secret := &Secret{
		ID: req.ID,
	}
	switch TypeSecret(req.Type) {
	case Password:
		secret.Secret = req.Data
	case PKey:
		block, _ := pem.Decode([]byte(req.Data))
		if block == nil || block.Type != "RSA PRIVATE KEY" {
			SecretStoreFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_store_failed"}
			SecretStoreFailed.Description = "Unable to parse private key PEM"
			SecretStoreFailed.Write(w)
			return
		}
		pkey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			SecretStoreFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_store_failed"}
			SecretStoreFailed.Description = err.Error()
			SecretStoreFailed.Write(w)
			return
		}
		secret.Secret = pkey
	default:
		SecretStoreFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_store_failed"}
		SecretStoreFailed.Description = "Unsupported secret type"
		SecretStoreFailed.Write(w)
		return
	}

	if err := secret.Store(globalStore); err != nil {
		SecretStoreFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_store_failed"}
		SecretStoreFailed.Description = err.Error()
		SecretStoreFailed.Write(w)
		return
	}

	log.Printf("audit: %v added secret '%s'\n", context.Get(r, "user"), req.ID)
	SecretStoreSuccessful := Response{Status: http.StatusOK}
	SecretStoreSuccessful.Write(w)
}

func secretDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	id := ""
	if vals, ok := r.Form["id"]; ok {
		id = vals[0]
	}

	if len(id) == 0 {
		BadRequest.Description = "no id given"
		BadRequest.Write(w)
		return
	}

	secret := &Secret{
		ID: id,
	}
	if err := secret.Delete(globalStore); err != nil {
		SecretDeleteFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_secret_delete_failed"}
		SecretDeleteFailed.Description = err.Error()
		SecretDeleteFailed.Write(w)
		return
	}

	log.Printf("audit: %v removed secret '%s'\n", context.Get(r, "user"), id)
	SecretDeleteSuccessful := Response{Status: http.StatusOK}
	SecretDeleteSuccessful.Write(w)
}

func userList(w http.ResponseWriter, r *http.Request) {
	skip := 0
	limit := 0

	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if vals, ok := r.Form["skip"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			skip = int(i)
		}
	}
	if vals, ok := r.Form["limit"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			limit = int(i)
		}
	}

	keys, err := globalStore.Scan(BucketUsers, "", skip, limit)
	if err != nil {
		UserListFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_user_list_failed"}
		UserListFailed.Description = err.Error()
		UserListFailed.Write(w)
		return
	}

	UserList := Response{Status: http.StatusOK}
	UserList.Content = keys
	UserList.Write(w)
}

func userAdd(w http.ResponseWriter, r *http.Request) {
	type _json struct {
		Name string `json:"name" format:"[a-zA-Z0-9\-\_]+"`
		Pub  string `json:"pub" format:"ssh-rsa [A-Za-z0-9\+\/]+[\=]{0,2}.*"`
	}
	var req _json

	if err := CheckRequestObject(r, &req); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	user, err := UserFromPublicKey(req.Name, req.Pub)
	if err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if err := user.Store(globalStore); err != nil {
		UserCreateFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_user_create_failed"}
		UserCreateFailed.Description = err.Error()
		UserCreateFailed.Write(w)
		return
	}

	log.Printf("audit: %v added user '%s' with fingerprint '%s'\n", context.Get(r, "user"), req.Name, user.KeyFingerprint)
	UserCreateSuccessful := Response{Status: http.StatusOK}
	UserCreateSuccessful.Write(w)
}

func userDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	id := ""
	if vals, ok := r.Form["id"]; ok {
		id = vals[0]
	}

	if len(id) == 0 {
		BadRequest.Description = "no id given"
		BadRequest.Write(w)
		return
	}

	user := &User{
		KeyFingerprint: id,
	}
	if err := user.Delete(globalStore); err != nil {
		UserDeleteFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_user_delete_failed"}
		UserDeleteFailed.Description = err.Error()
		UserDeleteFailed.Write(w)
		return
	}

	log.Printf("audit: %v removed user '%s'\n", context.Get(r, "user"), id)
	UserDeleteSuccessful := Response{Status: http.StatusOK}
	UserDeleteSuccessful.Write(w)
}

func targetList(w http.ResponseWriter, r *http.Request) {
	skip := 0
	limit := 0

	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if vals, ok := r.Form["skip"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			skip = int(i)
		}
	}
	if vals, ok := r.Form["limit"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			limit = int(i)
		}
	}

	keys, err := globalStore.Scan(BucketTargets, "", skip, limit)
	if err != nil {
		TargetListFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_target_list_failed"}
		TargetListFailed.Description = err.Error()
		TargetListFailed.Write(w)
		return
	}

	TargetList := Response{Status: http.StatusOK}
	TargetList.Content = keys
	TargetList.Write(w)
}

func targetAdd(w http.ResponseWriter, r *http.Request) {
	type _json struct {
		Username string `json:"user" format:"\w+"`
		Hostname string `json:"host" format:".+"`
		Port     int    `json:"port"`
		Secret   string `json:"secret" format:".+"`
	}
	var req _json

	if err := CheckRequestObject(r, &req); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if req.Port < 1 || req.Port > 65535 {
		BadRequest.Description = "port number out of range"
		BadRequest.Write(w)
		return
	}

	target := &Target{
		Username: req.Username,
		Hostname: req.Hostname,
		Port:     req.Port,
		Secret:   &Secret{ID: req.Secret},
	}

	if err := target.Store(globalStore); err != nil {
		TargetCreateFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_target_create_failed"}
		TargetCreateFailed.Description = err.Error()
		TargetCreateFailed.Write(w)
		return
	}

	log.Printf("audit: %v added target '%s' referencing secret '%s'\n", context.Get(r, "user"), target.ID(), req.Secret)
	TargetCreateSuccessful := Response{Status: http.StatusOK}
	TargetCreateSuccessful.Write(w)
}

func targetDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	id := ""
	if vals, ok := r.Form["id"]; ok {
		id = vals[0]
	}

	if len(id) == 0 {
		BadRequest.Description = "no id given"
		BadRequest.Write(w)
		return
	}

	if err := globalStore.Delete(BucketTargets, id); err != nil {
		TargetDeleteFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_target_delete_failed"}
		TargetDeleteFailed.Description = err.Error()
		TargetDeleteFailed.Write(w)
		return
	}

	log.Printf("audit: %v removed target '%s'\n", context.Get(r, "user"), id)
	TargetDeleteSuccessful := Response{Status: http.StatusOK}
	TargetDeleteSuccessful.Write(w)
}

func roleList(w http.ResponseWriter, r *http.Request) {
	skip := 0
	limit := 0

	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if vals, ok := r.Form["skip"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			skip = int(i)
		}
	}
	if vals, ok := r.Form["limit"]; ok {
		if i, err := strconv.ParseInt(vals[0], 10, 64); err == nil {
			limit = int(i)
		}
	}

	keys, err := globalStore.Scan(BucketRoles, "", skip, limit)
	if err != nil {
		RoleListFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_role_list_failed"}
		RoleListFailed.Description = err.Error()
		RoleListFailed.Write(w)
		return
	}

	RoleList := Response{Status: http.StatusOK}
	RoleList.Content = keys
	RoleList.Write(w)
}

func roleAdd(w http.ResponseWriter, r *http.Request) {
	var req Role
	if err := CheckRequestObject(r, &req); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	if err := req.Store(globalStore); err != nil {
		RoleCreateFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_role_create_failed"}
		RoleCreateFailed.Description = err.Error()
		RoleCreateFailed.Write(w)
		return
	}

	log.Printf("audit: %v added role '%s' with user regex '%s' and target regex '%s'\n", context.Get(r, "user"), req.Name, req.UserRegex, req.TargetRegex)
	RoleCreateSuccessful := Response{Status: http.StatusOK}
	RoleCreateSuccessful.Write(w)
}

func roleDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	id := ""
	if vals, ok := r.Form["id"]; ok {
		id = vals[0]
	}

	if len(id) == 0 {
		BadRequest.Description = "no id given"
		BadRequest.Write(w)
		return
	}

	role := &Role{
		Name: id,
	}
	if err := role.Delete(globalStore); err != nil {
		RoleDeleteFailed := ErrorResponse{Status: http.StatusForbidden, Code: "err_role_delete_failed"}
		RoleDeleteFailed.Description = err.Error()
		RoleDeleteFailed.Write(w)
		return
	}

	log.Printf("audit: %v removed role '%s'\n", context.Get(r, "user"), id)
	RoleDeleteSuccessful := Response{Status: http.StatusOK}
	RoleDeleteSuccessful.Write(w)
}

func castGet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		BadRequest.Description = err.Error()
		BadRequest.Write(w)
		return
	}

	id := ""
	if vals, ok := r.Form["id"]; ok {
		id = vals[0]
	}

	if len(id) == 0 {
		BadRequest.Description = "no id given"
		BadRequest.Write(w)
		return
	}

	// raw load
	data, err := globalStore.Get(BucketCasts, id)
	if len(data) == 0 || err != nil {
		NotFound := ErrorResponse{Status: http.StatusNotFound, Code: "err_cast_not_found"}
		if err == nil {
			NotFound.Description = "Specified cast was not found"
		} else {
			NotFound.Description = err.Error()
		}
		NotFound.Write(w)
		return
	}

	var cast Cast
	if err := json.Unmarshal([]byte(data), &cast); err != nil {
		CastError := ErrorResponse{Status: http.StatusForbidden, Code: "err_cast_load_error"}
		CastError.Description = err.Error()
		CastError.Write(w)
		return
	}

	CastLoaded := Response{Status: http.StatusOK}
	CastLoaded.Content = cast
	CastLoaded.Write(w)
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

		api.Path("/store/unlock").Methods("POST").HandlerFunc(StackMiddleware(storeUnlock, LoginRequired))
		api.Path("/store/lock").Methods("POST").HandlerFunc(StackMiddleware(storeLock, LoginRequired))
		api.Path("/store/status").Methods("GET").HandlerFunc(StackMiddleware(storeStatus, LoginRequired))

		api.Path("/secrets").Methods("GET").HandlerFunc(StackMiddleware(secretList, StoreUnlockRequired, LoginRequired))
		api.Path("/secrets").Methods("POST").HandlerFunc(StackMiddleware(secretSet, StoreUnlockRequired, LoginRequired))
		api.Path("/secrets").Methods("DELETE").HandlerFunc(StackMiddleware(secretDelete, LoginRequired))

		api.Path("/users").Methods("GET").HandlerFunc(StackMiddleware(userList, StoreUnlockRequired, LoginRequired))
		api.Path("/users").Methods("POST").HandlerFunc(StackMiddleware(userAdd, StoreUnlockRequired, LoginRequired))
		api.Path("/users").Methods("DELETE").HandlerFunc(StackMiddleware(userDelete, LoginRequired))

		api.Path("/targets").Methods("GET").HandlerFunc(StackMiddleware(targetList, StoreUnlockRequired, LoginRequired))
		api.Path("/targets").Methods("POST").HandlerFunc(StackMiddleware(targetAdd, StoreUnlockRequired, LoginRequired))
		api.Path("/targets").Methods("DELETE").HandlerFunc(StackMiddleware(targetDelete, LoginRequired))

		api.Path("/roles").Methods("GET").HandlerFunc(StackMiddleware(roleList, StoreUnlockRequired, LoginRequired))
		api.Path("/roles").Methods("POST").HandlerFunc(StackMiddleware(roleAdd, StoreUnlockRequired, LoginRequired))
		api.Path("/roles").Methods("DELETE").HandlerFunc(StackMiddleware(roleDelete, LoginRequired))

		api.Path("/casts").Methods("GET").HandlerFunc(StackMiddleware(castGet, StoreUnlockRequired, LoginRequired))

		logger := &logger{log.New(os.Stdout, "", 0)}
		n := negroni.New(negroni.NewRecovery(), logger)
		n.UseHandler(StackMiddleware(router.ServeHTTP, ContextMiddleware))
		n.Run("127.0.0.1:4200")
	}()
}
