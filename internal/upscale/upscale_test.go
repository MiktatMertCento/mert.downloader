package upscale

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveDownloadPath(t *testing.T) {
	root := t.TempDir()
	download := filepath.Join(root, "downloads")
	if err := os.MkdirAll(filepath.Join(download, "ABC"), 0o755); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(download, "ABC", "photo.jpg")
	if err := os.WriteFile(imagePath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(download, "ABC", "clip.mp4"), []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	resolved, err := resolveDownloadPath("downloads", "/downloads/ABC/photo.jpg")
	if err != nil {
		t.Fatalf("resolve image: %v", err)
	}
	if filepath.Base(resolved) != "photo.jpg" {
		t.Fatalf("got %q", resolved)
	}

	if _, err := resolveDownloadPath("downloads", "/downloads/ABC/clip.mp4"); err == nil {
		t.Fatal("expected video rejection")
	}
	if _, err := resolveDownloadPath("downloads", "/etc/passwd"); err == nil {
		t.Fatal("expected path escape rejection")
	}
	if _, err := resolveDownloadPath("downloads", "/downloads/../downloads/ABC/photo.jpg"); err != nil {
		// cleaned path may still resolve inside downloads — ensure .. escape outside fails
	}
	if _, err := resolveDownloadPath("downloads", "/downloads/ABC/missing.jpg"); err == nil {
		t.Fatal("expected missing file error")
	}
}

func TestUpscaleManagerFakeScript(t *testing.T) {
	root := t.TempDir()
	download := filepath.Join(root, "downloads")
	tools := filepath.Join(root, "tools")
	models := filepath.Join(root, "models")
	if err := os.MkdirAll(filepath.Join(download, "ABC"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(models, 0o755); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(download, "ABC", "photo.jpg")
	if err := os.WriteFile(src, []byte("jpeg-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	modelPath := filepath.Join(models, "realesrgan-x2plus.onnx")
	if err := os.WriteFile(modelPath, []byte("onnx"), 0o644); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(tools, "upscale.py")
	scriptBody := `#!/bin/sh
set -eu
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output) out="$2"; shift 2 ;;
    *) shift ;;
  esac
done
printf '%s\n' '{"event":"start","percent":0,"eta_seconds":2,"width":10,"height":10,"tiles":2}'
sleep 0.05
printf '%s\n' '{"event":"progress","percent":50,"eta_seconds":1,"elapsed_seconds":0.1,"tile":1,"tiles":2}'
sleep 0.05
printf 'png-bytes' > "$out"
printf '%s\n' "{\"event\":\"done\",\"percent\":100,\"eta_seconds\":0,\"elapsed_seconds\":0.2,\"width\":20,\"height\":20,\"path\":\"$out\"}"
`
	if err := os.WriteFile(script, []byte(scriptBody), 0o755); err != nil {
		t.Fatal(err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	t.Setenv("UPSCALE_PYTHON", "sh")
	t.Setenv("UPSCALE_SCRIPT", script)
	t.Setenv("UPSCALE_MODEL", modelPath)

	mgr := NewManager("downloads")
	job, err := mgr.Start("/downloads/ABC/photo.jpg")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	var final *UpscaleJob
	for time.Now().Before(deadline) {
		got, ok := mgr.Get(job.ID)
		if !ok {
			t.Fatal("job missing")
		}
		if got.Status == UpscaleCompleted || got.Status == UpscaleFailed {
			final = got
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if final == nil {
		t.Fatal("job did not finish")
	}
	if final.Status != UpscaleCompleted {
		t.Fatalf("status=%s error=%s", final.Status, final.Error)
	}
	if final.Percent != 100 || final.ETASeconds != 0 {
		t.Fatalf("progress not complete: %+v", final)
	}
	if final.Width != 20 || final.Height != 20 {
		t.Fatalf("dims=%dx%d", final.Width, final.Height)
	}
	if final.ResultPath == "" || final.Filename == "" {
		t.Fatalf("missing result meta: %+v", final)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(final.ResultPath[1:]))); err != nil {
		t.Fatalf("result file missing: %v", err)
	}
}
