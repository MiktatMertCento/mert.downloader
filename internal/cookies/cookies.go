package cookies

import (
	"bufio"
	"os"
	"strings"

	"insta-downloader/internal/domain"
)

func ParseFile(path string) ([]domain.NetscapeCookie, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cookies []domain.NetscapeCookie
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
		cookies = append(cookies, domain.NetscapeCookie{
			Domain: parts[0],
			Name:   parts[5],
			Value:  parts[6],
		})
	}
	return cookies, scanner.Err()
}

func ExtractInstagram(cookies []domain.NetscapeCookie) map[string]string {
	result := make(map[string]string)
	for _, c := range cookies {
		if strings.Contains(c.Domain, "instagram.com") {
			result[c.Name] = c.Value
		}
	}
	return result
}

func BuildHeader(igCookies map[string]string) string {
	var parts []string
	for k, v := range igCookies {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, "; ")
}
