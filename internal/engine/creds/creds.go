// Package creds · 服务连接凭证读取（ui-spec §3.3/3.4 版本卡片连接抽屉）。
// 事实来源：~/phpbox/.env 的真实键（bash install.sh:8/38 键名契约：
// <SVC>_<去点版本>_ROOT_PASSWORD 与 <SVC>_<去点版本>_PORT，转大写）。
// 只读；密码不落任何 GUI 状态，仅在调用方请求的窗口内返回。
package creds

import (
	"strings"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
)

// Cred 一个服务版本的连接信息（全部真实值；密码仅在显式请求时填入）。
type Cred struct {
	Service  string `json:"service"`  // php / mysql / pgsql / redis
	Version  string `json:"version"`  // 8.0（含点，与 label 一致）
	Port     string `json:"port"`     // .env PORT 键；缺失为空（用 bash 默认由调用方决定）
	User     string `json:"user"`     // bash 契约：mysql/pgsql root/postgres；redis 空（密码认证）
	HasPass  bool    `json:"hasPass"` // 密码键存在（不返回值——请求时单独取）
	DSNMask  string `json:"dsnMask"`  // 掩码 DSN（密码段 ***）——展示用
}

// key 服务版本键名（bash 契约大写化：mysql 8.0 → MYSQL_80）。
func key(svc, ver string) string {
	return strings.ToUpper(svc) + "_" + strings.ReplaceAll(ver, ".", "")
}

// userFor bash 契约的固定用户名（install.sh 密码显示逻辑同源）。
func userFor(svc string) string {
	switch svc {
	case "mysql":
		return "root"
	case "pgsql":
		return "postgres"
	default:
		return "" // redis: 无用户概念（requirepass 密码认证）
	}
}

// Read 列出某服务全部版本的连接信息（不含密码明文——HasPass 只报存在）。
// envPath 是 .env 路径；versions 是该服务已安装版本（installed 派生，调用方给）。
func Read(envPath, svc string, versions []string) []Cred {
	kvs, _ := env.Read(envPath)
	lookup := map[string]string{}
	for _, kv := range kvs {
		lookup[kv.Key] = kv.Value
	}
	out := make([]Cred, 0, len(versions))
	for _, v := range versions {
		k := key(svc, v)
		c := Cred{Service: svc, Version: v, User: userFor(svc)}
		c.Port = lookup[k+"_PORT"]
		if _, ok := lookup[k+"_ROOT_PASSWORD"]; ok {
			c.HasPass = true
		}
		c.DSNMask = dsn(svc, c.User, c.Port, "***")
		out = append(out, c)
	}
	return out
}

// GetPass 返回某版本密码明文（连接抽屉"点击显示"时调用；无密码返回空）。
func GetPass(envPath, svc, ver string) string {
	kvs, _ := env.Read(envPath)
	for _, kv := range kvs {
		if kv.Key == key(svc, ver)+"_ROOT_PASSWORD" {
			return kv.Value
		}
	}
	return ""
}

// dsn 按 ui-spec §3.3 生成：mysql -h127.0.0.1 -P<port> -uroot -p<pass 段> /
// postgresql://postgres:<pass>@127.0.0.1:<port>/postgres / redis-cli -a <pass>。
func dsn(svc, user, port, pass string) string {
	switch svc {
	case "mysql":
		return "mysql -h127.0.0.1 -P" + port + " -u" + user + " -p" + pass
	case "pgsql":
		return "postgresql://" + user + ":" + pass + "@127.0.0.1:" + port + "/postgres"
	case "redis":
		return "redis-cli -h 127.0.0.1 -p " + port + " -a " + pass
	default:
		return ""
	}
}
