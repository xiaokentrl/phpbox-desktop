#!/usr/bin/env bash
# phpbox Desktop 用户级安装（Linux）：无 root，全部落在 $HOME 下。
# 事实来源：build/linux/phpbox-desktop.desktop（wails3 generate + StartupWMClass 行级追加）。
# 幂等可重复执行；卸载 = 删除下列三个文件（脚本末尾打印清单）。
set -euo pipefail

BIN_NAME="phpbox-desktop"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"   # 仓库根

BIN_SRC="$ROOT/bin/$BIN_NAME"
BIN_DST="$HOME/.local/bin/$BIN_NAME"

ICON_SRC="$ROOT/build/appicon.png"
ICON_DST="$HOME/.local/share/icons/hicolor/128x128/apps/$BIN_NAME.png"

DESKTOP_SRC="$ROOT/build/linux/$BIN_NAME.desktop"
DESKTOP_DST="$HOME/.local/share/applications/$BIN_NAME.desktop"

[ -x "$BIN_SRC" ]   || { echo "[ERR] 未找到 $BIN_SRC（先构建：wails3 task build）" >&2; exit 1; }
[ -f "$ICON_SRC" ]  || { echo "[ERR] 未找到 $ICON_SRC" >&2; exit 1; }
[ -f "$DESKTOP_SRC" ] || { echo "[ERR] 未找到 $DESKTOP_SRC" >&2; exit 1; }

mkdir -p "$(dirname "$BIN_DST")" "$(dirname "$ICON_DST")" "$(dirname "$DESKTOP_DST")"

install -m 0755 "$BIN_SRC" "$BIN_DST"
install -m 0644 "$ICON_SRC" "$ICON_DST"
install -m 0644 "$DESKTOP_SRC" "$DESKTOP_DST"

# Exec 用裸名，依赖 ~/.local/bin 在 PATH 中；desktop 侧补绝对路径兜底（双击启动不继承 shell PATH）
if ! grep -q '^Exec=/.*/' "$DESKTOP_DST"; then
  sed -i "s|^Exec=$BIN_NAME|Exec=$BIN_DST|" "$DESKTOP_DST"
fi

# 图标缓存 + 应用菜单缓存（任一缺失则跳过，不失败）
gtk-update-icon-cache -q -t -f "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true

desktop-file-validate "$DESKTOP_DST" || exit 1

echo "[OK] 已安装（用户级，无 root）："
echo "  $BIN_DST"
echo "  $ICON_DST"
echo "  $DESKTOP_DST"
echo "[OK] 应用菜单搜索 phpbox-desktop 即可启动；~/.local/bin 不在 PATH 时终端直接执行：$BIN_DST"
