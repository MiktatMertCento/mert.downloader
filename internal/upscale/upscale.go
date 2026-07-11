package upscale

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type UpscaleStatus string

const (
	UpscaleQueued    UpscaleStatus = "queued"
	UpscaleRunning   UpscaleStatus = "running"
	UpscaleCompleted UpscaleStatus = "completed"
	UpscaleFailed    UpscaleStatus = "failed"
)

type UpscaleRequest struct {
	Path string `json:"path"`
}

type UpscaleJob struct {
	ID             string        `json:"id"`
	Status         UpscaleStatus `json:"status"`
	SourcePath     string        `json:"source_path"`
	ResultPath     string        `json:"result_path,omitempty"`
	Filename       string        `json:"filename,omitempty"`
	Width          int           `json:"width,omitempty"`
	Height         int           `json:"height,omitempty"`
	Size           int64         `json:"size,omitempty"`
	Percent        float64       `json:"percent"`
	ETASeconds     int           `json:"eta_seconds"`
	ElapsedSeconds float64       `json:"elapsed_seconds"`
	Error          string        `json:"error,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type upscaleEvent struct {
	Event          string  `json:"event"`
	Percent        float64 `json:"percent"`
	ETASeconds     int     `json:"eta_seconds"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Path           string  `json:"path"`
	Message        string  `json:"message"`
	Tiles          int     `json:"tiles"`
	Tile           int     `json:"tile"`
}

type Manager struct {
	mu        sync.RWMutex
	jobs      map[string]*UpscaleJob
	download  string
	pythonBin string
	script    string
	model     string
	tile      string
}

func NewManager(downloadDir string) *Manager {
	pythonBin := envOr("UPSCALE_PYTHON", "python3")
	script := envOr("UPSCALE_SCRIPT", filepath.Join("tools", "upscale", "upscale.py"))
	model := envOr("UPSCALE_MODEL", filepath.Join("models", "realesrgan-x2plus.onnx"))
	tile := envOr("UPSCALE_TILE", "128")
	return &Manager{
		jobs:      make(map[string]*UpscaleJob),
		download:  downloadDir,
		pythonBin: pythonBin,
		script:    script,
		model:     model,
		tile:      tile,
	}
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func (m *Manager) Available() error {
	if _, err := os.Stat(m.script); err != nil {
		return fmt.Errorf("upscale script missing: %w", err)
	}
	if _, err := os.Stat(m.model); err != nil {
		return fmt.Errorf("upscale model missing: %w", err)
	}
	return nil
}

func resolveDownloadPath(downloadDir, requestPath string) (string, error) {
	cleaned := filepath.Clean("/" + strings.TrimSpace(requestPath))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", errors.New("geçersiz dosya yolu")
	}
	if strings.Contains(cleaned, "..") {
		return "", errors.New("geçersiz dosya yolu")
	}
	if !strings.HasPrefix(cleaned, downloadDir+string(os.PathSeparator)) && cleaned != downloadDir {
		if !strings.HasPrefix(cleaned, downloadDir+"/") {
			return "", errors.New("dosya downloads dışında")
		}
	}

	absDownload, err := filepath.Abs(downloadDir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absDownload, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("dosya downloads dışında")
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return "", errors.New("dosya bulunamadı")
	}
	if info.IsDir() {
		return "", errors.New("klasör upscale edilemez")
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
	default:
		return "", errors.New("sadece görsel dosyaları upscale edilebilir")
	}

	return absPath, nil
}

func (m *Manager) Start(requestPath string) (*UpscaleJob, error) {
	if err := m.Available(); err != nil {
		return nil, err
	}

	source, err := resolveDownloadPath(m.download, requestPath)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	job := &UpscaleJob{
		ID:         uuid.NewString(),
		Status:     UpscaleQueued,
		SourcePath: "/" + filepath.ToSlash(mustRelDownload(m.download, source)),
		Percent:    0,
		ETASeconds: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	go m.run(job.ID, source)
	return cloneUpscaleJob(job), nil
}

func mustRelDownload(downloadDir, absPath string) string {
	cwd, err := filepath.Abs(".")
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	return filepath.ToSlash(rel)
}

func (m *Manager) Get(id string) (*UpscaleJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneUpscaleJob(job), true
}

func cloneUpscaleJob(job *UpscaleJob) *UpscaleJob {
	copied := *job
	return &copied
}

func (m *Manager) update(id string, mutate func(*UpscaleJob)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[id]
	if !ok {
		return
	}
	mutate(job)
	job.UpdatedAt = time.Now().UTC()
}

func (m *Manager) run(id, source string) {
	m.update(id, func(job *UpscaleJob) {
		job.Status = UpscaleRunning
	})

	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	outName := fmt.Sprintf("%s_upscaled_x2.png", base)
	outPath := filepath.Join(filepath.Dir(source), outName)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		m.pythonBin,
		m.script,
		"--input", source,
		"--output", outPath,
		"--model", m.model,
		"--tile", m.tile,
		"--scale", "2",
	)
	cmd.Env = append(os.Environ(), "OMP_NUM_THREADS="+envOr("UPSCALE_THREADS", "4"))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.fail(id, err.Error())
		return
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		m.fail(id, err.Error())
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event upscaleEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		switch event.Event {
		case "start", "progress":
			m.update(id, func(job *UpscaleJob) {
				job.Percent = event.Percent
				job.ETASeconds = event.ETASeconds
				job.ElapsedSeconds = event.ElapsedSeconds
			})
		case "done":
			info, statErr := os.Stat(outPath)
			size := int64(0)
			if statErr == nil {
				size = info.Size()
			}
			rel := "/" + filepath.ToSlash(mustRelDownload(m.download, outPath))
			m.update(id, func(job *UpscaleJob) {
				job.Status = UpscaleCompleted
				job.Percent = 100
				job.ETASeconds = 0
				job.ElapsedSeconds = event.ElapsedSeconds
				job.ResultPath = rel
				job.Filename = filepath.Base(outPath)
				job.Width = event.Width
				job.Height = event.Height
				job.Size = size
			})
		case "error":
			msg := event.Message
			if msg == "" {
				msg = "upscale başarısız"
			}
			m.fail(id, msg)
			_ = cmd.Process.Kill()
			return
		}
	}

	if err := cmd.Wait(); err != nil {
		m.mu.RLock()
		job := m.jobs[id]
		status := UpscaleFailed
		if job != nil {
			status = job.Status
		}
		m.mu.RUnlock()
		if status != UpscaleCompleted {
			if ctx.Err() == context.DeadlineExceeded {
				m.fail(id, "upscale zaman aşımına uğradı")
			} else {
				m.fail(id, err.Error())
			}
		}
		return
	}

	m.mu.RLock()
	job := m.jobs[id]
	done := job != nil && job.Status == UpscaleCompleted
	m.mu.RUnlock()
	if !done {
		m.fail(id, "upscale çıktı üretmedi")
	}
}

func (m *Manager) fail(id, message string) {
	m.update(id, func(job *UpscaleJob) {
		if job.Status == UpscaleCompleted {
			return
		}
		job.Status = UpscaleFailed
		job.Error = message
		job.ETASeconds = 0
	})
}
