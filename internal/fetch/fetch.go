package fetch

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"insta-downloader/internal/config"
)

func DownloadFile(mediaURL, destPath string) (int64, error) {
	req, err := http.NewRequest("GET", mediaURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", config.BrowserUA)
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

func CopyToTemp(src string) (string, error) {
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

func WithYTDLP(videoURL, outputDir, id string, useCookies bool) (string, error) {
	outputPath := filepath.Join(outputDir, id+".mp4")

	args := []string{
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/bestvideo+bestaudio/best",
		"--merge-output-format", "mp4",
		"-o", outputPath,
		"--no-playlist",
		"--js-runtimes", "node",
	}

	if useCookies {
		tmpCookies, err := CopyToTemp(config.CookieFile)
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

func CleanupLoop(dir string, maxAge time.Duration) {
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
