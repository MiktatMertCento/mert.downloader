package mediaurl_test

import (
	"testing"

	"insta-downloader/internal/mediaurl"
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
			parsed, err := mediaurl.Parse(tt.url)
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
			parsed, err := mediaurl.Parse(tt.url)
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
			parsed, err := mediaurl.Parse(tt.url)
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
			parsed, err := mediaurl.Parse(tt.url)
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
			parsed, err := mediaurl.Parse(tt.url)
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

func TestParseURL_InstagramHighlight(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		highlightID string
	}{
		{"basic", "https://www.instagram.com/stories/highlights/17849176446661385/", "17849176446661385"},
		{"no trailing slash", "https://www.instagram.com/stories/highlights/17849176446661385", "17849176446661385"},
		{"with query", "https://www.instagram.com/stories/highlights/17849176446661385/?utm_source=ig", "17849176446661385"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := mediaurl.Parse(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !parsed.IsHighlight {
				t.Fatal("isHighlight = false, want true")
			}
			if parsed.HighlightID != tt.highlightID {
				t.Errorf("highlightID = %q, want %q", parsed.HighlightID, tt.highlightID)
			}
		})
	}
}

func TestParseURL_InstagramProfile(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		username string
	}{
		{"basic", "https://www.instagram.com/miktatmertcento/", "miktatmertcento"},
		{"no trailing slash", "https://www.instagram.com/miktatmertcento", "miktatmertcento"},
		{"with query", "https://www.instagram.com/test.user/?hl=tr", "test.user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := mediaurl.Parse(tt.url)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !parsed.IsProfile {
				t.Fatal("isProfile = false, want true")
			}
			if parsed.Username != tt.username {
				t.Errorf("username = %q, want %q", parsed.Username, tt.username)
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
		{"basic", "https://www.instagram.com/stories/miktatmertcento/", "miktatmertcento", ""},
		{"no trailing slash", "https://www.instagram.com/stories/miktatmertcento", "miktatmertcento", ""},
		{"with story id", "https://www.instagram.com/stories/miktatmertcento/1234567890/", "miktatmertcento", "1234567890"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := mediaurl.Parse(tt.url)
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

func TestNormalizeShortcode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"DLUWShcs4b6", "DLUWShcs4b6"},
		{"DaqUlZUCOrtLLY6V5V895aXAoI35s3Im9RjhLY0", "DaqUlZUCOrt"},
		{"CCQQsCXjOaB", "CCQQsCXjOaB"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mediaurl.NormalizeShortcode(tt.input)
			if got != tt.want {
				t.Errorf("mediaurl.NormalizeShortcode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortcodeToMediaID(t *testing.T) {
	tests := []struct {
		shortcode string
		want      string
	}{
		{"DLUWShcs4b6", "3662650426847889146"},
		{"B8KjT5QHq1x", "2236755463714549105"},
		{"DaqUlZUCOrtLLY6V5V895aXAoI35s3Im9RjhLY0", "3939051354819455725"},
	}

	for _, tt := range tests {
		t.Run(tt.shortcode, func(t *testing.T) {
			got := mediaurl.ShortcodeToMediaID(tt.shortcode)
			if got != tt.want {
				t.Errorf("mediaurl.ShortcodeToMediaID(%q) = %q, want %q", tt.shortcode, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilenamePart(t *testing.T) {
	if got := mediaurl.SanitizeFilename("G'"); got != "G" {
		t.Errorf("mediaurl.SanitizeFilename(\"G'\") = %q, want G", got)
	}
	if got := mediaurl.SanitizeFilename(":)"); got != "highlight" {
		t.Errorf("mediaurl.SanitizeFilename(\":)\") = %q, want highlight", got)
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
		{"instagram explore", "https://www.instagram.com/explore/"},
		{"plain domain", "https://www.google.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mediaurl.Parse(tt.url)
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
		parsed, err := mediaurl.Parse(url)
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
		parsed, err := mediaurl.Parse(tt.url)
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
