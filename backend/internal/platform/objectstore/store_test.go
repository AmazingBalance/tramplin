package objectstore

import (
	"testing"

	"tramplin/backend/internal/config"
)

func TestResolvePresignEndpoint(t *testing.T) {
	t.Run("falls back to internal endpoint when public url is empty", func(t *testing.T) {
		endpoint, secure := resolvePresignEndpoint(config.Config{
			ObjectStorageEndpoint: "minio:9000",
			ObjectStorageUseSSL:   false,
		})

		if endpoint != "minio:9000" || secure {
			t.Fatalf("unexpected presign endpoint: got %q secure=%v", endpoint, secure)
		}
	})

	t.Run("uses explicit public url host and scheme", func(t *testing.T) {
		endpoint, secure := resolvePresignEndpoint(config.Config{
			ObjectStorageEndpoint:  "minio:9000",
			ObjectStorageUseSSL:    false,
			ObjectStoragePublicURL: "https://cdn.example.test:9443",
		})

		if endpoint != "cdn.example.test:9443" || !secure {
			t.Fatalf("unexpected presign endpoint: got %q secure=%v", endpoint, secure)
		}
	})

	t.Run("uses host-only public endpoint with explicit ssl flag", func(t *testing.T) {
		endpoint, secure := resolvePresignEndpoint(config.Config{
			ObjectStorageEndpoint:   "minio:9000",
			ObjectStorageUseSSL:     false,
			ObjectStoragePublicURL:  "localhost:9100",
			ObjectStoragePublicSSL:  false,
		})

		if endpoint != "localhost:9100" || secure {
			t.Fatalf("unexpected presign endpoint: got %q secure=%v", endpoint, secure)
		}
	})
}
