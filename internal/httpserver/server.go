package httpserver

import (
	"fmt"
	"os"
	"strings"

	"insta-downloader/internal/config"
	"insta-downloader/internal/cookies"
	"insta-downloader/internal/domain"
	"insta-downloader/internal/downloader"
	"insta-downloader/internal/fetch"
	"insta-downloader/internal/upscale"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// Server is the HTTP edge for the downloader API and SPA.
type Server struct {
	app      *fiber.App
	download *downloader.Service
	upscale  *upscale.Manager
	userID   string
}

// New boots dependencies from disk/env and wires routes.
func New() (*Server, error) {
	allCookies, err := cookies.ParseFile(config.CookieFile)
	if err != nil {
		return nil, fmt.Errorf("cookie dosyası okunamadı: %w", err)
	}

	igCookies := cookies.ExtractInstagram(allCookies)
	if igCookies["sessionid"] == "" {
		return nil, fmt.Errorf("instagram sessionid bulunamadı")
	}

	fmt.Printf("Instagram cookies yüklendi (user: %s)\n", igCookies["ds_user_id"])

	if err := os.MkdirAll(config.DownloadDir, 0o755); err != nil {
		return nil, err
	}
	go fetch.CleanupLoop(config.DownloadDir, config.CleanupMaxAge())

	s := &Server{
		app:      fiber.New(fiber.Config{BodyLimit: 10 * 1024 * 1024}),
		download: downloader.New(igCookies),
		upscale:  upscale.NewManager(config.DownloadDir),
		userID:   igCookies["ds_user_id"],
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.app.Use(logger.New())
	s.app.Use(cors.New(cors.Config{
		ExposeHeaders: "Content-Length, Content-Range, Accept-Ranges",
	}))
	s.app.Static("/downloads", "./"+config.DownloadDir, fiber.Static{ByteRange: true})

	s.app.Get("/api/health", s.handleHealth)
	s.app.Post("/api/upscale", s.handleUpscaleStart)
	s.app.Get("/api/upscale/:id", s.handleUpscaleStatus)
	s.app.Post("/api/download", s.handleDownload)

	s.app.Get("/manifest.webmanifest", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/manifest+json")
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
		return c.SendFile("./web/dist/manifest.webmanifest")
	})
	s.app.Get("/sw.js", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
		c.Set("Service-Worker-Allowed", "/")
		return c.SendFile("./web/dist/sw.js")
	})
	s.app.Get("/registerSW.js", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/javascript; charset=utf-8")
		c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
		return c.SendFile("./web/dist/registerSW.js")
	})

	// SPA shell must never be cached — hashed assets under /assets can be immutable.
	s.app.Get("/", s.serveIndexHTML)
	s.app.Get("/index.html", s.serveIndexHTML)
	s.app.Use(func(c *fiber.Ctx) error {
		path := c.Path()
		if strings.HasPrefix(path, "/assets/") {
			c.Set(fiber.HeaderCacheControl, "public, max-age=31536000, immutable")
		}
		return c.Next()
	})
	s.app.Static("/", "./web/dist", fiber.Static{
		Index:  "",
		MaxAge: 31536000,
	})
	// Client-side routes (no file extension) fall through to index.html
	s.app.Get("/*", func(c *fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") || strings.HasPrefix(c.Path(), "/downloads") {
			return fiber.ErrNotFound
		}
		if strings.Contains(c.Path(), ".") {
			return fiber.ErrNotFound
		}
		return s.serveIndexHTML(c)
	})
}

func (s *Server) serveIndexHTML(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")
	c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
	return c.SendFile("./web/dist/index.html")
}

func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":        "ok",
		"user_id":       s.userID,
		"upscale_ready": s.upscale.Available() == nil,
	})
}

func (s *Server) handleUpscaleStart(c *fiber.Ctx) error {
	req := new(upscale.UpscaleRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek"})
	}
	if strings.TrimSpace(req.Path) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "path boş"})
	}
	job, err := s.upscale.Start(req.Path)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusAccepted).JSON(job)
}

func (s *Server) handleUpscaleStatus(c *fiber.Ctx) error {
	job, ok := s.upscale.Get(c.Params("id"))
	if !ok {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "job bulunamadı"})
	}
	return c.JSON(job)
}

func (s *Server) handleDownload(c *fiber.Ctx) error {
	req := new(domain.DownloadRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.DownloadResponse{Success: false, Error: "Geçersiz istek"})
	}
	if strings.TrimSpace(req.URL) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.DownloadResponse{Success: false, Error: "URL boş"})
	}

	resp, err := s.download.Download(req.URL)
	if err != nil {
		status := fiber.StatusInternalServerError
		if resp != nil && strings.Contains(resp.Error, "desteklenmeyen URL") {
			status = fiber.StatusBadRequest
		}
		if resp == nil {
			resp = &domain.DownloadResponse{Success: false, Error: err.Error()}
		}
		return c.Status(status).JSON(resp)
	}
	return c.JSON(resp)
}

// Listen starts the HTTP server on config.Port().
func (s *Server) Listen() error {
	port := config.Port()
	fmt.Printf("Sunucu :%s portunda başlatılıyor\n", port)
	fmt.Println("POST /api/download  {\"url\": \"...\"}")
	fmt.Println("POST /api/upscale   {\"path\": \"/downloads/...\"}")
	fmt.Println("GET  /api/upscale/:id")
	fmt.Println("GET  /api/health")
	fmt.Println("GET  /downloads/...")
	return s.app.Listen(":" + port)
}

// App exposes the Fiber instance for tests.
func (s *Server) App() *fiber.App {
	return s.app
}
