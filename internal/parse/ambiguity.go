package parse

import "regexp"

// dateLike: 기한처럼 보이는 표현. A1 판정용 (DESIGN.md 8장).
// 넓게 잡는다 — 틀려도 확인창이 한 번 더 뜰 뿐이다.
var dateLike = regexp.MustCompile(
	`[월화수목금토일]요일|` + // 화요일
		`\d+\s*일|` + // 25일, 3일
		`\d{1,2}\s*/\s*\d{1,2}|` + // 10/2
		`\d+\s*월|` + // 10월
		`오늘|내일|모레|낼|` +
		`이번\s*주|다음\s*주|담주|이번\s*달|다음\s*달|주말|월말|말일|월초|` +
		`아침|점심|저녁|오전|오후|퇴근\s*전|마감|기한|데드라인`)

// Reasons는 확인창을 띄울 이유를 모은다. 비어 있으면 그냥 저장한다.
// A1(기한 없는데 날짜 표현 있음), A4(Haiku가 unsure)는 여기서,
// A2·A3·A5(날짜를 넘기거나 당김)는 ResolveDue가 준 Reason을 그대로 쓴다.
func Reasons(sentence string, r Result, resolved Resolved) []string {
	var out []string
	if r.Due.Kind == "none" && dateLike.MatchString(sentence) {
		out = append(out, "기한을 찾지 못했어요")
	}
	if resolved.Reason != "" {
		out = append(out, resolved.Reason)
	}
	if r.Unsure {
		out = append(out, "문장이 여러 뜻으로 읽혀요")
	}
	return out
}
