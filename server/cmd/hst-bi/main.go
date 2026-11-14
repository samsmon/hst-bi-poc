// Command hst-bi runs the BI server: API plus the embedded web app.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samsmon/hst-bi-poc/server/internal/api"
	"github.com/samsmon/hst-bi-poc/server/internal/webui"
)

func main() {
	addr := env("BI_ADDR", "127.0.0.1:8610")
	schemas := strings.Split(env("BI_ALLOWED_SCHEMAS", "public,ml"), ",")
	for i := range schemas {
		schemas[i] = strings.TrimSpace(schemas[i])
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(schemas, webui.Handler()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("hst-bi listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
