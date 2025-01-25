package main

import (
	"crypto/tls"
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"text/template" // New import
	"time"

	"github.com/Vanshikav123/gosnippet.git/internal/models"
	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
)

type application struct {
	errorLog          *log.Logger
	infoLog           *log.Logger
	snippets          *models.SnippetModel
	users             *models.UserModel
	templateCache     map[string]*template.Template
	formDecoder       *form.Decoder
	apiRequestCounter *prometheus.CounterVec
	sessionManager    *scs.SessionManager
}

func main() {
	addr := flag.String("addr", ":4000", "http network address")
	dsn := flag.String("dsn", "web:pass@/snippetbox?parseTime=true", "MySQL data source name")

	flag.Parse()
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer db.Close()

	templateCache, err := newTemplateCache()
	if err != nil {
		errorLog.Fatal(err)
	}
	formDecoder := form.NewDecoder()

	sessionManager := scs.New()
	sessionManager.Store = mysqlstore.New(db)
	sessionManager.Lifetime = 12 * time.Hour

	// Initialize Prometheus metrics
	apiRequestCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gosnippet_api_requests_total",
			Help: "Total number of API requests processed by GoSnippet.",
		},
		[]string{"endpoint", "method"},
	)
	prometheus.MustRegister(apiRequestCounter)

	app := &application{
		errorLog:          errorLog,
		infoLog:           infoLog,
		snippets:          &models.SnippetModel{DB: db},
		users:             &models.UserModel{DB: db},
		templateCache:     templateCache,
		formDecoder:       formDecoder,
		apiRequestCounter: apiRequestCounter, // Attach the counter
		sessionManager:    sessionManager,
	}

	tlsConfig := &tls.Config{
		CurvePreferences: []tls.CurveID{tls.X25519, tls.CurveP256},
	}

	srv := &http.Server{
		Addr:      *addr,
		ErrorLog:  errorLog,
		Handler:   app.routes(),
		TLSConfig: tlsConfig,

		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServeTLS("./tls/cert.pem", "./tls/key.pem")
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
