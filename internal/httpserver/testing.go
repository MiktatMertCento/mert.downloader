package httpserver

import (
	"insta-downloader/internal/downloader"
	"insta-downloader/internal/upscale"

	"github.com/gofiber/fiber/v2"
)

// NewTestServer builds a server without loading cookies — for handler unit tests.
func NewTestServer(dl *downloader.Service, up *upscale.Manager) *Server {
	if up == nil {
		up = upscale.NewManager("downloads")
	}
	s := &Server{
		app:      fiber.New(),
		download: dl,
		upscale:  up,
		userID:   "test-user",
	}
	s.routes()
	return s
}
