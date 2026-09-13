package health

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 三个桩服务器分别对应三态：200 → up / 502 → degraded / 无服务 → down
func TestProbeStatuses(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "ok.test" { // Host 头路由契约：域名必须送达服务器
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway) // PHP 容器异常的真实形态
	}))
	defer bad.Close()

	if got := Probe("ok.test", portOf(t, ok.URL), 2*time.Second); got.Status != Up || got.Code != 200 {
		t.Errorf("期望 up/200，得到 %+v", got)
	}
	if got := Probe("bad.test", portOf(t, bad.URL), 2*time.Second); got.Status != Degraded || got.Code != 502 {
		t.Errorf("期望 degraded/502，得到 %+v", got)
	}
	if got := Probe("none.test", 1, 2*time.Second); got.Status != Down || got.Err == "" {
		t.Errorf("期望 down 带原因，得到 %+v", got)
	}
}

// 超时必须归为 Down（长操作截止时间契约，禁无边界等待）
func TestProbeTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()
	if got := Probe("slow.test", portOf(t, slow.URL), 50*time.Millisecond); got.Status != Down {
		t.Errorf("超时应归 down，得到 %+v", got)
	}
}

// portOf 从 httptest URL（http://127.0.0.1:PORT）提取端口
func portOf(t *testing.T, url string) int {
	t.Helper()
	port, err := strconv.Atoi(url[strings.LastIndex(url, ":")+1:])
	if err != nil {
		t.Fatalf("无法解析测试端口 %s: %v", url, err)
	}
	return port
}
