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
	dynamic := alice.New(app.sessionManager.LoadAndSave)
	// API routes with metrics tracking
	router.Handler(http.MethodGet, "/", dynamic.ThenFunc(app.withMetrics(app.home)))
	router.Handler(http.MethodGet, "/snippet/view/:id", dynamic.ThenFunc(app.withMetrics(app.snippetView)))
	router.Handler(http.MethodGet, "/snippet/create", dynamic.ThenFunc(app.withMetrics(app.snippetCreate)))
	router.Handler(http.MethodPost, "/snippet/create", dynamic.ThenFunc(app.withMetrics(app.snippetCreatePost)))

	// Metrics endpoint
	router.Handler(http.MethodGet, "/metrics", promhttp.Handler())

	standard := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return standard.Then(router)
}
