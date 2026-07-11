package main

import (
	"fmt"
	"os"

	"insta-downloader/internal/httpserver"
)

func main() {
	server, err := httpserver.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := server.Listen(); err != nil {
		fmt.Fprintf(os.Stderr, "Sunucu başlatılamadı: %v\n", err)
		fmt.Fprintf(os.Stderr, "Port kullanımda olabilir. Çalışan süreci durdurun: fuser -k %s/tcp\n", os.Getenv("PORT"))
		os.Exit(1)
	}
}
