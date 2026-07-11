package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

const (
	cookieFile  = "cookies.txt"
	downloadDir = "downloads"
	browserUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	igAppID     = "936619743392459"
	alphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)

type NetscapeCookie struct {
	Domain string
	Name   string
	Value  string
}

type ParsedURL struct {
	Shortcode  string
	IsReel     bool
	IsStory      bool
	IsProfile    bool
	IsHighlight  bool
	Username     string
	StoryID      string
	HighlightID  string
	Platform   string
	VideoID    string
}

type HighlightCover struct {
	Title string
	ID    string
	Item  MediaItem
}

type MediaItem struct {
	URL    string `json:"url"`
	Type   string `json:"type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type MediaInfo struct {
	Shortcode string      `json:"shortcode"`
	MediaType string      `json:"media_type"`
	Username  string      `json:"username"`
	Caption   string      `json:"caption"`
	Items     []MediaItem `json:"items"`
}

type DownloadRequest struct {
	URL string `json:"url"`
}

type DownloadedFile struct {
	Filename string `json:"filename"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
}

type DownloadResponse struct {
	Success   bool             `json:"success"`
	Shortcode string           `json:"shortcode"`
	MediaType string           `json:"media_type"`
	Username  string           `json:"username"`
	Caption   string           `json:"caption,omitempty"`
	Files     []DownloadedFile `json:"files"`
	Error     string           `json:"error,omitempty"`
}

func parseCookieFile(path string) ([]NetscapeCookie, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cookies []NetscapeCookie
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 7 {
			continue
		}
		cookies = append(cookies, NetscapeCookie{
			Domain: parts[0],
			Name:   parts[5],
			Value:  parts[6],
		})
	}
	return cookies, scanner.Err()
}

func extractInstagramCookies(cookies []NetscapeCookie) map[string]string {
	result := make(map[string]string)
	for _, c := range cookies {
		if strings.Contains(c.Domain, "instagram.com") {
			result[c.Name] = c.Value
		}
	}
	return result
}

func buildCookieHeader(igCookies map[string]string) string {
	var parts []string
	for k, v := range igCookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}

var (
	storyPattern      = regexp.MustCompile(`instagram\.com/stories/([A-Za-z0-9._]+)(?:/(\d+))?`)
	highlightPattern  = regexp.MustCompile(`instagram\.com/stories/highlights/(\d+)`)
	reelPattern     = regexp.MustCompile(`instagram\.com/reels?/([A-Za-z0-9_-]+)`)
	postPattern     = regexp.MustCompile(`instagram\.com/p/([A-Za-z0-9_-]+)`)
	profilePattern  = regexp.MustCompile(`instagram\.com/([A-Za-z0-9._]+)(?:/)?(?:\?|#|$)`)
	ytWatchPattern  = regexp.MustCompile(`youtube\.com/watch\?v=([A-Za-z0-9_-]+)`)
	ytShortsPattern = regexp.MustCompile(`youtube\.com/shorts/([A-Za-z0-9_-]+)`)
	ytShortPattern  = regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]+)`)
	reservedIGPaths = map[string]struct{}{
		"p": {}, "reel": {}, "reels": {}, "stories": {}, "explore": {}, "accounts": {},
		"direct": {}, "tv": {}, "legal": {}, "about": {}, "developer": {}, "privacy": {},
		"terms": {}, "session": {}, "login": {}, "directory": {}, "api": {}, "graphql": {},
		"challenge": {}, "username": {}, "oauth": {}, "help": {}, "emails": {}, "locations": {},
		"tags": {}, "nametag": {}, "archive": {}, "web": {}, "static": {},
	}
	filenameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
)

func parseURL(url string) (*ParsedURL, error) {
	if m := highlightPattern.FindStringSubmatch(url); len(m) > 1 {
		return &ParsedURL{
			HighlightID: m[1],
			IsHighlight: true,
			Platform:    "instagram",
		}, nil
	}
	if m := storyPattern.FindStringSubmatch(url); len(m) > 1 {
		parsed := &ParsedURL{
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
		return &ParsedURL{Shortcode: m[1], IsReel: true, Platform: "instagram"}, nil
	}
	if m := postPattern.FindStringSubmatch(url); len(m) > 1 {
		return &ParsedURL{Shortcode: m[1], IsReel: false, Platform: "instagram"}, nil
	}
	if m := ytWatchPattern.FindStringSubmatch(url); len(m) > 1 {
		return &ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := ytShortsPattern.FindStringSubmatch(url); len(m) > 1 {
		return &ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := ytShortPattern.FindStringSubmatch(url); len(m) > 1 {
		return &ParsedURL{VideoID: m[1], Platform: "youtube"}, nil
	}
	if m := profilePattern.FindStringSubmatch(url); len(m) > 1 {
		username := m[1]
		if _, reserved := reservedIGPaths[strings.ToLower(username)]; !reserved {
			return &ParsedURL{Username: username, IsProfile: true, Platform: "instagram"}, nil
		}
	}
	return nil, fmt.Errorf("desteklenmeyen URL formatı")
}

func normalizeShortcode(shortcode string) string {
	if len(shortcode) > 11 {
		return shortcode[:11]
	}
	return shortcode
}

func shortcodeToMediaID(shortcode string) string {
	shortcode = normalizeShortcode(shortcode)
	id := big.NewInt(0)
	for _, c := range shortcode {
		idx := strings.IndexRune(alphabet, c)
		if idx < 0 {
			idx = 0
		}
		id.Mul(id, big.NewInt(64))
		id.Add(id, big.NewInt(int64(idx)))
	}
	return id.String()
}

func mediaInfoEndpoints(mediaID string) []string {
	return []string{
		fmt.Sprintf("https://www.instagram.com/api/v1/media/%s/info/", mediaID),
		fmt.Sprintf("https://i.instagram.com/api/v1/media/%s/info/", mediaID),
	}
}

func fetchMediaInfo(shortcode string, referer string, igCookies map[string]string) (*MediaInfo, error) {
	mediaID := shortcodeToMediaID(shortcode)
	if referer == "" {
		referer = fmt.Sprintf("https://www.instagram.com/p/%s/", shortcode)
	}

	var lastErr error
	for _, apiURL := range mediaInfoEndpoints(mediaID) {
		body, err := instagramAPIRequest("GET", apiURL, referer, igCookies)
		if err != nil {
			lastErr = fmt.Errorf("API isteği başarısız: %w", err)
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			lastErr = fmt.Errorf("JSON parse hatası: %w", err)
			continue
		}

		items, ok := raw["items"].([]interface{})
		if !ok || len(items) == 0 {
			lastErr = fmt.Errorf("medya bulunamadı")
			continue
		}

		first, ok := items[0].(map[string]interface{})
		if !ok {
			lastErr = fmt.Errorf("medya yanıtı geçersiz")
			continue
		}

		return parseAPIItem(first, shortcode)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("medya bilgisi alınamadı")
	}
	return nil, lastErr
}

func parseAPIItem(item map[string]interface{}, shortcode string) (*MediaInfo, error) {
	info := &MediaInfo{Shortcode: shortcode}

	if user, ok := item["user"].(map[string]interface{}); ok {
		if u, ok := user["username"].(string); ok {
			info.Username = u
		}
	}

	if caption, ok := item["caption"].(map[string]interface{}); ok {
		if text, ok := caption["text"].(string); ok {
			info.Caption = text
		}
	}

	mediaType := int(toFloat(item, "media_type"))

	switch mediaType {
	case 1:
		info.MediaType = "image"
		info.Items = append(info.Items, getBestImage(item)...)
	case 2:
		info.MediaType = "video"
		info.Items = append(info.Items, getBestVideo(item)...)
	case 8:
		info.MediaType = "carousel"
		if carousel, ok := item["carousel_media"].([]interface{}); ok {
			for _, cm := range carousel {
				cmMap, ok := cm.(map[string]interface{})
				if !ok {
					continue
				}
				cmType := int(toFloat(cmMap, "media_type"))
				if cmType == 2 {
					info.Items = append(info.Items, getBestVideo(cmMap)...)
				} else {
					info.Items = append(info.Items, getBestImage(cmMap)...)
				}
			}
		}
	}

	return info, nil
}

func toFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func pickBestVersion(versions []interface{}, mediaType string) []MediaItem {
	var best MediaItem
	bestArea := -1

	for _, raw := range versions {
		version, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		item := mediaItemFromVersion(version, mediaType)
		if item.URL == "" {
			continue
		}
		area := itemArea(item)
		if area > bestArea {
			best = item
			bestArea = area
		}
	}

	if best.URL == "" {
		return nil
	}
	return []MediaItem{best}
}

func getBestImage(item map[string]interface{}) []MediaItem {
	iv2, ok := item["image_versions2"].(map[string]interface{})
	if !ok {
		return nil
	}
	candidates, ok := iv2["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return nil
	}
	return pickBestVersion(candidates, "image")
}

func getBestVideo(item map[string]interface{}) []MediaItem {
	versions, ok := item["video_versions"].([]interface{})
	if !ok || len(versions) == 0 {
		return nil
	}
	return pickBestVersion(versions, "video")
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func stringifyID(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return fmt.Sprintf("%.0f", t)
	case json.Number:
		return t.String()
	default:
		return ""
	}
}

func setInstagramAPIHeaders(req *http.Request, referer string, igCookies map[string]string) {
	req.Header.Set("Cookie", buildCookieHeader(igCookies))
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("X-IG-App-ID", igAppID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://www.instagram.com")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	if csrf, ok := igCookies["csrftoken"]; ok {
		req.Header.Set("X-CSRFToken", csrf)
	}
}

func doInstagramAPIRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		preview := string(body)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, preview)
	}

	return body, nil
}

func instagramAPIRequest(method, apiURL, referer string, igCookies map[string]string) ([]byte, error) {
	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, err
	}
	setInstagramAPIHeaders(req, referer, igCookies)
	return doInstagramAPIRequest(req)
}

func instagramAPIPost(apiURL, referer string, formData url.Values, igCookies map[string]string) ([]byte, error) {
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	setInstagramAPIHeaders(req, referer, igCookies)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return doInstagramAPIRequest(req)
}

func mediaItemFromVersion(version map[string]interface{}, mediaType string) MediaItem {
	return MediaItem{
		Type:   mediaType,
		URL:    strVal(version, "url"),
		Width:  int(toFloat(version, "width")),
		Height: int(toFloat(version, "height")),
	}
}

func bestMediaFromCoverMedia(coverMedia map[string]interface{}) (MediaItem, bool) {
	if full, ok := coverMedia["full_image_version"].(map[string]interface{}); ok {
		if item := mediaItemFromVersion(full, "image"); item.URL != "" {
			return item, true
		}
	}

	if images := getBestImage(coverMedia); len(images) > 0 && images[0].URL != "" {
		return images[0], true
	}

	if videos := getBestVideo(coverMedia); len(videos) > 0 && videos[0].URL != "" {
		return videos[0], true
	}

	if cropped, ok := coverMedia["cropped_image_version"].(map[string]interface{}); ok {
		if item := mediaItemFromVersion(cropped, "image"); item.URL != "" {
			return item, true
		}
	}

	return MediaItem{}, false
}

func parseHighlightCover(highlight map[string]interface{}) (HighlightCover, error) {
	title := strVal(highlight, "title")
	if title == "" {
		title = "highlight"
	}

	id := strVal(highlight, "id")
	if id == "" {
		id = strVal(highlight, "strong_id__")
	}
	if id == "" {
		return HighlightCover{}, fmt.Errorf("öne çıkan kimliği bulunamadı")
	}

	coverMedia, ok := highlight["cover_media"].(map[string]interface{})
	if !ok || coverMedia == nil {
		return HighlightCover{}, fmt.Errorf("öne çıkan kapağı bulunamadı: %s", title)
	}

	item, ok := bestMediaFromCoverMedia(coverMedia)
	if !ok {
		return HighlightCover{}, fmt.Errorf("öne çıkan kapağı bulunamadı: %s", title)
	}

	return HighlightCover{Title: title, ID: id, Item: item}, nil
}

func itemArea(item MediaItem) int {
	if item.Width > 0 && item.Height > 0 {
		return item.Width * item.Height
	}
	return 0
}

func ensureHighlightReelID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "highlight:") {
		return id
	}
	return "highlight:" + id
}

func highlightNumericID(id string) string {
	return strings.TrimPrefix(ensureHighlightReelID(id), "highlight:")
}

func sanitizeFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	value = filenameSanitizer.ReplaceAllString(value, "_")
	value = strings.Trim(value, "._-")
	if value == "" {
		return "highlight"
	}
	return value
}

func mediaFileExt(item MediaItem) string {
	ext := filepath.Ext(strings.Split(item.URL, "?")[0])
	if ext != "" {
		return ext
	}
	if item.Type == "video" {
		return ".mp4"
	}
	return ".jpg"
}

func fetchHighlightReels(highlightIDs []string, referer string, igCookies map[string]string) (map[string]map[string]interface{}, error) {
	if len(highlightIDs) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	quoted := make([]string, len(highlightIDs))
	for i, id := range highlightIDs {
		quoted[i] = fmt.Sprintf(`"%s"`, ensureHighlightReelID(id))
	}

	form := url.Values{}
	form.Set("reel_ids", fmt.Sprintf("[%s]", strings.Join(quoted, ",")))

	body, err := instagramAPIPost(
		"https://www.instagram.com/api/v1/feed/reels_media/",
		referer,
		form,
		igCookies,
	)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("öne çıkan detayları okunamadı: %w", err)
	}

	reelsRaw, ok := raw["reels"].(map[string]interface{})
	if !ok {
		return map[string]map[string]interface{}{}, nil
	}

	result := make(map[string]map[string]interface{}, len(reelsRaw))
	for key, value := range reelsRaw {
		reel, ok := value.(map[string]interface{})
		if ok {
			result[ensureHighlightReelID(key)] = reel
		}
	}
	return result, nil
}

func fetchUserHighlights(username string, igCookies map[string]string) ([]HighlightCover, error) {
	userID, err := fetchInstagramUserID(username, igCookies)
	if err != nil {
		return nil, err
	}

	referer := fmt.Sprintf("https://www.instagram.com/%s/", username)
	apiURL := fmt.Sprintf("https://www.instagram.com/api/v1/highlights/%s/highlights_tray/", userID)
	body, err := instagramAPIRequest("GET", apiURL, referer, igCookies)
	if err != nil {
		return nil, fmt.Errorf("öne çıkanlar alınamadı: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("öne çıkan yanıtı okunamadı: %w", err)
	}

	tray, ok := raw["tray"].([]interface{})
	if !ok || len(tray) == 0 {
		return nil, fmt.Errorf("öne çıkan bulunamadı")
	}

	covers := make([]HighlightCover, 0, len(tray))
	for _, entry := range tray {
		highlight, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		cover, err := parseHighlightCover(highlight)
		if err != nil {
			continue
		}
		cover.ID = ensureHighlightReelID(cover.ID)
		covers = append(covers, cover)
	}

	if len(covers) == 0 {
		return nil, fmt.Errorf("öne çıkan kapağı bulunamadı")
	}

	return covers, nil
}

func fetchHighlightStories(highlightID string, igCookies map[string]string) (string, string, []MediaItem, error) {
	reelKey := ensureHighlightReelID(highlightID)
	numericID := highlightNumericID(highlightID)
	referer := fmt.Sprintf("https://www.instagram.com/stories/highlights/%s/", numericID)

	reels, err := fetchHighlightReels([]string{reelKey}, referer, igCookies)
	if err != nil {
		return "", "", nil, fmt.Errorf("öne çıkan içerikleri alınamadı: %w", err)
	}

	reel, ok := reels[reelKey]
	if !ok || reel == nil {
		return "", "", nil, fmt.Errorf("öne çıkan bulunamadı")
	}

	items, err := parseStoryItems(reel, "")
	if err != nil {
		return "", "", nil, err
	}

	title := strVal(reel, "title")
	if title == "" {
		title = numericID
	}

	username := ""
	if user, ok := reel["user"].(map[string]interface{}); ok {
		username = strVal(user, "username")
	}

	return title, username, items, nil
}

func fetchInstagramUserID(username string, igCookies map[string]string) (string, error) {
	apiURL := fmt.Sprintf(
		"https://www.instagram.com/web/search/topsearch/?query=%s",
		url.QueryEscape(username),
	)

	body, err := instagramAPIRequest(
		"GET",
		apiURL,
		fmt.Sprintf("https://www.instagram.com/stories/%s/", username),
		igCookies,
	)
	if err != nil {
		return "", fmt.Errorf("kullanıcı aranamadı: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("kullanıcı yanıtı okunamadı: %w", err)
	}

	users, ok := raw["users"].([]interface{})
	if !ok {
		return "", fmt.Errorf("kullanıcı bulunamadı: %s", username)
	}

	target := strings.ToLower(username)
	for _, entry := range users {
		userMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		user, ok := userMap["user"].(map[string]interface{})
		if !ok {
			continue
		}
		if strings.ToLower(strVal(user, "username")) != target {
			continue
		}
		if id := stringifyID(user["pk"]); id != "" {
			return id, nil
		}
		if id := stringifyID(user["id"]); id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("kullanıcı bulunamadı: %s", username)
}

func storyItemPK(item map[string]interface{}) string {
	if pk := stringifyID(item["pk"]); pk != "" {
		return pk
	}
	id := strVal(item, "id")
	if idx := strings.Index(id, "_"); idx > 0 {
		return id[:idx]
	}
	return id
}

func parseStoryItems(reel map[string]interface{}, storyID string) ([]MediaItem, error) {
	itemsRaw, ok := reel["items"].([]interface{})
	if !ok || len(itemsRaw) == 0 {
		return nil, fmt.Errorf("story bulunamadı veya süresi dolmuş")
	}

	var items []MediaItem
	for _, raw := range itemsRaw {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		if storyID != "" {
			pk := storyItemPK(item)
			if pk != storyID && !strings.HasPrefix(strVal(item, "id"), storyID) {
				continue
			}
		}

		mediaType := int(toFloat(item, "media_type"))
		switch mediaType {
		case 2:
			items = append(items, getBestVideo(item)...)
		case 8:
			if carousel, ok := item["carousel_media"].([]interface{}); ok {
				for _, cm := range carousel {
					cmMap, ok := cm.(map[string]interface{})
					if !ok {
						continue
					}
					cmType := int(toFloat(cmMap, "media_type"))
					if cmType == 2 {
						items = append(items, getBestVideo(cmMap)...)
					} else {
						items = append(items, getBestImage(cmMap)...)
					}
				}
			} else {
				items = append(items, getBestImage(item)...)
			}
		default:
			items = append(items, getBestImage(item)...)
		}
	}

	if len(items) == 0 {
		if storyID != "" {
			return nil, fmt.Errorf("story bulunamadı: %s", storyID)
		}
		return nil, fmt.Errorf("story bulunamadı veya süresi dolmuş")
	}

	return items, nil
}

func fetchUserStories(username, storyID string, igCookies map[string]string) ([]MediaItem, error) {
	userID, err := fetchInstagramUserID(username, igCookies)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://www.instagram.com/api/v1/feed/user/%s/story/", userID)
	body, err := instagramAPIRequest(
		"GET",
		apiURL,
		fmt.Sprintf("https://www.instagram.com/stories/%s/", username),
		igCookies,
	)
	if err != nil {
		return nil, fmt.Errorf("story bilgisi alınamadı: %w", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("story yanıtı okunamadı: %w", err)
	}

	reel, ok := raw["reel"].(map[string]interface{})
	if !ok || reel == nil {
		return nil, fmt.Errorf("story bulunamadı veya süresi dolmuş")
	}

	return parseStoryItems(reel, storyID)
}

func downloadFile(mediaURL, destPath string) (int64, error) {
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("Origin", "https://www.instagram.com")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	return io.Copy(out, resp.Body)
}

func copyToTemp(src string) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp("", "cookies-*.txt")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func downloadWithYTDLP(videoURL, outputDir, id string, useCookies bool) (string, error) {
	outputPath := filepath.Join(outputDir, id+".mp4")

	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best",
		"--merge-output-format", "mp4",
		"-o", outputPath,
		"--no-playlist",
		"--js-runtimes", "node",
	}

	if useCookies {
		tmpCookies, err := copyToTemp(cookieFile)
		if err != nil {
			return "", fmt.Errorf("cookie kopyalanamadı: %w", err)
		}
		defer os.Remove(tmpCookies)
		args = append([]string{"--cookies", tmpCookies}, args...)
	}

	args = append(args, videoURL)

	cmd := exec.Command("yt-dlp", args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("yt-dlp: %s", stderr.String())
	}

	entries, _ := os.ReadDir(outputDir)
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(name, ".mp4") {
			return filepath.Join(outputDir, name), nil
		}
	}

	return "", fmt.Errorf("indirilen dosya bulunamadı")
}

func cleanupDownloads(dir string, maxAge time.Duration) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if time.Since(info.ModTime()) > maxAge {
				path := filepath.Join(dir, entry.Name())
				os.RemoveAll(path)
				fmt.Printf("Temizlendi: %s\n", path)
			}
		}
	}
}

func main() {
	allCookies, err := parseCookieFile(cookieFile)
	if err != nil {
		fmt.Printf("Cookie dosyası okunamadı: %v\n", err)
		os.Exit(1)
	}

	igCookies := extractInstagramCookies(allCookies)
	if igCookies["sessionid"] == "" {
		fmt.Println("Instagram sessionid bulunamadı")
		os.Exit(1)
	}

	fmt.Printf("Instagram cookies yüklendi (user: %s)\n", igCookies["ds_user_id"])

	os.MkdirAll(downloadDir, 0755)

	go cleanupDownloads(downloadDir, 5*time.Minute)

	app := fiber.New(fiber.Config{BodyLimit: 10 * 1024 * 1024})
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		ExposeHeaders: "Content-Length, Content-Range, Accept-Ranges",
	}))
	app.Static("/downloads", "./downloads", fiber.Static{
		ByteRange: true,
	})

	app.Get("/api/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"user_id": igCookies["ds_user_id"],
		})
	})

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

		if parsed.IsStory {
			outDir := filepath.Join(downloadDir, "story_"+parsed.Username)
			if parsed.StoryID != "" {
				outDir = filepath.Join(downloadDir, "story_"+parsed.Username+"_"+parsed.StoryID)
			}
			os.MkdirAll(outDir, 0755)

			storyItems, err := fetchUserStories(parsed.Username, parsed.StoryID, igCookies)
			if err != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: err.Error(),
				})
			}

			response := DownloadResponse{
				Success:   true,
				Shortcode: parsed.Username,
				Username:  parsed.Username,
				MediaType: "story",
			}
			if parsed.StoryID != "" {
				response.Shortcode = parsed.StoryID
			}

			for i, item := range storyItems {
				filename := fmt.Sprintf("%s_%d%s", response.Shortcode, i+1, mediaFileExt(item))
				destPath := filepath.Join(outDir, filename)

				size, err := downloadFile(item.URL, destPath)
				if err != nil {
					return c.Status(500).JSON(DownloadResponse{
						Success: false, Error: fmt.Sprintf("Story indirilemedi: %v", err),
					})
				}

				response.Files = append(response.Files, DownloadedFile{
					Filename: filename,
					Path:     "/" + filepath.ToSlash(destPath),
					Type:     item.Type,
					Size:     size,
					Width:    item.Width,
					Height:   item.Height,
				})
			}

			return c.JSON(response)
		}

		if parsed.IsHighlight {
			outDir := filepath.Join(downloadDir, "highlight_"+parsed.HighlightID)
			os.MkdirAll(outDir, 0755)

			title, username, highlightItems, err := fetchHighlightStories(parsed.HighlightID, igCookies)
			if err != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: err.Error(),
				})
			}

			response := DownloadResponse{
				Success:   true,
				Shortcode: parsed.HighlightID,
				Username:  username,
				Caption:   title,
				MediaType: "highlight",
			}

			baseName := sanitizeFilenamePart(title)
			for i, item := range highlightItems {
				filename := fmt.Sprintf("%s_%d%s", baseName, i+1, mediaFileExt(item))
				destPath := filepath.Join(outDir, filename)

				size, err := downloadFile(item.URL, destPath)
				if err != nil {
					return c.Status(500).JSON(DownloadResponse{
						Success: false, Error: fmt.Sprintf("Öne çıkan indirilemedi (%s): %v", title, err),
					})
				}

				response.Files = append(response.Files, DownloadedFile{
					Filename: filename,
					Path:     "/" + filepath.ToSlash(destPath),
					Type:     item.Type,
					Size:     size,
					Width:    item.Width,
					Height:   item.Height,
				})
			}

			return c.JSON(response)
		}

		if parsed.IsProfile {
			outDir := filepath.Join(downloadDir, "highlights_"+parsed.Username)
			os.MkdirAll(outDir, 0755)

			highlights, err := fetchUserHighlights(parsed.Username, igCookies)
			if err != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: err.Error(),
				})
			}

			response := DownloadResponse{
				Success:   true,
				Shortcode: parsed.Username,
				Username:  parsed.Username,
				MediaType: "highlight_covers",
			}

			for i, highlight := range highlights {
				filename := fmt.Sprintf("%s_%d%s", sanitizeFilenamePart(highlight.Title), i+1, mediaFileExt(highlight.Item))
				destPath := filepath.Join(outDir, filename)

				size, err := downloadFile(highlight.Item.URL, destPath)
				if err != nil {
					return c.Status(500).JSON(DownloadResponse{
						Success: false, Error: fmt.Sprintf("Öne çıkan indirilemedi (%s): %v", highlight.Title, err),
					})
				}

				response.Files = append(response.Files, DownloadedFile{
					Filename: filename,
					Path:     "/" + filepath.ToSlash(destPath),
					Type:     highlight.Item.Type,
					Size:     size,
					Width:    highlight.Item.Width,
					Height:   highlight.Item.Height,
				})
			}

			return c.JSON(response)
		}

		if parsed.Platform == "youtube" {
			outDir := filepath.Join(downloadDir, parsed.VideoID)
			os.MkdirAll(outDir, 0755)

			response := DownloadResponse{
				Success:   true,
				Shortcode: parsed.VideoID,
				MediaType: "video",
			}

			filePath, err := downloadWithYTDLP(req.URL, outDir, parsed.VideoID, false)
			if err != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: fmt.Sprintf("YouTube video indirilemedi: %v", err),
				})
			}

			finfo, _ := os.Stat(filePath)
			var size int64
			if finfo != nil {
				size = finfo.Size()
			}

			response.Files = append(response.Files, DownloadedFile{
				Filename: filepath.Base(filePath),
				Path:     "/" + filepath.ToSlash(filePath),
				Type:     "video",
				Size:     size,
			})

			return c.JSON(response)
		}

		outDir := filepath.Join(downloadDir, parsed.Shortcode)
		os.MkdirAll(outDir, 0755)

		response := DownloadResponse{
			Success:   true,
			Shortcode: parsed.Shortcode,
		}

		referer := fmt.Sprintf("https://www.instagram.com/p/%s/", parsed.Shortcode)
		if parsed.IsReel {
			referer = fmt.Sprintf("https://www.instagram.com/reel/%s/", parsed.Shortcode)
		}
		mediaInfo, apiErr := fetchMediaInfo(parsed.Shortcode, referer, igCookies)
		if apiErr == nil {
			response.Username = mediaInfo.Username
			response.Caption = mediaInfo.Caption
			response.MediaType = mediaInfo.MediaType
		}

		if parsed.IsReel {
			response.MediaType = "reel"

			reelURL := fmt.Sprintf("https://www.instagram.com/reel/%s/", parsed.Shortcode)
			filePath, err := downloadWithYTDLP(reelURL, outDir, parsed.Shortcode, true)
			if err != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: fmt.Sprintf("Reel indirilemedi: %v", err),
				})
			}

			finfo, _ := os.Stat(filePath)
			var size int64
			if finfo != nil {
				size = finfo.Size()
			}

			response.Files = append(response.Files, DownloadedFile{
				Filename: filepath.Base(filePath),
				Path:     "/" + filepath.ToSlash(filePath),
				Type:     "video",
				Size:     size,
			})
		} else {
			if apiErr != nil {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: fmt.Sprintf("Medya bilgisi alınamadı: %v", apiErr),
				})
			}

			for i, item := range mediaInfo.Items {
				filename := fmt.Sprintf("%s_%d%s", parsed.Shortcode, i+1, mediaFileExt(item))
				destPath := filepath.Join(outDir, filename)

				size, err := downloadFile(item.URL, destPath)
				if err != nil {
					continue
				}

				response.Files = append(response.Files, DownloadedFile{
					Filename: filename,
					Path:     "/" + filepath.ToSlash(destPath),
					Type:     item.Type,
					Size:     size,
					Width:    item.Width,
					Height:   item.Height,
				})
			}

			if len(response.Files) == 0 {
				return c.Status(500).JSON(DownloadResponse{
					Success: false, Error: "Hiçbir medya dosyası indirilemedi",
				})
			}
		}

		return c.JSON(response)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "1905"
	}

	fmt.Printf("Sunucu :%s portunda başlatılıyor\n", port)
	fmt.Println("POST /api/download  {\"url\": \"...\"}")
	fmt.Println("GET  /api/health")
	fmt.Println("GET  /downloads/...")

	app.Get("/manifest.webmanifest", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/manifest+json")
		return c.SendFile("./web/dist/manifest.webmanifest")
	})

	app.Static("/", "./web/dist", fiber.Static{
		Index: "index.html",
	})

	if err := app.Listen(":" + port); err != nil {
		fmt.Fprintf(os.Stderr, "Sunucu başlatılamadı: %v\n", err)
		fmt.Fprintf(os.Stderr, "Port %s kullanımda olabilir. Çalışan süreci durdurun: fuser -k %s/tcp\n", port, port)
		os.Exit(1)
	}
}
