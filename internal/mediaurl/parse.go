package mediaurl

import (
	"fmt"
	"math/big"
	"regexp"
	"strings"

	"insta-downloader/internal/config"
	"insta-downloader/internal/domain"
)

var (
	storyPattern     = regexp.MustCompile(`instagram\.com/stories/([A-Za-z0-9._]+)(?:/(\d+))?`)
	highlightPattern = regexp.MustCompile(`instagram\.com/stories/highlights/(\d+)`)
	reelPattern      = regexp.MustCompile(`instagram\.com/reels?/([A-Za-z0-9_-]+)`)
	postPattern      = regexp.MustCompile(`instagram\.com/p/([A-Za-z0-9_-]+)`)
	profilePattern   = regexp.MustCompile(`instagram\.com/([A-Za-z0-9._]+)(?:/)?(?:\?|#|$)`)
	ytWatchPattern   = regexp.MustCompile(`youtube\.com/watch\?v=([A-Za-z0-9_-]+)`)
	ytShortsPattern  = regexp.MustCompile(`youtube\.com/shorts/([A-Za-z0-9_-]+)`)
	ytShortPattern   = regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]+)`)
	reservedIGPaths  = map[string]struct{}{
		"p": {}, "reel": {}, "reels": {}, "stories": {}, "explore": {}, "accounts": {},
		"direct": {}, "tv": {}, "legal": {}, "about": {}, "developer": {}, "privacy": {},
		"terms": {}, "session": {}, "login": {}, "directory": {}, "api": {}, "graphql": {},
		"challenge": {}, "username": {}, "oauth": {}, "help": {}, "emails": {}, "locations": {},
		"tags": {}, "nametag": {}, "archive": {}, "web": {}, "static": {},
	}
	filenameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

func Parse(url string) (*domain.ParsedURL, error) {
	if m := highlightPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{
			HighlightID: m[1],
			IsHighlight: true,
			Platform:    "instagram",
		}, nil
	}
	if m := storyPattern.FindStringSubmatch(url); len(m) > 1 {
		parsed := &domain.ParsedURL{
			Username: m[1],
			IsStory:  true,
			Platform: "instagram",
		}
		if len(m) > 2 && m[2] != "" {
			parsed.StoryID = m[2]
		}
		return parsed, nil
	}
	if m := reelPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{Shortcode: m[1], IsReel: true, Platform: "instagram"}, nil
	}
	if m := postPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{Shortcode: m[1], IsReel: false, Platform: "instagram"}, nil
	}
	if m := ytWatchPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := ytShortsPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := ytShortPattern.FindStringSubmatch(url); len(m) > 1 {
		return &domain.ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := profilePattern.FindStringSubmatch(url); len(m) > 1 {
		username := m[1]
		if _, reserved := reservedIGPaths[strings.ToLower(username)]; !reserved {
			return &domain.ParsedURL{Username: username, IsProfile: true, Platform: "instagram"}, nil
		}
	}
	return nil, fmt.Errorf("desteklenmeyen URL formatı")
}

func NormalizeShortcode(shortcode string) string {
	if len(shortcode) > 11 {
		return shortcode[:11]
	}
	return shortcode
}

func ShortcodeToMediaID(shortcode string) string {
	shortcode = NormalizeShortcode(shortcode)
	id := big.NewInt(0)
	for _, c := range shortcode {
		idx := strings.IndexRune(config.Alphabet, c)
		if idx < 0 {
			idx = 0
		}
		id.Mul(id, big.NewInt(64))
		id.Add(id, big.NewInt(int64(idx)))
	}
	return id.String()
}

func SanitizeFilename(value string) string {
	cleaned := filenameSanitizer.ReplaceAllString(strings.TrimSpace(value), "_")
	cleaned = strings.Trim(cleaned, "._-")
	if cleaned == "" {
		return "highlight"
	}
	return cleaned
}
