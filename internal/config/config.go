package config

import (
	"os"
	"time"
)

const (
	CookieFile  = "cookies.txt"
	DownloadDir = "downloads"
	BrowserUA   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	IGAppID     = "936619743392459"
	Alphabet    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	DefaultPort = "1905"
)

func Port() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return DefaultPort
}

func CleanupMaxAge() time.Duration {
	return 5 * time.Minute
}
