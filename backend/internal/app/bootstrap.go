package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	"tramplin/backend/internal/platform/objectstore"
	postgresplatform "tramplin/backend/internal/platform/postgres"
	httptransport "tramplin/backend/internal/transport/http"
)

type Runtime struct {
	HTTPServer *http.Server
	DB         *pgxpool.Pool
	Objects    objectstore.Store
}

func NewRuntime(cfg config.Config) (*Runtime, error) {
	if err := cfg.ValidateObjectStorage(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()

	db, err := postgresplatform.Open(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var objects objectstore.Store
	if cfg.HasObjectStorageConfig() {
		objects, err = objectstore.NewMinIO(ctx, cfg)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("open object storage: %w", err)
		}
	}

	handler := httptransport.NewServer(cfg, db, httptransport.WithObjectStore(objects))
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
		Objects:    objects,
	}, nil
}

func (r *Runtime) Close() {
	if r.DB != nil {
		r.DB.Close()
	}
}
