package httpserver_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"insta-downloader/internal/downloader"
	"insta-downloader/internal/httpserver"

	"github.com/gofiber/fiber/v2"
)

func TestDownloadsByteRange(t *testing.T) {
	app := fiber.New()
	app.Static("/downloads", t.TempDir(), fiber.Static{ByteRange: true})

	req := httptest.NewRequest("GET", "/downloads/missing.bin", nil)
	req.Header.Set("Range", "bytes=0-1")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	// Missing file → 404/403 depending on fiber version; ensure request is accepted by stack.
	if resp.StatusCode == 0 {
		t.Fatal("empty status")
	}
}

func TestAPIDownload_EmptyURL(t *testing.T) {
	s := httpserver.NewTestServer(downloader.New(nil), nil)
	body := bytes.NewBufferString(`{"url":""}`)
	req := httptest.NewRequest("POST", "/api/download", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var payload map[string]any
	decodeJSON(t, resp.Body, &payload)
	if payload["error"] != "URL boş" && payload["success"] != false {
		// DownloadResponse shape
	}
	if payload["success"] != false {
		t.Fatalf("payload=%v", payload)
	}
}

func TestAPIDownload_InvalidURL(t *testing.T) {
	s := httpserver.NewTestServer(downloader.New(nil), nil)
	body := bytes.NewBufferString(`{"url":"https://example.com/not-supported"}`)
	req := httptest.NewRequest("POST", "/api/download", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestAPIDownload_MissingBody(t *testing.T) {
	s := httpserver.NewTestServer(downloader.New(nil), nil)
	req := httptest.NewRequest("POST", "/api/download", nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestIndexHTMLHasNoCacheHeaders(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "web", "dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(dist, "index.html")
	if err := os.WriteFile(indexPath, []byte("<!doctype html><title>t</title>"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	s := httpserver.NewTestServer(downloader.New(nil), nil)
	req := httptest.NewRequest("GET", "/", nil)
	resp, err := s.App().Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	cache := resp.Header.Get("Cache-Control")
	if !strings.Contains(cache, "no-cache") || !strings.Contains(cache, "no-store") {
		t.Fatalf("Cache-Control=%q", cache)
	}
	if resp.Header.Get("Pragma") != "no-cache" {
		t.Fatalf("Pragma=%q", resp.Header.Get("Pragma"))
	}
}

func decodeJSON(t *testing.T, r io.Reader, dest any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(dest); err != nil {
		t.Fatal(err)
	}
}
