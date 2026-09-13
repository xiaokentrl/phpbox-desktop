// Package bindings · env 绑定：phpbox .env 读写（用户级配置）。
// 安全边界：编辑白名单只含用户级键；系统级键（容器命名/标签契约）只读展示，
// 改它们会破坏 bash 侧 get_container_name/标签契约。
package bindings

import (
	"fmt"
	"log"

	engineEnv "github.com/xiaokentrl/phpbox-desktop/internal/engine/env"
)

// Env 暴露 .env 读写能力。
type Env struct{}

// EnvKV 一个 .env 条目（Editable=是否允许 GUI 修改）。
type EnvKV struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Editable bool   `json:"editable"`
}

// envEditable GUI 可编辑白名单：用户级路径/端口/镜像源等。
// 系统级键（PROJECT_NAME、*_SERVICE_PREFIX、*_SEPARATOR、CURRENT_UID/GID、IMAGE_PREFIX）禁止修改。
var envEditable = map[string]bool{
	"WWW_ROOT": true, "MYSQL_DATA_ROOT": true, "PGSQL_DATA_ROOT": true,
	"OFFLINE_DIR": true, "APK_MIRRORS": true, "APK_TIMEOUT": true,
	"NGINX_PORT": true, "NGINX_VERSION": true, "PHP_DEFAULT_EXTENSIONS": true,
	"GO_PROJECTS_ROOT": true, "GO_DEFAULT_VERSION": true, "GO_DEFAULT_PORT": true,
	"GO_PROXY": true, "GO_CACHE_ROOT": true, "GO_CGO_ENABLED": true,
}

// ReadEnv 返回全部 .env 条目（含可编辑标记）。
func (e *Env) ReadEnv() ([]EnvKV, error) {
	kvs, err := engineEnv.Read(envFile())
	if err != nil {
		log.Printf("[绑定] Env.ReadEnv 失败: %v", err)
		return nil, err
	}
	out := make([]EnvKV, len(kvs))
	for i, kv := range kvs {
		out[i] = EnvKV{Key: kv.Key, Value: kv.Value, Editable: envEditable[kv.Key]}
	}
	log.Printf("[绑定] Env.ReadEnv → %d 个条目（可编辑 %d）", len(out), countEditable(out))
	return out, nil
}

// PatchEnv 更新白名单内的键。白名单外的键直接拒绝（而非忽略），让调用方明确失败。
func (e *Env) PatchEnv(changes map[string]string) error {
	for k := range changes {
		if !envEditable[k] {
			err := fmt.Errorf("键 %s 不在可编辑白名单（系统级配置请直接编辑 .env）", k)
			log.Printf("[绑定] Env.PatchEnv 拒绝: %v", err)
			return err
		}
	}
	if err := engineEnv.Patch(envFile(), changes); err != nil {
		log.Printf("[绑定] Env.PatchEnv 失败: %v", err)
		return err
	}
	log.Printf("[绑定] Env.PatchEnv → %d 个键已更新（原子替换）", len(changes))
	return nil
}

func countEditable(kvs []EnvKV) int {
	n := 0
	for _, kv := range kvs {
		if kv.Editable {
			n++
		}
	}
	return n
}
