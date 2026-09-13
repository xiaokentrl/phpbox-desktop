package bindings

import (
	"testing"

	"github.com/xiaokentrl/phpbox-desktop/internal/engine/health"
)

// 集成测试：真实解析 ~/phpbox/config/nginx/sites/。目录不存在（未安装 nginx）时 SKIPPED。
func TestListSitesThroughBinding(t *testing.T) {
	sites, err := (&Site{}).ListSites()
	if err != nil {
		t.Skipf("sites 目录不可读（%v）：SKIPPED", err)
	}
	t.Logf("绑定链路贯通：%s → %d 个站点", sitesDir(), len(sites))
	if len(sites) == 0 {
		return
	}
	for _, s := range sites {
		if s.PHP == "" {
			t.Errorf("站点 %s 未解析出 PHP upstream（vhost 模板不符）: %+v", s.Domain, s)
		}
		if s.Root == "" {
			t.Errorf("站点 %s 未解析出 root: %+v", s.Domain, s)
		}
	}
}

// 集成测试：真实探活（nginx 运行中时站点应 up；探测失败=down 也合法——诚实降级不臆造）。
// 无站点或 .env 不可读时 SKIPPED。
func TestProbeSiteHealthThroughBinding(t *testing.T) {
	sites, err := (&Site{}).ListSites()
	if err != nil || len(sites) == 0 {
		t.Skipf("无站点可探（%v）：SKIPPED", err)
	}
	res := (&Site{}).ProbeSiteHealth(sites[0].Domain)
	t.Logf("真实探活 %s → %s code=%d err=%q", sites[0].Domain, res.Status, res.Code, res.Err)
	switch res.Status {
	case health.Up, health.Degraded, health.Down: // 三态均合法：down 意 nginx 未跑/端口未监听，是真实状态
	default:
		t.Errorf("探测返回非法状态: %+v", res)
	}
	if res.Status == health.Up && res.Code < 200 {
		t.Errorf("up 状态必须有状态码: %+v", res)
	}
}
