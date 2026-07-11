package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"insta-downloader/internal/config"
	"insta-downloader/internal/cookies"
	"insta-downloader/internal/domain"
	"insta-downloader/internal/mediaurl"
)

func MediaInfoEndpoints(mediaID string) []string {
	return []string{
		fmt.Sprintf("https://www.instagram.com/api/v1/media/%s/info/", mediaID),
		fmt.Sprintf("https://i.instagram.com/api/v1/media/%s/info/", mediaID),
	}
}

func FetchMediaInfo(shortcode string, referer string, igCookies map[string]string) (*domain.MediaInfo, error) {
	mediaID := mediaurl.ShortcodeToMediaID(shortcode)
	if referer == "" {
		referer = fmt.Sprintf("https://www.instagram.com/p/%s/", shortcode)
	}

	var lastErr error
	for _, apiURL := range MediaInfoEndpoints(mediaID) {
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

		return ParseAPIItem(first, shortcode)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("medya bilgisi alınamadı")
	}
	return nil, lastErr
}

func ParseAPIItem(item map[string]interface{}, shortcode string) (*domain.MediaInfo, error) {
	info := &domain.MediaInfo{Shortcode: shortcode}

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
		info.Items = append(info.Items, GetBestImage(item)...)
	case 2:
		info.MediaType = "video"
		info.Items = append(info.Items, GetBestVideo(item)...)
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
					info.Items = append(info.Items, GetBestVideo(cmMap)...)
				} else {
					info.Items = append(info.Items, GetBestImage(cmMap)...)
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

func pickBestVersion(versions []interface{}, mediaType string) []domain.MediaItem {
	var best domain.MediaItem
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
	return []domain.MediaItem{best}
}

func GetBestImage(item map[string]interface{}) []domain.MediaItem {
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

func GetBestVideo(item map[string]interface{}) []domain.MediaItem {
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
	req.Header.Set("Cookie", cookies.BuildHeader(igCookies))
	req.Header.Set("User-Agent", config.BrowserUA)
	req.Header.Set("X-IG-App-ID", config.IGAppID)
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

func mediaItemFromVersion(version map[string]interface{}, mediaType string) domain.MediaItem {
	return domain.MediaItem{
		Type:   mediaType,
		URL:    strVal(version, "url"),
		Width:  int(toFloat(version, "width")),
		Height: int(toFloat(version, "height")),
	}
}

func bestMediaFromCoverMedia(coverMedia map[string]interface{}) (domain.MediaItem, bool) {
	if full, ok := coverMedia["full_image_version"].(map[string]interface{}); ok {
		if item := mediaItemFromVersion(full, "image"); item.URL != "" {
			return item, true
		}
	}

	if images := GetBestImage(coverMedia); len(images) > 0 && images[0].URL != "" {
		return images[0], true
	}

	if videos := GetBestVideo(coverMedia); len(videos) > 0 && videos[0].URL != "" {
		return videos[0], true
	}

	if cropped, ok := coverMedia["cropped_image_version"].(map[string]interface{}); ok {
		if item := mediaItemFromVersion(cropped, "image"); item.URL != "" {
			return item, true
		}
	}

	return domain.MediaItem{}, false
}

func ParseHighlightCover(highlight map[string]interface{}) (domain.HighlightCover, error) {
	title := strVal(highlight, "title")
	if title == "" {
		title = "highlight"
	}

	id := strVal(highlight, "id")
	if id == "" {
		id = strVal(highlight, "strong_id__")
	}
	if id == "" {
		return domain.HighlightCover{}, fmt.Errorf("öne çıkan kimliği bulunamadı")
	}

	coverMedia, ok := highlight["cover_media"].(map[string]interface{})
	if !ok || coverMedia == nil {
		return domain.HighlightCover{}, fmt.Errorf("öne çıkan kapağı bulunamadı: %s", title)
	}

	item, ok := bestMediaFromCoverMedia(coverMedia)
	if !ok {
		return domain.HighlightCover{}, fmt.Errorf("öne çıkan kapağı bulunamadı: %s", title)
	}

	return domain.HighlightCover{Title: title, ID: id, Item: item}, nil
}

func itemArea(item domain.MediaItem) int {
	if item.Width > 0 && item.Height > 0 {
		return item.Width * item.Height
	}
	return 0
}

func EnsureHighlightReelID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if strings.HasPrefix(id, "highlight:") {
		return id
	}
	return "highlight:" + id
}

func HighlightNumericID(id string) string {
	return strings.TrimPrefix(EnsureHighlightReelID(id), "highlight:")
}

func MediaFileExt(item domain.MediaItem) string {
	ext := filepath.Ext(strings.Split(item.URL, "?")[0])
	if ext != "" {
		return ext
	}
	if item.Type == "video" {
		return ".mp4"
	}
	return ".jpg"
}

func FetchHighlightReels(highlightIDs []string, referer string, igCookies map[string]string) (map[string]map[string]interface{}, error) {
	if len(highlightIDs) == 0 {
		return map[string]map[string]interface{}{}, nil
	}

	quoted := make([]string, len(highlightIDs))
	for i, id := range highlightIDs {
		quoted[i] = fmt.Sprintf(`"%s"`, EnsureHighlightReelID(id))
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
			result[EnsureHighlightReelID(key)] = reel
		}
	}
	return result, nil
}

func FetchUserHighlights(username string, igCookies map[string]string) ([]domain.HighlightCover, error) {
	userID, err := FetchUserID(username, igCookies)
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

	covers := make([]domain.HighlightCover, 0, len(tray))
	for _, entry := range tray {
		highlight, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		cover, err := ParseHighlightCover(highlight)
		if err != nil {
			continue
		}
		cover.ID = EnsureHighlightReelID(cover.ID)
		covers = append(covers, cover)
	}

	if len(covers) == 0 {
		return nil, fmt.Errorf("öne çıkan kapağı bulunamadı")
	}

	return covers, nil
}

func FetchHighlightStories(highlightID string, igCookies map[string]string) (string, string, []domain.MediaItem, error) {
	reelKey := EnsureHighlightReelID(highlightID)
	numericID := HighlightNumericID(highlightID)
	referer := fmt.Sprintf("https://www.instagram.com/stories/highlights/%s/", numericID)

	reels, err := FetchHighlightReels([]string{reelKey}, referer, igCookies)
	if err != nil {
		return "", "", nil, fmt.Errorf("öne çıkan içerikleri alınamadı: %w", err)
	}

	reel, ok := reels[reelKey]
	if !ok || reel == nil {
		return "", "", nil, fmt.Errorf("öne çıkan bulunamadı")
	}

	items, err := ParseStoryItems(reel, "")
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

func FetchUserID(username string, igCookies map[string]string) (string, error) {
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

func ParseStoryItems(reel map[string]interface{}, storyID string) ([]domain.MediaItem, error) {
	itemsRaw, ok := reel["items"].([]interface{})
	if !ok || len(itemsRaw) == 0 {
		return nil, fmt.Errorf("story bulunamadı veya süresi dolmuş")
	}

	var items []domain.MediaItem
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
			items = append(items, GetBestVideo(item)...)
		case 8:
			if carousel, ok := item["carousel_media"].([]interface{}); ok {
				for _, cm := range carousel {
					cmMap, ok := cm.(map[string]interface{})
					if !ok {
						continue
					}
					cmType := int(toFloat(cmMap, "media_type"))
					if cmType == 2 {
						items = append(items, GetBestVideo(cmMap)...)
					} else {
						items = append(items, GetBestImage(cmMap)...)
					}
				}
			} else {
				items = append(items, GetBestImage(item)...)
			}
		default:
			items = append(items, GetBestImage(item)...)
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

func FetchUserStories(username, storyID string, igCookies map[string]string) ([]domain.MediaItem, error) {
	userID, err := FetchUserID(username, igCookies)
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

	return ParseStoryItems(reel, storyID)
}
