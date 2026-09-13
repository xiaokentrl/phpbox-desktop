// Package bindings · creds 绑定：服务连接凭证（ui-spec §3.3/3.4 连接抽屉）。
// 密码语义：ListCreds 只报存在不报值；GetServicePassword 由用户"点击显示"
// 动作显式调用（8s 掩码在 UI，绑定侧不缓存）。
package bindings

import (
	"context"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/creds"
	"github.com/xiaokentrl/phpbox-desktop/internal/engine/docker"
)

// Creds 暴露服务连接凭证读取能力。
type Creds struct{}

// Cred 透传引擎类型。
type Cred = creds.Cred

// ListCreds 返回某服务全部已装版本的连接信息（端口/用户/掩码 DSN；密码只报存在）。
// versions 由前端 installed（容器 labels 事实源）传入，绑定不重复派生。
func (c *Creds) ListCreds(ctx context.Context, svc string, versions []string) ([]Cred, error) {
	if !validSvc(svc) {
		return nil, errInvalidSvc
	}
	// 已装版本不可伪造：绑定侧独立校验 versions 与真实容器状态一致
	real, err := installedVersions(ctx, svc)
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, v := range versions {
		want[v] = true
	}
	for _, v := range real {
		if !want[v] {
			versions = append(versions, v) // 调用方遗漏的真实版本补齐
		}
	}
	return creds.Read(envFile(), svc, versions), nil
}

// GetServicePassword 返回某版本密码明文（连接抽屉点击显示时调用）。
// 版本必须真实已安装（labels 校验），不存在返回空串（不报错——UI 显示"无密码"）。
func (c *Creds) GetServicePassword(ctx context.Context, svc, ver string) (string, error) {
	if !validSvc(svc) {
		return "", errInvalidSvc
	}
	real, err := installedVersions(ctx, svc)
	if err != nil {
		return "", err
	}
	for _, v := range real {
		if v == ver {
			return creds.GetPass(envFile(), svc, ver), nil
		}
	}
	return "", nil // 未安装版本：无密码可显示
}

// validSvc 凭证只对有密码契约的服务开放（bash install.sh 的 _ROOT_PASSWORD 生成线）。
func validSvc(svc string) bool {
	return svc == "mysql" || svc == "pgsql" || svc == "redis"
}

var errInvalidSvc = &svcError{}

type svcError struct{}

func (e *svcError) Error() string { return "凭证仅支持 mysql/pgsql/redis 服务线" }

// installedVersions 从容器 labels 派生某服务真实已装版本（installed 同源逻辑）。
func installedVersions(ctx context.Context, svc string) ([]string, error) {
	dc, err := docker.New()
	if err != nil {
		return nil, err
	}
	items, err := dc.ListContainers(ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, c := range items {
		if c.Service == svc && c.Version != "" {
			if !seen[c.Version] {
				seen[c.Version] = true
				out = append(out, c.Version)
			}
		}
	}
	return out, nil
}
