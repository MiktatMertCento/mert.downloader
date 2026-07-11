# === Export RealESRGAN x2plus → ONNX (amd64 / arm64 builders) ===
FROM python:3.12-slim-bookworm AS upscale-model

RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY tools/upscale/rrdbnet.py tools/upscale/export_onnx.py ./

RUN pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir torch --index-url https://download.pytorch.org/whl/cpu \
    && pip install --no-cache-dir numpy onnx onnxscript

RUN curl -fsSL -o RealESRGAN_x2plus.pth \
      https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.1/RealESRGAN_x2plus.pth \
    && python export_onnx.py \
      --weights RealESRGAN_x2plus.pth \
      --output /out/realesrgan-x2plus.onnx \
      --tile 128 \
    && rm -f RealESRGAN_x2plus.pth

# === Frontend Build ===
FROM node:22-bookworm-slim AS frontend

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /build

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ .

ARG VITE_APP_ORIGIN=https://downloader.miktatmert.dev
ENV VITE_APP_ORIGIN=${VITE_APP_ORIGIN}

RUN pnpm run build

# === Go Build Stage ===
FROM golang:1.26-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test -count=1 ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o insta-downloader ./cmd/server

# === Runtime Stage (glibc — required by onnxruntime wheels on amd64/arm64) ===
FROM python:3.12-slim-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg \
      ca-certificates \
      libgl1 \
      libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/* \
    && pip install --no-cache-dir --upgrade pip \
    && pip install --no-cache-dir yt-dlp onnxruntime opencv-python-headless numpy

WORKDIR /app

COPY --from=frontend /build/dist ./web/dist
COPY --from=builder /build/insta-downloader .
COPY --from=upscale-model /out/realesrgan-x2plus.onnx ./models/realesrgan-x2plus.onnx
COPY tools/upscale/upscale.py ./tools/upscale/upscale.py

ENV UPSCALE_PYTHON=python3 \
    UPSCALE_SCRIPT=/app/tools/upscale/upscale.py \
    UPSCALE_MODEL=/app/models/realesrgan-x2plus.onnx \
    UPSCALE_TILE=128 \
    UPSCALE_THREADS=4

RUN mkdir -p downloads \
    && chmod +x /app/tools/upscale/upscale.py

EXPOSE 1905

CMD ["./insta-downloader"]
