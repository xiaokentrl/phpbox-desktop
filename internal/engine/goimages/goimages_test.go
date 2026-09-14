package goimages

import "testing"

// normalizeTag 是 bash _go_resolve_version 的逆映射（server.sh:26-38 契约）：
// install "1.24" → golang:1.24-alpine；install "latest"/"alpine" → golang:alpine。
func TestNormalizeTag(t *testing.T) {
	cases := []struct{ ref, wantTag, wantImage string }{
		{"golang:alpine", "alpine", "golang:alpine"},
		{"golang:1.24-alpine", "1.24", "golang:1.24-alpine"},
		{"golang:1.24.3-alpine", "1.24.3", "golang:1.24.3-alpine"},
		{"golang:1.24-bookworm", "1.24-bookworm", "golang:1.24-bookworm"}, // 非 alpine 变体：原样如实展示
	}
	for _, c := range cases {
		tag, image := normalizeTag(c.ref)
		if tag != c.wantTag || image != c.wantImage {
			t.Errorf("normalizeTag(%q) = (%q, %q), want (%q, %q)", c.ref, tag, image, c.wantTag, c.wantImage)
		}
	}
}
