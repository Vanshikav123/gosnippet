package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	})
	fileServer := http.FileServer(http.Dir("./ui/static/"))
	router.Handler(http.MethodGet, "/static/*filepath", http.StripPrefix("/static", fileServer))

	// API routes with metrics tracking
	router.HandlerFunc(http.MethodGet, "/", app.withMetrics(app.home))
	router.HandlerFunc(http.MethodGet, "/snippet/view/:id", app.withMetrics(app.snippetView))
	router.HandlerFunc(http.MethodGet, "/snippet/create", app.withMetrics(app.snippetCreate))
	router.HandlerFunc(http.MethodPost, "/snippet/create", app.withMetrics(app.snippetCreatePost))

	// Metrics endpoint
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
