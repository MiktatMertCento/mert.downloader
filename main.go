package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
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
	Shortcode string
	IsReel    bool
	Platform  string
	VideoID   string
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
	reelPattern     = regexp.MustCompile(`instagram\.com/reels?/([A-Za-z0-9_-]+)`)
	postPattern     = regexp.MustCompile(`instagram\.com/p/([A-Za-z0-9_-]+)`)
	ytWatchPattern  = regexp.MustCompile(`youtube\.com/watch\?v=([A-Za-z0-9_-]+)`)
	ytShortsPattern = regexp.MustCompile(`youtube\.com/shorts/([A-Za-z0-9_-]+)`)
	ytShortPattern  = regexp.MustCompile(`youtu\.be/([A-Za-z0-9_-]+)`)
)

func parseURL(url string) (*ParsedURL, error) {
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
	return nil, fmt.Errorf("desteklenmeyen URL formatı")
}

func shortcodeToMediaID(shortcode string) string {
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

func fetchMediaInfo(shortcode string, igCookies map[string]string) (*MediaInfo, error) {
	mediaID := shortcodeToMediaID(shortcode)
	apiURL := fmt.Sprintf("https://i.instagram.com/api/v1/media/%s/info/", mediaID)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", buildCookieHeader(igCookies))
	req.Header.Set("User-Agent", browserUA)
	req.Header.Set("X-IG-App-ID", igAppID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://www.instagram.com")
	req.Header.Set("Referer", "https://www.instagram.com/")
	if csrf, ok := igCookies["csrftoken"]; ok {
		req.Header.Set("X-CSRFToken", csrf)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API isteği başarısız: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response okunamadı: %w", err)
	}

	if resp.StatusCode != 200 {
		preview := string(body)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, preview)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("JSON parse hatası: %w", err)
	}

	items, ok := raw["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("medya bulunamadı")
	}

	return parseAPIItem(items[0].(map[string]interface{}), shortcode)
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
				cmMap := cm.(map[string]interface{})
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

func getBestImage(item map[string]interface{}) []MediaItem {
	iv2, ok := item["image_versions2"].(map[string]interface{})
	if !ok {
		return nil
	}
	candidates, ok := iv2["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return nil
	}

	best := candidates[0].(map[string]interface{})
	return []MediaItem{{
		Type:   "image",
		URL:    strVal(best, "url"),
		Width:  int(toFloat(best, "width")),
		Height: int(toFloat(best, "height")),
	}}
}

func getBestVideo(item map[string]interface{}) []MediaItem {
	versions, ok := item["video_versions"].([]interface{})
	if !ok || len(versions) == 0 {
		return nil
	}

	best := versions[0].(map[string]interface{})
	return []MediaItem{{
		Type:   "video",
		URL:    strVal(best, "url"),
		Width:  int(toFloat(best, "width")),
		Height: int(toFloat(best, "height")),
	}}
}

func strVal(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func downloadFile(url, destPath string) (int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", browserUA)

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
		"--extractor-args", "youtube:player_client=android",
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
	app.Use(cors.New())
	app.Static("/downloads", "./downloads")
	app.Static("/", "./web/dist")

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

		mediaInfo, apiErr := fetchMediaInfo(parsed.Shortcode, igCookies)
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
				ext := ".jpg"
				if item.Type == "video" {
					ext = ".mp4"
				}
				filename := fmt.Sprintf("%s_%d%s", parsed.Shortcode, i+1, ext)
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

	// SPA fallback
	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendFile("./web/dist/index.html")
	})

	app.Listen(":" + port)
}
