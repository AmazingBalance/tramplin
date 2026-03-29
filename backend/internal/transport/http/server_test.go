package http

import (
	"testing"

	"tramplin/backend/internal/config"
)

func TestExternalObjectURL(t *testing.T) {
	t.Run("returns original when public endpoint is not configured", func(t *testing.T) {
		server := &Server{cfg: config.Config{}}

		got := server.externalObjectURL("http://minio:9000/tramplin-media/object?x=1")
		want := "http://minio:9000/tramplin-media/object?x=1"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("rewrites to explicit public url", func(t *testing.T) {
		server := &Server{cfg: config.Config{
			ObjectStoragePublicURL: "http://127.0.0.1:9100",
		}}

		got := server.externalObjectURL("http://minio:9000/tramplin-media/object?x=1")
		want := "http://127.0.0.1:9100/tramplin-media/object?x=1"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("rewrites to host only public endpoint", func(t *testing.T) {
		server := &Server{cfg: config.Config{
			ObjectStoragePublicURL: "cdn.example.test:9443",
			ObjectStoragePublicSSL: true,
		}}

		got := server.externalObjectURL("http://minio:9000/tramplin-media/object?x=1")
		want := "https://cdn.example.test:9443/tramplin-media/object?x=1"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}
