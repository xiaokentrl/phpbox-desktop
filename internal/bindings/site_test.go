package bindings

import "testing"

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
