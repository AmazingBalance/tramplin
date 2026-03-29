package main

import (
	"log"
	"net/http"

	"tramplin/backend/internal/app"
	"tramplin/backend/internal/config"
)

func main() {
	cfg := config.Load()
	runtime, err := app.NewRuntime(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()

	log.Printf("tramplin backend listening on %s", cfg.HTTPAddr)

	if err := runtime.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
