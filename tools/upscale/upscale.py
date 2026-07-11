#!/usr/bin/env python3
"""2x Real-ESRGAN upscale with tile progress + ETA (JSON lines on stdout)."""

from __future__ import annotations

import argparse
import json
import math
import sys
import time
from pathlib import Path

import cv2
import numpy as np
import onnxruntime as ort


def emit(event: str, **payload) -> None:
    print(json.dumps({"event": event, **payload}, ensure_ascii=False), flush=True)


def pad_to_multiple(img: np.ndarray, multiple: int) -> tuple[np.ndarray, int, int]:
    h, w = img.shape[:2]
    pad_h = (multiple - h % multiple) % multiple
    pad_w = (multiple - w % multiple) % multiple
    if pad_h == 0 and pad_w == 0:
        return img, h, w
    padded = cv2.copyMakeBorder(img, 0, pad_h, 0, pad_w, cv2.BORDER_REFLECT_101)
    return padded, h, w


def tile_grid(h: int, w: int, tile: int, overlap: int) -> list[tuple[int, int, int, int]]:
    stride = max(tile - overlap, 1)
    tiles: list[tuple[int, int, int, int]] = []
    for y in range(0, h, stride):
        for x in range(0, w, stride):
            y0 = y
            x0 = x
            y1 = min(y0 + tile, h)
            x1 = min(x0 + tile, w)
            if y1 - y0 < tile and y0 > 0:
                y0 = max(0, y1 - tile)
            if x1 - x0 < tile and x0 > 0:
                x0 = max(0, x1 - tile)
            tiles.append((y0, x0, y1, x1))
    # de-dupe identical tiles at edges
    unique: list[tuple[int, int, int, int]] = []
    seen: set[tuple[int, int, int, int]] = set()
    for t in tiles:
        if t not in seen:
            seen.add(t)
            unique.append(t)
    return unique


def main() -> int:
    parser = argparse.ArgumentParser(description="2x Real-ESRGAN ONNX upscaler")
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--tile", type=int, default=128)
    parser.add_argument("--overlap", type=int, default=16)
    parser.add_argument("--scale", type=int, default=2)
    args = parser.parse_args()

    if args.scale != 2:
        emit("error", message="only scale=2 is supported")
        return 1
    if not args.input.is_file():
        emit("error", message=f"input not found: {args.input}")
        return 1
    if not args.model.is_file():
        emit("error", message=f"model not found: {args.model}")
        return 1

    bgr = cv2.imread(str(args.input), cv2.IMREAD_COLOR)
    if bgr is None:
        emit("error", message="failed to read input image")
        return 1

    rgb = cv2.cvtColor(bgr, cv2.COLOR_BGR2RGB)
    # pixel_unshuffle requires even dimensions
    rgb, orig_h, orig_w = pad_to_multiple(rgb, 2)

    session = ort.InferenceSession(
        str(args.model),
        providers=["CPUExecutionProvider"],
    )
    input_name = session.get_inputs()[0].name

    tile = max(32, args.tile - (args.tile % 2))
    overlap = max(0, min(args.overlap, tile // 2))
    h, w = rgb.shape[:2]
    tiles = tile_grid(h, w, tile, overlap)
    scale = args.scale
    out_h, out_w = h * scale, w * scale
    output = np.zeros((out_h, out_w, 3), dtype=np.float32)
    weight = np.zeros((out_h, out_w, 1), dtype=np.float32)

    total = len(tiles)
    megapixels = (orig_h * orig_w) / 1_000_000
    # Initial ETA heuristic; refined from measured tile timings.
    seed_seconds = max(8.0, megapixels * 35.0)
    emit(
        "start",
        width=orig_w,
        height=orig_h,
        tiles=total,
        megapixels=round(megapixels, 3),
        eta_seconds=int(math.ceil(seed_seconds)),
        percent=0.0,
    )

    started = time.monotonic()
    tile_times: list[float] = []

    for index, (y0, x0, y1, x1) in enumerate(tiles, start=1):
        tile_start = time.monotonic()
        patch = rgb[y0:y1, x0:x1].astype(np.float32) / 255.0
        # reflect-pad patch to even size if needed (already even image, tiles may be odd at edge)
        ph, pw = patch.shape[:2]
        pad_h = ph % 2
        pad_w = pw % 2
        if pad_h or pad_w:
            patch = cv2.copyMakeBorder(patch, 0, pad_h, 0, pad_w, cv2.BORDER_REFLECT_101)
        tensor = np.transpose(patch, (2, 0, 1))[None, ...].astype(np.float32)
        pred = session.run(None, {input_name: tensor})[0][0]
        pred = np.transpose(pred, (1, 2, 0))
        pred = pred[: ph * scale, : pw * scale]

        oy0, ox0 = y0 * scale, x0 * scale
        oy1, ox1 = oy0 + pred.shape[0], ox0 + pred.shape[1]

        # feather mask for overlap blending
        mask = np.ones((pred.shape[0], pred.shape[1], 1), dtype=np.float32)
        fade = max(overlap * scale, 1)
        if fade > 1:
            for i in range(min(fade, mask.shape[0])):
                mask[i, :, 0] *= (i + 1) / fade
                mask[-i - 1, :, 0] *= (i + 1) / fade
            for i in range(min(fade, mask.shape[1])):
                mask[:, i, 0] *= (i + 1) / fade
                mask[:, -i - 1, 0] *= (i + 1) / fade

        output[oy0:oy1, ox0:ox1] += pred * mask
        weight[oy0:oy1, ox0:ox1] += mask

        elapsed_tile = time.monotonic() - tile_start
        tile_times.append(elapsed_tile)
        remaining = total - index
        avg = sum(tile_times) / len(tile_times)
        eta = int(math.ceil(avg * remaining))
        percent = round(100.0 * index / total, 1)
        emit(
            "progress",
            percent=percent,
            tile=index,
            tiles=total,
            eta_seconds=eta,
            elapsed_seconds=round(time.monotonic() - started, 1),
        )

    weight = np.maximum(weight, 1e-6)
    merged = np.clip(output / weight, 0.0, 1.0)
    merged = merged[: orig_h * scale, : orig_w * scale]
    out_bgr = cv2.cvtColor((merged * 255.0).round().astype(np.uint8), cv2.COLOR_RGB2BGR)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    if not cv2.imwrite(str(args.output), out_bgr):
        emit("error", message="failed to write output image")
        return 1

    emit(
        "done",
        path=str(args.output),
        width=orig_w * scale,
        height=orig_h * scale,
        elapsed_seconds=round(time.monotonic() - started, 1),
        percent=100.0,
        eta_seconds=0,
    )
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as exc:  # noqa: BLE001 - surface to Go parent
        emit("error", message=str(exc))
        raise SystemExit(1) from exc
