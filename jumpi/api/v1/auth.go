package apiv1

import (
	"net/http"

	"github.com/drtoful/jumpi/jumpi/api"
)

var (
	RoutesAuth = api.Routes{
		api.Route{
			Name:        "AuthLogin",
			Method:      "POST",
			Pattern:     "/auth/login",
			HandlerFunc: login,
		},
	}
)

func login(w http.ResponseWriter, r *http.Request) {
	type _request struct {
		Username string `json:"username" valid:"^[a-z][a-z0-9\\-\\_]{2,}$"`
		Password string `json:"password"`
	}
	var request _request

	jreq, err := api.ParseJsonRequest(r, &request)
	if err != nil {
		api.ResponseError(w, 422, err)
		return
	}

	if err := jreq.Validate(); err != nil {
		api.ResponseError(w, http.StatusBadRequest, err)
		return
	}
}
