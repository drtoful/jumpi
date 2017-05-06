package api

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
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
