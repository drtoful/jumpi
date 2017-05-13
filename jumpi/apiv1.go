package jumpi

import (
	"net/http"

	"github.com/gorilla/mux"
)

var (
	v1routes = Routes{
		Route{
			Name:        "AuthLogin",
			Method:      "POST",
			Pattern:     "/auth/login",
			HandlerFunc: authLogin,
		},
	}
)

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
}

func NewAPIv1Router(router *mux.Router) (*mux.Router, error) {
	router.StrictSlash(true)
	v1routes.Attach(router)
	return router, nil
}
