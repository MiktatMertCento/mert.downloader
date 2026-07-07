package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestParseURL_InstagramStory(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		username string
		storyID  string
	}{
		{"basic", "https://www.instagram.com/stories/haleee_.m/", "haleee_.m", ""},
		{"no trailing slash", "https://www.instagram.com/stories/haleee_.m", "haleee_.m", ""},
		{"with story id", "https://www.instagram.com/stories/haleee_.m/1234567890/", "haleee_.m", "1234567890"},
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
			if !parsed.IsStory {
				t.Error("isStory = false, want true")
			}
			if parsed.Username != tt.username {
				t.Errorf("username = %q, want %q", parsed.Username, tt.username)
			}
			if parsed.StoryID != tt.storyID {
				t.Errorf("storyID = %q, want %q", parsed.StoryID, tt.storyID)
			}
		})
	}
}

func TestParseStoryItems(t *testing.T) {
	reel := map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{
				"pk":         "3935990187228994714",
				"media_type": float64(1),
				"image_versions2": map[string]interface{}{
					"candidates": []interface{}{
						map[string]interface{}{
							"url":    "https://cdn.example/741495341_18453157378136140_8825083769244252955_n.webp",
							"width":  float64(1440),
							"height": float64(2560),
						},
					},
				},
			},
			map[string]interface{}{
				"pk":         "3935990187228994715",
				"media_type": float64(1),
				"image_versions2": map[string]interface{}{
					"candidates": []interface{}{
						map[string]interface{}{
							"url":    "https://cdn.example/733641915_18453157555136140_5876843432736789290_n.webp",
							"width":  float64(1440),
							"height": float64(2560),
						},
					},
				},
			},
		},
	}

	t.Run("all stories", func(t *testing.T) {
		items, err := parseStoryItems(reel, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("items = %d, want 2", len(items))
		}
	})

	t.Run("single story by id", func(t *testing.T) {
		items, err := parseStoryItems(reel, "3935990187228994715")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 1 {
			t.Fatalf("items = %d, want 1", len(items))
		}
		if !strings.Contains(items[0].URL, "733641915") {
			t.Errorf("unexpected url: %s", items[0].URL)
		}
	})

	t.Run("empty reel", func(t *testing.T) {
		_, err := parseStoryItems(map[string]interface{}{"items": []interface{}{}}, "")
		if err == nil {
			t.Fatal("expected error for empty reel")
		}
	})
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

func TestDownloadsByteRange(t *testing.T) {
	dir := t.TempDir()
	content := []byte("0123456789abcdef")
	if err := os.WriteFile(filepath.Join(dir, "sample.mp4"), content, 0644); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	app.Static("/downloads", dir, fiber.Static{ByteRange: true})

	req := httptest.NewRequest("GET", "/downloads/sample.mp4", nil)
	req.Header.Set("Range", "bytes=0-4")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 206 {
		t.Fatalf("status = %d, want 206", resp.StatusCode)
	}
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		t.Errorf("Accept-Ranges = %q, want bytes", resp.Header.Get("Accept-Ranges"))
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "01234" {
		t.Errorf("body = %q, want %q", body, "01234")
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
