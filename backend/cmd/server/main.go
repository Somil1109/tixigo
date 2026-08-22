package main

import (
	"log"
	"net/http"
	"time"

	"github.com/tixigo/tixigo-api/internal/config"
	"github.com/tixigo/tixigo-api/internal/httpapi"
)

func main() {
	cfg := config.Load()
	handler := httpapi.NewRouter(cfg)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Tixigo API listening on :%s (%s)", cfg.Port, cfg.Environment)
	log.Fatal(server.ListenAndServe())
}
