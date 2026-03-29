package main

import (
	"context"
	"log"
	"time"

	"tramplin/backend/db/migrations"
	"tramplin/backend/internal/config"
	postgresplatform "tramplin/backend/internal/platform/postgres"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := postgresplatform.Open(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := migrations.Apply(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Print("tramplin migrations applied")
}
