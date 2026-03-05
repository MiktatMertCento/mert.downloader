# === Build Stage ===
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# === Test Stage ===
RUN go test -v ./...

# === Compile ===
RUN CGO_ENABLED=0 GOOS=linux go build -o insta-downloader .

# === Runtime Stage ===
FROM alpine:3.21

RUN apk add --no-cache ffmpeg python3 py3-pip && \
    pip3 install --no-cache-dir --break-system-packages yt-dlp

WORKDIR /app

COPY --from=builder /build/insta-downloader .

RUN mkdir -p downloads

EXPOSE 3000

CMD ["./insta-downloader"]
