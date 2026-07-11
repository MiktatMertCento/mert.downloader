#!/usr/bin/env python3
"""Export RealESRGAN_x2plus.pth to ONNX for portable onnxruntime inference."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

import torch

from rrdbnet import RRDBNet


def load_state_dict(path: Path) -> dict:
    raw = torch.load(path, map_location="cpu", weights_only=True)
    if isinstance(raw, dict):
        if "params_ema" in raw:
            return raw["params_ema"]
        if "params" in raw:
            return raw["params"]
    if not isinstance(raw, dict):
        raise ValueError(f"unexpected checkpoint type: {type(raw)}")
    return raw


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--weights", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--tile", type=int, default=128)
    args = parser.parse_args()

    model = RRDBNet(num_in_ch=3, num_out_ch=3, scale=2, num_feat=64, num_block=23, num_grow_ch=32)
    state = load_state_dict(args.weights)
    model.load_state_dict(state, strict=True)
    model.eval()

    tile = args.tile
    if tile % 2 != 0:
        raise SystemExit("tile must be even for x2 pixel_unshuffle")

    dummy = torch.rand(1, 3, tile, tile, dtype=torch.float32)
    args.output.parent.mkdir(parents=True, exist_ok=True)

    with torch.inference_mode():
        torch.onnx.export(
            model,
            dummy,
            args.output,
            input_names=["input"],
            output_names=["output"],
            dynamic_axes={
                "input": {2: "height", 3: "width"},
                "output": {2: "height", 3: "width"},
            },
            opset_version=17,
            dynamo=False,
        )

    print(f"exported {args.output}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
