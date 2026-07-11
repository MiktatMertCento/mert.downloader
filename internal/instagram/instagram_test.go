package instagram_test

import (
	"strings"
	"testing"

	"insta-downloader/internal/instagram"
)

func TestMediaInfoEndpoints(t *testing.T) {
	endpoints := instagram.MediaInfoEndpoints("12345")
	if len(endpoints) != 2 {
		t.Fatalf("endpoints = %d, want 2", len(endpoints))
	}
	if !strings.HasPrefix(endpoints[0], "https://www.instagram.com/api/v1/media/") {
		t.Errorf("primary endpoint = %q, want www.instagram.com", endpoints[0])
	}
	if !strings.HasSuffix(endpoints[0], "/12345/info/") {
		t.Errorf("primary endpoint = %q", endpoints[0])
	}
	if !strings.HasPrefix(endpoints[1], "https://i.instagram.com/api/v1/media/") {
		t.Errorf("fallback endpoint = %q, want i.instagram.com", endpoints[1])
	}
}

func TestParseAPIItem(t *testing.T) {
	t.Run("image post", func(t *testing.T) {
		item := map[string]interface{}{
			"media_type": float64(1),
			"user":       map[string]interface{}{"username": "private_user"},
			"caption":    map[string]interface{}{"text": "test caption"},
			"image_versions2": map[string]interface{}{
				"candidates": []interface{}{
					map[string]interface{}{
						"url":    "https://cdn.example/image.jpg",
						"width":  float64(1080),
						"height": float64(1350),
					},
				},
			},
		}

		info, err := instagram.ParseAPIItem(item, "ABC123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.MediaType != "image" {
			t.Errorf("mediaType = %q, want image", info.MediaType)
		}
		if info.Username != "private_user" {
			t.Errorf("username = %q, want private_user", info.Username)
		}
		if len(info.Items) != 1 || info.Items[0].Type != "image" {
			t.Errorf("unexpected items: %+v", info.Items)
		}
	})

	t.Run("carousel post", func(t *testing.T) {
		item := map[string]interface{}{
			"media_type": float64(8),
			"carousel_media": []interface{}{
				map[string]interface{}{
					"media_type": float64(1),
					"image_versions2": map[string]interface{}{
						"candidates": []interface{}{
							map[string]interface{}{"url": "https://cdn.example/1.jpg", "width": float64(1080), "height": float64(1080)},
						},
					},
				},
				map[string]interface{}{
					"media_type": float64(2),
					"video_versions": []interface{}{
						map[string]interface{}{"url": "https://cdn.example/2.mp4", "width": float64(720), "height": float64(1280)},
					},
				},
			},
		}

		info, err := instagram.ParseAPIItem(item, "CAR123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if info.MediaType != "carousel" {
			t.Errorf("mediaType = %q, want carousel", info.MediaType)
		}
		if len(info.Items) != 2 {
			t.Fatalf("items = %d, want 2", len(info.Items))
		}
	})
}

func TestParseHighlightCover(t *testing.T) {
	highlight := map[string]interface{}{
		"id":    "highlight:17849176446661385",
		"title": "G'",
		"cover_media": map[string]interface{}{
			"full_image_version": map[string]interface{}{
				"url":    "https://cdn.example/full.jpg",
				"width":  float64(1080),
				"height": float64(1920),
			},
			"cropped_image_version": map[string]interface{}{
				"url":    "https://cdn.example/thumb.jpg",
				"width":  float64(150),
				"height": float64(150),
			},
		},
	}

	cover, err := instagram.ParseHighlightCover(highlight)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cover.Title != "G'" {
		t.Errorf("title = %q, want G'", cover.Title)
	}
	if cover.Item.URL != "https://cdn.example/full.jpg" {
		t.Errorf("url = %q, want full image", cover.Item.URL)
	}
	if cover.Item.Width != 1080 {
		t.Errorf("width = %d, want 1080", cover.Item.Width)
	}
}

func TestEnsureHighlightReelID(t *testing.T) {
	if got := instagram.EnsureHighlightReelID("17849176446661385"); got != "highlight:17849176446661385" {
		t.Errorf("got %q", got)
	}
	if got := instagram.EnsureHighlightReelID("highlight:17849176446661385"); got != "highlight:17849176446661385" {
		t.Errorf("got %q", got)
	}
	if got := instagram.HighlightNumericID("highlight:17849176446661385"); got != "17849176446661385" {
		t.Errorf("got %q", got)
	}
}

func TestGetBestImagePicksHighestResolution(t *testing.T) {
	item := map[string]interface{}{
		"image_versions2": map[string]interface{}{
			"candidates": []interface{}{
				map[string]interface{}{"url": "https://cdn.example/small.jpg", "width": float64(150), "height": float64(150)},
				map[string]interface{}{"url": "https://cdn.example/large.jpg", "width": float64(1290), "height": float64(2293)},
				map[string]interface{}{"url": "https://cdn.example/medium.jpg", "width": float64(640), "height": float64(1136)},
			},
		},
	}

	got := instagram.GetBestImage(item)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].URL != "https://cdn.example/large.jpg" {
		t.Errorf("url = %q, want large.jpg", got[0].URL)
	}
	if got[0].Width != 1290 || got[0].Height != 2293 {
		t.Errorf("size = %dx%d, want 1290x2293", got[0].Width, got[0].Height)
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
		items, err := instagram.ParseStoryItems(reel, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("items = %d, want 2", len(items))
		}
	})

	t.Run("single story by id", func(t *testing.T) {
		items, err := instagram.ParseStoryItems(reel, "3935990187228994715")
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
		_, err := instagram.ParseStoryItems(map[string]interface{}{"items": []interface{}{}}, "")
		if err == nil {
			t.Fatal("expected error for empty reel")
		}
	})
}
