// Command hst-bi runs the BI server: API plus the embedded web app.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/samsmon/hst-bi-poc/server/internal/api"
	"github.com/samsmon/hst-bi-poc/server/internal/db"
	"github.com/samsmon/hst-bi-poc/server/internal/semantic"
	"github.com/samsmon/hst-bi-poc/server/internal/webui"
)

func main() {
	addr := env("BI_ADDR", "127.0.0.1:8610")
	schemas := strings.Split(env("BI_ALLOWED_SCHEMAS", "public,bi,ml"), ",")
	for i := range schemas {
		schemas[i] = strings.TrimSpace(schemas[i])
	}
	models, err := semantic.LoadEmbedded()
	if err != nil {
		log.Fatalf("semantic models: %v", err)
	}
	cfg := api.Config{AllowedSchemas: schemas, Models: models, Fallback: webui.Handler()}

	if url := os.Getenv("DATABASE_URL"); url != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err := db.Open(ctx, url, 6) // below the role's limit of 10
		cancel()
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		defer pool.Close()
		cfg.DB = pool
	} else {
		log.Print("DATABASE_URL is not set: data routes will answer 503")
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.New(cfg),
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
