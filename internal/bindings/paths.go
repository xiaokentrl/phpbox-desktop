// Package bindings · 路径统一层：从 phpbox .env 派生全部目录，
// 语义对齐 bash lib/common/env.sh 的 load_env 归一化：
//   ~/ 与 ~ → $HOME；./ → $BASE_DIR 相对；裸名 → $BASE_DIR 拼接；缺失回退默认值。
// 阶段 0：BASE_DIR 硬编码 ~/phpbox（install.sh 固定布局，.env 无此键）。
package bindings

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	engineEnv "github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
)

// baseDir phpbox 安装根（env.sh：BASE_DIR=$HOME/phpbox）。
func baseDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "phpbox"
	}
	return filepath.Join(home, "phpbox")
}

// envLookup 读 .env 单键（缺失返回空）。
func envLookup(key string) string {
	kvs, err := engineEnv.Read(filepath.Join(baseDir(), ".env"))
	if err != nil {
		return ""
	}
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value
		}
	}
	return ""
}

// expandPath 对齐 env.sh 的路径归一化（OFFLINE_DIR/GO_PROJECTS_ROOT/GO_CACHE_ROOT 同规则）：
// "~" → HOME；"~/x" → HOME/x；"./x" → BASE_DIR/x；裸名 → BASE_DIR/x；绝对路径原样。
func expandPath(v string) string {
	switch {
	case v == "~":
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return v
	case strings.HasPrefix(v, "~/"):
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, v[2:])
		}
		return v
	case strings.HasPrefix(v, "./"):
		return filepath.Join(baseDir(), v[2:])
	case strings.HasPrefix(v, "/"):
		return v
	case v == "":
		return v
	default:
		return filepath.Join(baseDir(), v)
	}
}

// envPath 读 .env 路径键并归一化；空值/缺失回退 def。
func envPath(key, def string) string {
	v := envLookup(key)
	if v == "" {
		return def
	}
	return expandPath(v)
}

// 各绑定域的统一入口（此前散落各文件的 UserHomeDir+Join 硬编码全部收敛于此）：

// envFile .env 路径（ENV_FILE=$BASE_DIR/.env）。
func envFile() string { return filepath.Join(baseDir(), ".env") }

// sitesDir 站点 vhost 目录（add.sh：SITES_DIR=$CONFIG_DIR/nginx/sites，与 Nginx 版本解耦）。
func sitesDir() string { return filepath.Join(baseDir(), "config", "nginx", "sites") }

// backupsDir 备份归档目录（BACKUP_DIR=$BASE_DIR/backups）。
func backupsDir() string { return filepath.Join(baseDir(), "backups") }

// offlineDir 离线缓存库（.env OFFLINE_DIR，归一化后；默认 $BASE_DIR/offline）。
func offlineDir() string { return envPath("OFFLINE_DIR", filepath.Join(baseDir(), "offline")) }

// phpConfigDir PHP 版本配置目录（$CONFIG_DIR/php/<版本>，extensions.env 所在）。
func phpConfigDir(version string) string { return filepath.Join(baseDir(), "config", "php", version) }

// goProjectsRoot Go 项目扫描根（.env GO_PROJECTS_ROOT，归一化后；默认 ~/www）。
func goProjectsRoot() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = ""
	}
	return envPath("GO_PROJECTS_ROOT", filepath.Join(home, "www"))
}

// nginxPort 读 .env 的 NGINX_PORT；缺失/不可解析回退 80（bash 默认）。
func nginxPort() int {
	if p, err := strconv.Atoi(envLookup("NGINX_PORT")); err == nil && p > 0 {
		return p
	}
	return 80
}
