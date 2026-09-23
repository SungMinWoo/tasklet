package render

import (
	"embed"
	"encoding/base64"
	"fmt"
)

// PNG는 scripts/bake_mascots.sh가 구운 36×36 그림 (레티나 2배 = 메뉴바 18pt).
//
//go:embed mascot/png/*.png
var mascotPNG embed.FS

// MascotImageParam은 SwiftBar 타이틀에 붙일 image= 파라미터.
// 그림이 없으면 빈 문자열 — 글자만 나오고 메뉴는 그대로 뜬다.
func MascotImageParam(key string) string {
	b, err := mascotPNG.ReadFile("mascot/png/" + key + ".png")
	if err != nil {
		return ""
	}
	return fmt.Sprintf("image=%s", base64.StdEncoding.EncodeToString(b))
}
