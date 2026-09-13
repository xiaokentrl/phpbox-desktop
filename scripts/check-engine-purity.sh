#!/usr/bin/env bash
# 引擎零依赖检查：internal/engine/** 禁止 import 任何 Wails 包（设计文档 §3.3-1）
set -euo pipefail
cd "$(dirname "$0")/.."
if go list -deps ./internal/engine/... 2>/dev/null | grep -qi wails; then
  echo "FAIL: internal/engine 出现 wails 依赖（违反设计文档 §3.3-1）" >&2
  go list -deps ./internal/engine/... | grep -i wails >&2
  exit 1
fi
echo "引擎零依赖检查通过"
