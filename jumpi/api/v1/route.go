package apiv1

import (
	"github.com/gorilla/mux"
)

func NewRouter(router *mux.Router) (*mux.Router, error) {
	router.StrictSlash(true)

	// attaching all necessary routes
	RoutesAuth.Attach(router)

	return router, nil
}
