# Mert Downloader

Instagram ve YouTube içeriklerini indiren Go API + React arayüzü. Docker ile tek container’da çalışır; amd64 ve arm64 (Raspberry Pi 5) destekler.

## Özellikler

- **Instagram:** post, carousel, reel, story, highlight, profil highlight cover’ları
- **YouTube:** watch, Shorts, `youtu.be` — en yüksek kalite MP4 (`yt-dlp` + `ffmpeg`)
- **Web UI:** tam ekran carousel önizleme, swipe / ok / nokta gezinme
- **2x Netleştir:** RealESRGAN x2plus (ONNX) ile görsel upscale; progress + ETA; basılı tutarak eski/yeni karşılaştırma
- **PWA:** ana ekrana ekle, Web Share Target ile uygulamaya link paylaş; `index.html` / SW no-cache (deploy sonrası hard refresh gerekmez)
- **Otomatik temizlik:** `downloads/` altındaki klasörler ~5 dk sonra silinir

## Desteklenen URL’ler

| Platform | Örnek | İçerik |
|---|---|---|
| Instagram | `instagram.com/p/<code>` | Fotoğraf, video, carousel |
| Instagram | `instagram.com/reel/<code>` | Reels |
| Instagram | `instagram.com/stories/<user>` veya `.../<user>/<id>` | Story |
| Instagram | `instagram.com/stories/highlights/<id>` | Highlight |
| Instagram | `instagram.com/<username>` | Profil highlight cover’ları |
| YouTube | `youtube.com/watch?v=<id>` | Video |
| YouTube | `youtube.com/shorts/<id>` | Shorts |
| YouTube | `youtu.be/<id>` | Kısa link |

## Gereksinimler

- Docker ve Docker Compose
- `cookies.txt` — Instagram için Netscape formatında cookie dosyası (YouTube için gerekmez)

## Cookie ayarı

1. [Get cookies.txt Locally](https://chromewebstore.google.com/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc) eklentisini kurun
2. Instagram’a giriş yapın, cookie’leri export edin
3. Dosyayı proje köküne `cookies.txt` olarak koyun

Örnek format: `cookies.example.txt`. Dosya `.gitignore`’dadır; commit etmeyin.

## Çalıştırma

```bash
docker compose up -d --build
```

Bu komut frontend’i derler, RealESRGAN modelini ONNX’e çevirir, Go testlerini çalıştırır, `ffmpeg` / `yt-dlp` / `onnxruntime` ile image üretir ve `http://localhost:1905` üzerinde ayağa kaldırır.

```bash
docker compose down
```

Manuel:

```bash
docker build -t mert-downloader .
docker run -d \
  -p 1905:1905 \
  -v ./cookies.txt:/app/cookies.txt:ro \
  -v ./downloads:/app/downloads \
  --name mert-downloader \
  mert-downloader
```

Port: `PORT` env (varsayılan `1905`).

## Proje yapısı

```
cmd/server/                 # giriş noktası
internal/
  config/                   # env + sabitler
  domain/                   # DTO / domain modelleri
  mediaurl/                 # URL parse (IG + YouTube)
  cookies/                  # Netscape cookie okuma
  fetch/                    # HTTP indirme + yt-dlp + cleanup
  instagram/                # Instagram API / parse
  downloader/               # use-case orkestrasyonu
  httpserver/               # Fiber HTTP katmanı (API + static + SPA)
  upscale/                  # 2x Real-ESRGAN job yöneticisi
tools/upscale/              # ONNX export + inference
web/                        # React (Vite) + PWA
models/                     # runtime ONNX (Docker build üretir)
downloads/                  # indirilen medya (gitignore)
```

## API

### Sağlık

```bash
curl http://localhost:1905/api/health
```

```json
{
  "status": "ok",
  "user_id": "123456789",
  "upscale_ready": true
}
```

### İndirme

```bash
curl -X POST http://localhost:1905/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.instagram.com/p/ABC123xyz/"}'
```

Aynı endpoint story / highlight / reel / YouTube URL’leriyle de çalışır.

Başarılı yanıt:

```json
{
  "success": true,
  "shortcode": "ABC123xyz",
  "media_type": "carousel",
  "username": "ornek",
  "caption": "...",
  "files": [
    {
      "filename": "ABC123xyz_1.jpg",
      "path": "/downloads/ABC123xyz/ABC123xyz_1.jpg",
      "type": "image",
      "size": 123456,
      "width": 1080,
      "height": 1350
    }
  ]
}
```

Hata:

```json
{
  "success": false,
  "error": "desteklenmeyen URL formatı"
}
```

Dosyalar `GET /downloads/...` üzerinden sunulur (byte-range destekli).

### 2x Netleştir

UI’daki **2x Netleştir** butonu veya:

```bash
curl -X POST http://localhost:1905/api/upscale \
  -H "Content-Type: application/json" \
  -d '{"path":"/downloads/ABC123/photo.jpg"}'

curl http://localhost:1905/api/upscale/<job-id>
```

```json
{
  "id": "...",
  "status": "running",
  "percent": 42.5,
  "eta_seconds": 18,
  "elapsed_seconds": 12.3
}
```

Model: **RealESRGAN_x2plus** (tile tabanlı). Pi 5’te tipik IG görselleri genelde 1–2 dk.

Upscale env (Docker varsayılanları):

| Değişken | Anlamı |
|---|---|
| `UPSCALE_PYTHON` | Python binary |
| `UPSCALE_SCRIPT` | `tools/upscale/upscale.py` |
| `UPSCALE_MODEL` | ONNX model yolu |
| `UPSCALE_TILE` | Tile boyutu (varsayılan 128) |
| `UPSCALE_THREADS` | Thread sayısı |

## Testler

```bash
go test -count=1 ./...

cd web && pnpm test -- --run
```

Docker build sırasında Go testleri otomatik çalışır; başarısızsa image üretilmez.

## CI/CD

GitHub Actions (`.github/workflows/ci.yml`):

- PR’larda Go + frontend testleri
- `main`’e merge’de image `ghcr.io`’ya push
- Branch adı `v*` ise versiyon tag’i + GitHub Release

## Notlar

- Instagram indirmeleri için geçerli `cookies.txt` gerekir; süresi dolunca yenileyin
- SPA shell (`/`, `index.html`) ve service worker no-cache; hashed `/assets/*` uzun cache
- Videolar mümkün olan en yüksek kalitede MP4 olarak birleştirilir
- Sadece kişisel kullanım içindir
