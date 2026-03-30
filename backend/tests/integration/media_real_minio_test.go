//go:build integration && realminio

package integration

import (
	"context"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tramplin/backend/internal/config"
	objectstoreplatform "tramplin/backend/internal/platform/objectstore"
	httptransport "tramplin/backend/internal/transport/http"
)

func TestUploadLifecycleRealMinIOIntegration(t *testing.T) {
	env := newRealMinIOIntegrationEnv(t)
	runUploadLifecycleIntegration(t, env)
}

func TestPublicMediaDownloadAccessRealMinIOIntegration(t *testing.T) {
	env := newRealMinIOIntegrationEnv(t)
	runPublicMediaDownloadAccessIntegration(t, env)
}

func TestPublicMediaDownloadAccessRealMinIODistinctPublicURLIntegration(t *testing.T) {
	t.Setenv("TRAMPLIN_OBJECT_STORAGE_PUBLIC_URL", "http://localhost:9000")

	env := newRealMinIOIntegrationEnv(t)
	runPublicMediaDownloadAccessIntegration(t, env)
}

func newRealMinIOIntegrationEnv(t *testing.T) *integrationEnv {
	t.Helper()

	backendRoot := backendRoot(t)
	loadDotEnvIfPresent(t, filepath.Join(backendRoot, ".env"))

	cfg := config.Load()
	if !cfg.HasObjectStorageConfig() {
		t.Fatal("real MinIO integration test requires object storage configuration")
	}

	ctx := context.Background()

	admin, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}

	schema := uniqueTestSchema()
	if _, err := admin.Exec(ctx, `CREATE SCHEMA `+quoteIdent(schema)); err != nil {
		admin.Close()
		t.Fatalf("create schema %s: %v", schema, err)
	}

	testDB, err := openSchemaPool(ctx, cfg.DatabaseURL, schema)
	if err != nil {
		admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
		t.Fatalf("open schema pool: %v", err)
	}

	applyMigrations(t, testDB, filepath.Join(backendRoot, "db", "migrations"))

	objects, err := objectstoreplatform.NewMinIO(ctx, cfg)
	if err != nil {
		testDB.Close()
		admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
		t.Fatalf("open real MinIO object store: %v", err)
	}

	server := httptest.NewServer(httptransport.NewServer(cfg, testDB, httptransport.WithObjectStore(objects)))
	jar, err := cookiejar.New(nil)
	if err != nil {
		server.Close()
		testDB.Close()
		admin.Exec(ctx, `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
		t.Fatalf("create cookie jar: %v", err)
	}

	client := server.Client()
	client.Jar = jar

	env := &integrationEnv{
		server: server,
		client: client,
		db:     testDB,
		admin:  admin,
		schema: schema,
	}

	t.Cleanup(func() {
		cleanupUploadedObjects(t, testDB, objects)
		server.Close()
		testDB.Close()
		admin.Exec(context.Background(), `DROP SCHEMA IF EXISTS `+quoteIdent(schema)+` CASCADE`)
		admin.Close()
	})

	return env
}

func cleanupUploadedObjects(t *testing.T, db *pgxpool.Pool, objects objectstoreplatform.Store) {
	t.Helper()

	rows, err := db.Query(context.Background(), `SELECT file_key FROM media_files`)
	if err != nil {
		t.Errorf("list uploaded objects for cleanup: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var objectKey string
		if err := rows.Scan(&objectKey); err != nil {
			t.Errorf("scan uploaded object key: %v", err)
			return
		}
		if err := objects.DeleteObject(context.Background(), objectKey); err != nil {
			t.Errorf("delete uploaded object %s: %v", objectKey, err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Errorf("iterate uploaded objects for cleanup: %v", err)
	}
}

func uniqueTestSchema() string {
	return "itest_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}
