// engine · offline 验证状态记录：GUI verify 操作是本应用执行的真实操作，
// 其发生时间/结论是事实（非伪造状态）。记录落盘于离线库根 .verify-state
// （KEY=VALUE 行格式），可被用户直接查看/编辑/删除——文件即接口。
package offline

import (
	"os"
	"strings"
	"time"
)

const verifyStateFile = ".verify-state"

// VerifyState 一条验证记录（svc/ver 键）。
type VerifyState struct {
	At     time.Time
	OK     bool
	Detail string
}

// keyOf svc/ver → 记录键。
func keyOf(svc, ver string) string { return svc + "/" + ver }

// ReadVerifyState 读取全部验证记录（文件不存在返回空——从未验证过）。
func ReadVerifyState(root string) map[string]VerifyState {
	out := map[string]VerifyState{}
	data, err := os.ReadFile(rootVerifyPath(root))
	if err != nil {
		return out
	}
	for _, ln := range strings.Split(string(data), "\n") {
		// 行格式：svc/ver|ok=1|at=RFC3339|detail（detail 可含 | 转义为 \x7c）
		parts := strings.SplitN(ln, "|", 4)
		if len(parts) != 4 {
			continue
		}
		at, err := time.Parse(time.RFC3339, parts[2][3:])
		if err != nil {
			continue
		}
		out[parts[0]] = VerifyState{
			At:     at,
			OK:     parts[1] == "ok=1",
			Detail: strings.ReplaceAll(parts[3][7:], "\\x7c", "|"),
		}
	}
	return out
}

// WriteVerifyState 追加/覆盖一条记录（读改写整文件，保持行序其余不动）。
func WriteVerifyState(root, svc, ver string, ok bool, detail string) {
	key := keyOf(svc, ver)
	all := ReadVerifyState(root)
	all[key] = VerifyState{At: time.Now(), OK: ok, Detail: detail}
	var b strings.Builder
	for k, v := range all {
		detail := strings.ReplaceAll(v.Detail, "|", "\\x7c")
		okFlag := "ok=0"
		if v.OK {
			okFlag = "ok=1"
		}
		b.WriteString(k + "|" + okFlag + "|at=" + v.At.Format(time.RFC3339) + "|detail=" + detail + "\n")
	}
	_ = os.WriteFile(rootVerifyPath(root), []byte(b.String()), 0o644) // 记录失败不阻断 verify 主流程
}

// RemoveVerifyState 删除一条记录（缓存被清理时）。
func RemoveVerifyState(root, svc, ver string) {
	all := ReadVerifyState(root)
	delete(all, keyOf(svc, ver))
	var b strings.Builder
	for k, v := range all {
		detail := strings.ReplaceAll(v.Detail, "|", "\\x7c")
		okFlag := "ok=0"
		if v.OK {
			okFlag = "ok=1"
		}
		b.WriteString(k + "|" + okFlag + "|at=" + v.At.Format(time.RFC3339) + "|detail=" + detail + "\n")
	}
	_ = os.WriteFile(rootVerifyPath(root), []byte(b.String()), 0o644)
}

func rootVerifyPath(root string) string {
	return root + "/" + verifyStateFile
}
