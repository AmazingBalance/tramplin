package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	postgresplatform "tramplin/backend/internal/platform/postgres"
	httptransport "tramplin/backend/internal/transport/http"
)

type Runtime struct {
	HTTPServer *http.Server
	DB         *pgxpool.Pool
}

func NewRuntime(cfg config.Config) (*Runtime, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()

	db, err := postgresplatform.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	handler := httptransport.NewServer(cfg, db)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	server.RegisterOnShutdown(func() {
		db.Close()
	})

	return &Runtime{
		HTTPServer: server,
		DB:         db,
	}, nil
}

func (r *Runtime) Close() {
	if r.DB != nil {
		r.DB.Close()
	}
}
