package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestParseURL_YouTubeWatch(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		videoID string
	}{
		{"basic", "https://www.youtube.com/watch?v=Ma6mYcG4STw", "Ma6mYcG4STw"},
		{"no www", "https://youtube.com/watch?v=Ma6mYcG4STw", "Ma6mYcG4STw"},
		{"with extra params", "https://www.youtube.com/watch?v=dQw4w9WgXcQ&t=10s", "dQw4w9WgXcQ"},
		{"with list param", "https://www.youtube.com/watch?v=abc123_-XYZ&list=PLxyz", "abc123_-XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Platform != "youtube" {
				t.Errorf("platform = %q, want %q", parsed.Platform, "youtube")
			}
			if parsed.VideoID != tt.videoID {
				t.Errorf("videoID = %q, want %q", parsed.VideoID, tt.videoID)
			}
		})
	}
}

func TestParseURL_YouTubeShorts(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		videoID string
	}{
		{"basic", "https://www.youtube.com/shorts/ogGoZuJtG84", "ogGoZuJtG84"},
		{"no www", "https://youtube.com/shorts/ogGoZuJtG84", "ogGoZuJtG84"},
		{"with query", "https://www.youtube.com/shorts/abc123?feature=share", "abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Platform != "youtube" {
				t.Errorf("platform = %q, want %q", parsed.Platform, "youtube")
			}
			if parsed.VideoID != tt.videoID {
				t.Errorf("videoID = %q, want %q", parsed.VideoID, tt.videoID)
			}
		})
	}
}

func TestParseURL_YouTubeShortLink(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		videoID string
	}{
		{"basic", "https://youtu.be/Ma6mYcG4STw", "Ma6mYcG4STw"},
		{"with timestamp", "https://youtu.be/Ma6mYcG4STw?t=42", "Ma6mYcG4STw"},
		{"with feature", "https://youtu.be/abc_-123?feature=share", "abc_-123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Platform != "youtube" {
				t.Errorf("platform = %q, want %q", parsed.Platform, "youtube")
			}
			if parsed.VideoID != tt.videoID {
				t.Errorf("videoID = %q, want %q", parsed.VideoID, tt.videoID)
			}
		})
	}
}

func TestParseURL_InstagramPost(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		shortcode string
	}{
		{"basic", "https://www.instagram.com/p/ABC123xyz/", "ABC123xyz"},
		{"no trailing slash", "https://www.instagram.com/p/ABC123xyz", "ABC123xyz"},
		{"with query", "https://www.instagram.com/p/ABC123xyz/?utm_source=ig", "ABC123xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Platform != "instagram" {
				t.Errorf("platform = %q, want %q", parsed.Platform, "instagram")
			}
			if parsed.Shortcode != tt.shortcode {
				t.Errorf("shortcode = %q, want %q", parsed.Shortcode, tt.shortcode)
			}
			if parsed.IsReel {
				t.Error("isReel = true, want false")
			}
		})
	}
}

func TestParseURL_InstagramReel(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		shortcode string
	}{
		{"reel singular", "https://www.instagram.com/reel/XYZ789abc/", "XYZ789abc"},
		{"reels plural", "https://www.instagram.com/reels/XYZ789abc/", "XYZ789abc"},
		{"no trailing slash", "https://www.instagram.com/reel/XYZ789abc", "XYZ789abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Platform != "instagram" {
				t.Errorf("platform = %q, want %q", parsed.Platform, "instagram")
			}
			if parsed.Shortcode != tt.shortcode {
				t.Errorf("shortcode = %q, want %q", parsed.Shortcode, tt.shortcode)
			}
			if !parsed.IsReel {
				t.Error("isReel = false, want true")
			}
		})
	}
}

func TestParseURL_InvalidURLs(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"empty string", ""},
		{"random text", "not a url at all"},
		{"tiktok url", "https://www.tiktok.com/@user/video/123456"},
		{"youtube playlist", "https://www.youtube.com/playlist?list=PLxyz"},
		{"youtube channel", "https://www.youtube.com/channel/UCabc"},
		{"youtube home", "https://www.youtube.com/"},
		{"instagram stories", "https://www.instagram.com/stories/username/123456"},
		{"instagram profile", "https://www.instagram.com/username/"},
		{"instagram explore", "https://www.instagram.com/explore/"},
		{"plain domain", "https://www.google.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseURL(tt.url)
			if err == nil {
				t.Error("expected error for invalid URL, got nil")
			}
		})
	}
}

func TestParseURL_PlatformDetection(t *testing.T) {
	ytURLs := []string{
		"https://www.youtube.com/watch?v=test123",
		"https://www.youtube.com/shorts/test123",
		"https://youtu.be/test123",
	}

	for _, url := range ytURLs {
		parsed, err := parseURL(url)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", url, err)
		}
		if parsed.Platform != "youtube" {
			t.Errorf("url=%s: platform = %q, want %q", url, parsed.Platform, "youtube")
		}
		if parsed.VideoID != "test123" {
			t.Errorf("url=%s: videoID = %q, want %q", url, parsed.VideoID, "test123")
		}
	}

	igURLs := []struct {
		url       string
		shortcode string
	}{
		{"https://www.instagram.com/p/code123/", "code123"},
		{"https://www.instagram.com/reel/code123/", "code123"},
	}

	for _, tt := range igURLs {
		parsed, err := parseURL(tt.url)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", tt.url, err)
		}
		if parsed.Platform != "instagram" {
			t.Errorf("url=%s: platform = %q, want %q", tt.url, parsed.Platform, "instagram")
		}
		if parsed.Shortcode != tt.shortcode {
			t.Errorf("url=%s: shortcode = %q, want %q", tt.url, parsed.Shortcode, tt.shortcode)
		}
	}
}

func TestAPIDownload_EmptyURL(t *testing.T) {
	app := fiber.New()
	app.Post("/api/download", func(c *fiber.Ctx) error {
		req := new(DownloadRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "Geçersiz istek"})
		}
		if req.URL == "" {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "URL boş"})
		}
		return nil
	})

	body := bytes.NewBufferString(`{"url":""}`)
	req := httptest.NewRequest("POST", "/api/download", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	var result DownloadResponse
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Success {
		t.Error("expected success=false")
	}
}

func TestAPIDownload_InvalidURL(t *testing.T) {
	app := fiber.New()
	app.Post("/api/download", func(c *fiber.Ctx) error {
		req := new(DownloadRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "Geçersiz istek"})
		}
		if req.URL == "" {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "URL boş"})
		}
		parsed, err := parseURL(req.URL)
		if err != nil {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: err.Error()})
		}
		_ = parsed
		return nil
	})

	tests := []struct {
		name string
		url  string
	}{
		{"tiktok", `{"url":"https://www.tiktok.com/@user/video/123"}`},
		{"random", `{"url":"not_a_valid_url"}`},
		{"google", `{"url":"https://www.google.com"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString(tt.url)
			req := httptest.NewRequest("POST", "/api/download", body)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			if resp.StatusCode != 400 {
				t.Errorf("status = %d, want 400", resp.StatusCode)
			}
		})
	}
}

func TestAPIDownload_MissingBody(t *testing.T) {
	app := fiber.New()
	app.Post("/api/download", func(c *fiber.Ctx) error {
		req := new(DownloadRequest)
		if err := c.BodyParser(req); err != nil {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "Geçersiz istek"})
		}
		if req.URL == "" {
			return c.Status(400).JSON(DownloadResponse{Success: false, Error: "URL boş"})
		}
		return nil
	})

	req := httptest.NewRequest("POST", "/api/download", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
