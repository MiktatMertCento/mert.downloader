# 📥 Insta Downloader

Instagram ve YouTube içeriklerini en yüksek kalitede indiren Go API sunucusu.

## 🎯 Desteklenen Platformlar ve URL'ler

| Platform | URL Formatı | İçerik |
|---|---|---|
| Instagram | `instagram.com/p/<code>` | Fotoğraf, video, carousel |
| Instagram | `instagram.com/reel/<code>` | Reels |
| YouTube | `youtube.com/watch?v=<id>` | Video |
| YouTube | `youtube.com/shorts/<id>` | Shorts |
| YouTube | `youtu.be/<id>` | Kısa link |

## 📋 Gereksinimler

- 🐳 **Docker** ve **Docker Compose**
- 🍪 **cookies.txt** — Instagram indirmeleri için Netscape formatında cookie dosyası

## 🍪 Cookie Ayarı

Instagram indirmeleri için geçerli bir cookie dosyası gereklidir:

1. Tarayıcınıza [Get cookies.txt](https://chromewebstore.google.com/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc) eklentisini kurun
2. Instagram'a giriş yapın
3. Eklentiyle cookie'leri export edin
4. Dosyayı proje klasörüne `cookies.txt` olarak kaydedin

Örnek format için `cookies.example.txt` dosyasına bakın.

> ⚠️ `cookies.txt` dosyası `.gitignore`'a eklenmiştir ve repo'ya dahil edilmez. Hassas bilgilerinizi paylaşmayın.

## 🐳 Docker ile Çalıştırma

### Hızlı Başlangıç (Docker Compose)

```bash
docker compose up -d
```

Bu komut:
- Go uygulamasını derler
- `ffmpeg` ve `yt-dlp` araclarını yükler
- Sunucuyu `http://localhost:3000` adresinde başlatır

### Durdurma

```bash
docker compose down
```

### Manuel Docker Komutları

```bash
# Image oluştur
docker build -t insta-downloader .

# Container başlat
docker run -d \
  -p 3000:3000 \
  -v ./cookies.txt:/app/cookies.txt:ro \
  -v ./downloads:/app/downloads \
  --name insta-downloader \
  insta-downloader
```

## 🧪 Testler

```bash
# Container içinde testleri çalıştır
docker run --rm insta-downloader go test -v ./...
```

## 📡 API Kullanımı

### Sağlık Kontrolü

```bash
curl http://localhost:3000/api/health
```

```json
{
  "status": "ok",
  "user_id": "123456789"
}
```

### 📷 Instagram Post İndirme

```bash
curl -X POST http://localhost:3000/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.instagram.com/p/ABC123xyz/"}'
```

### 🎞️ Instagram Reel İndirme

```bash
curl -X POST http://localhost:3000/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.instagram.com/reel/XYZ789abc/"}'
```

### 🎬 YouTube Video İndirme

```bash
curl -X POST http://localhost:3000/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.youtube.com/watch?v=Ma6mYcG4STw"}'
```

### 📱 YouTube Shorts İndirme

```bash
curl -X POST http://localhost:3000/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://www.youtube.com/shorts/ogGoZuJtG84"}'
```

### 🔗 YouTube Kısa Link

```bash
curl -X POST http://localhost:3000/api/download \
  -H "Content-Type: application/json" \
  -d '{"url":"https://youtu.be/Ma6mYcG4STw"}'
```

### ✅ Başarılı Yanıt Örneği

```json
{
  "success": true,
  "shortcode": "Ma6mYcG4STw",
  "media_type": "video",
  "username": "",
  "files": [
    {
      "filename": "Ma6mYcG4STw.mp4",
      "path": "/downloads/Ma6mYcG4STw/Ma6mYcG4STw.mp4",
      "type": "video",
      "size": 15234567
    }
  ]
}
```

### ❌ Hata Yanıtı Örneği

```json
{
  "success": false,
  "error": "desteklenmeyen URL formatı"
}
```

## 📁 Proje Yapısı

```
insta-downloader/
├── main.go              # Ana uygulama kodu
├── main_test.go         # Unit testler
├── Dockerfile           # Multi-stage Docker build (test + build + runtime)
├── docker-compose.yml   # Docker Compose yapılandırması
├── .github/workflows/
│   └── ci.yml           # GitHub Actions CI/CD
├── .dockerignore        # Docker build context filtresi
├── .gitignore           # Git ignore kuralları
├── cookies.txt          # Instagram cookie dosyası (gitignore)
├── cookies.example.txt  # Örnek cookie formatı
├── go.mod
├── go.sum
└── downloads/           # İndirilen dosyalar
    ├── <shortcode>/     # Her içerik kendi klasöründe
    └── ...
```

## 📌 Notlar

- 🔒 `cookies.txt` hassas bilgi içerir, `.gitignore` ile repo dışında tutulur. Örnek format için `cookies.example.txt` dosyasını inceleyin.
- 🧪 Docker build sırasında tüm unit testler otomatik çalışır — testler başarısız olursa image oluşturulmaz.
- 🎵 `ffmpeg` ve `yt-dlp` Docker container içinde en güncel sürümleriyle otomatik yüklenir.
- 📦 İndirilen dosyalar `downloads/<id>/` klasörüne kaydedilir ve `/downloads/...` endpoint'i üzerinden erişilebilir.
- 🎥 Videolar her zaman en yüksek kalitede MP4 formatında indirilir.
- 🚀 GitHub Actions ile her push'ta testler çalışır. `v*` tag'i push edildiğinde Docker image `ghcr.io`'ya yüklenir ve GitHub Release oluşturulur.
