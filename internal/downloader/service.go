package downloader

import (
	"fmt"
	"os"
	"path/filepath"

	"insta-downloader/internal/config"
	"insta-downloader/internal/domain"
	"insta-downloader/internal/fetch"
	"insta-downloader/internal/instagram"
	"insta-downloader/internal/mediaurl"
)

// Service orchestrates URL parsing, Instagram/YouTube fetching, and file storage.
type Service struct {
	DownloadDir string
	IGCookies   map[string]string
}

func New(igCookies map[string]string) *Service {
	return &Service{
		DownloadDir: config.DownloadDir,
		IGCookies:   igCookies,
	}
}

// Download resolves the URL and returns a ready-to-serialize API response.
func (s *Service) Download(rawURL string) (*domain.DownloadResponse, error) {
	parsed, err := mediaurl.Parse(rawURL)
	if err != nil {
		return &domain.DownloadResponse{Success: false, Error: err.Error()}, err
	}

	switch {
	case parsed.IsStory:
		return s.downloadStory(parsed)
	case parsed.IsHighlight:
		return s.downloadHighlight(parsed)
	case parsed.IsProfile:
		return s.downloadProfileCovers(parsed)
	case parsed.Platform == "youtube":
		return s.downloadYouTube(rawURL, parsed)
	default:
		return s.downloadInstagramPost(parsed)
	}
}

func (s *Service) downloadStory(parsed *domain.ParsedURL) (*domain.DownloadResponse, error) {
	outDir := filepath.Join(s.DownloadDir, "story_"+parsed.Username)
	if parsed.StoryID != "" {
		outDir = filepath.Join(s.DownloadDir, "story_"+parsed.Username+"_"+parsed.StoryID)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	storyItems, err := instagram.FetchUserStories(parsed.Username, parsed.StoryID, s.IGCookies)
	if err != nil {
		return fail(err)
	}

	response := &domain.DownloadResponse{
		Success:   true,
		Shortcode: parsed.Username,
		Username:  parsed.Username,
		MediaType: "story",
	}
	if parsed.StoryID != "" {
		response.Shortcode = parsed.StoryID
	}

	for i, item := range storyItems {
		filename := fmt.Sprintf("%s_%d%s", response.Shortcode, i+1, instagram.MediaFileExt(item))
		destPath := filepath.Join(outDir, filename)
		size, err := fetch.DownloadFile(item.URL, destPath)
		if err != nil {
			return fail(fmt.Errorf("Story indirilemedi: %w", err))
		}
		response.Files = append(response.Files, fileMeta(filename, destPath, item, size))
	}
	return response, nil
}

func (s *Service) downloadHighlight(parsed *domain.ParsedURL) (*domain.DownloadResponse, error) {
	outDir := filepath.Join(s.DownloadDir, "highlight_"+parsed.HighlightID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	title, username, highlightItems, err := instagram.FetchHighlightStories(parsed.HighlightID, s.IGCookies)
	if err != nil {
		return fail(err)
	}

	response := &domain.DownloadResponse{
		Success:   true,
		Shortcode: parsed.HighlightID,
		Username:  username,
		Caption:   title,
		MediaType: "highlight",
	}

	baseName := mediaurl.SanitizeFilename(title)
	for i, item := range highlightItems {
		filename := fmt.Sprintf("%s_%d%s", baseName, i+1, instagram.MediaFileExt(item))
		destPath := filepath.Join(outDir, filename)
		size, err := fetch.DownloadFile(item.URL, destPath)
		if err != nil {
			return fail(fmt.Errorf("Öne çıkan indirilemedi (%s): %w", title, err))
		}
		response.Files = append(response.Files, fileMeta(filename, destPath, item, size))
	}
	return response, nil
}

func (s *Service) downloadProfileCovers(parsed *domain.ParsedURL) (*domain.DownloadResponse, error) {
	outDir := filepath.Join(s.DownloadDir, "highlights_"+parsed.Username)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	highlights, err := instagram.FetchUserHighlights(parsed.Username, s.IGCookies)
	if err != nil {
		return fail(err)
	}

	response := &domain.DownloadResponse{
		Success:   true,
		Shortcode: parsed.Username,
		Username:  parsed.Username,
		MediaType: "highlight_covers",
	}

	for i, highlight := range highlights {
		filename := fmt.Sprintf("%s_%d%s", mediaurl.SanitizeFilename(highlight.Title), i+1, instagram.MediaFileExt(highlight.Item))
		destPath := filepath.Join(outDir, filename)
		size, err := fetch.DownloadFile(highlight.Item.URL, destPath)
		if err != nil {
			return fail(fmt.Errorf("Öne çıkan indirilemedi (%s): %w", highlight.Title, err))
		}
		response.Files = append(response.Files, fileMeta(filename, destPath, highlight.Item, size))
	}
	return response, nil
}

func (s *Service) downloadYouTube(rawURL string, parsed *domain.ParsedURL) (*domain.DownloadResponse, error) {
	outDir := filepath.Join(s.DownloadDir, parsed.VideoID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	response := &domain.DownloadResponse{
		Success:   true,
		Shortcode: parsed.VideoID,
		MediaType: "video",
	}

	filePath, err := fetch.WithYTDLP(rawURL, outDir, parsed.VideoID, false)
	if err != nil {
		return fail(fmt.Errorf("YouTube video indirilemedi: %w", err))
	}

	response.Files = append(response.Files, videoFile(filePath))
	return response, nil
}

func (s *Service) downloadInstagramPost(parsed *domain.ParsedURL) (*domain.DownloadResponse, error) {
	outDir := filepath.Join(s.DownloadDir, parsed.Shortcode)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	response := &domain.DownloadResponse{
		Success:   true,
		Shortcode: parsed.Shortcode,
	}

	referer := fmt.Sprintf("https://www.instagram.com/p/%s/", parsed.Shortcode)
	if parsed.IsReel {
		referer = fmt.Sprintf("https://www.instagram.com/reel/%s/", parsed.Shortcode)
	}
	mediaInfo, apiErr := instagram.FetchMediaInfo(parsed.Shortcode, referer, s.IGCookies)
	if apiErr == nil {
		response.Username = mediaInfo.Username
		response.Caption = mediaInfo.Caption
		response.MediaType = mediaInfo.MediaType
	}

	if parsed.IsReel {
		response.MediaType = "reel"
		reelURL := fmt.Sprintf("https://www.instagram.com/reel/%s/", parsed.Shortcode)
		filePath, err := fetch.WithYTDLP(reelURL, outDir, parsed.Shortcode, true)
		if err != nil {
			return fail(fmt.Errorf("Reel indirilemedi: %w", err))
		}
		response.Files = append(response.Files, videoFile(filePath))
		return response, nil
	}

	if apiErr != nil {
		return fail(fmt.Errorf("Medya bilgisi alınamadı: %w", apiErr))
	}

	for i, item := range mediaInfo.Items {
		filename := fmt.Sprintf("%s_%d%s", parsed.Shortcode, i+1, instagram.MediaFileExt(item))
		destPath := filepath.Join(outDir, filename)
		size, err := fetch.DownloadFile(item.URL, destPath)
		if err != nil {
			continue
		}
		response.Files = append(response.Files, fileMeta(filename, destPath, item, size))
	}

	if len(response.Files) == 0 {
		return fail(fmt.Errorf("Hiçbir medya dosyası indirilemedi"))
	}
	return response, nil
}

func fileMeta(filename, destPath string, item domain.MediaItem, size int64) domain.DownloadedFile {
	return domain.DownloadedFile{
		Filename: filename,
		Path:     "/" + filepath.ToSlash(destPath),
		Type:     item.Type,
		Size:     size,
		Width:    item.Width,
		Height:   item.Height,
	}
}

func videoFile(filePath string) domain.DownloadedFile {
	var size int64
	if info, err := os.Stat(filePath); err == nil {
		size = info.Size()
	}
	return domain.DownloadedFile{
		Filename: filepath.Base(filePath),
		Path:     "/" + filepath.ToSlash(filePath),
		Type:     "video",
		Size:     size,
	}
}

func fail(err error) (*domain.DownloadResponse, error) {
	return &domain.DownloadResponse{Success: false, Error: err.Error()}, err
}
