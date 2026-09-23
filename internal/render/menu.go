package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/SungMinWoo/tasklet/internal/store"
)

// Menu는 SwiftBar가 읽는 텍스트를 만든다 (DESIGN.md 7장).
// 첫 줄 = 메뉴바 타이틀, --- 아래 = 드롭다운, -- 는 하위메뉴 한 단.
//
// exe는 클릭했을 때 실행할 tasklet 경로.
func Menu(f *store.File, cfg store.Config, now time.Time, exe string) string {
	th := ThemeByKey(cfg.Theme)
	b := Buckets(f.Tasks, now, cfg.WeekendDueBucket)

	var sb strings.Builder
	sb.WriteString(title(b, th, cfg.Mascot))
	sb.WriteString("\n---\n")

	for _, key := range BucketOrder {
		tasks := b[key]
		sb.WriteString(bucketLine(key, len(tasks), th))
		for _, t := range tasks {
			sb.WriteString("--" + escape(itemLabel(t, now)) + "\n")
			sb.WriteString("----완료 | " + action(exe, "done", t.ID) + "\n")
			sb.WriteString("----수정 | " + action(exe, "edit", t.ID) + "\n")
			sb.WriteString("----기한\n")
			for _, d := range dueChoices {
				sb.WriteString("------" + d.label + " | " + actionDue(exe, t.ID, d.spec) + "\n")
			}
		}
	}

	sb.WriteString("---\n")
	sb.WriteString("+ 추가하기 | bash=\"" + exe + "\" param1=prompt terminal=false refresh=true color=" + th.Accent.Param() + "\n")
	sb.WriteString(settingsMenu(cfg, th, exe))
	return sb.String()
}

// title은 메뉴바에 늘 보이는 첫 줄: 캐릭터 그림 + 숫자.
// 캐릭터에 고유 색이 있어 '지남'은 캐릭터 대신 숫자 색으로 알린다 (DESIGN.md 7장).
func title(b map[BucketKey][]store.Task, th Theme, mascot string) string {
	past, today := len(b[Past]), len(b[Today])
	rest := 0
	for _, k := range BucketOrder {
		rest += len(b[k])
	}
	var text, extra string
	switch {
	case past > 0:
		text = fmt.Sprintf("%d · 오늘 %d", past, today)
		extra = " color=" + th.Late.Param()
	case today > 0:
		text = fmt.Sprintf("오늘 %d", today)
	case rest > 0:
		// 오늘·지남은 없지만 남은 일이 있을 때. "클리어"로 보이면 오해된다.
		text = fmt.Sprintf("남음 %d", rest)
	default:
		text = "클리어"
	}
	if img := MascotImageParam(mascot); img != "" {
		return text + " | " + img + extra
	}
	if extra != "" {
		return text + " |" + extra
	}
	return text
}

func bucketLine(key BucketKey, n int, th Theme) string {
	line := fmt.Sprintf("%s %s %d | font=Menlo", pad(key.Label(), 4), bar(n), n)
	switch key {
	case Past:
		if n > 0 {
			line += " color=" + th.Late.Param()
		}
	case Today:
		if n > 0 {
			line += " color=" + th.TodayC.Param()
		}
	case Tomorrow, ThisWeek:
		if n > 0 {
			line += " color=" + th.Soon.Param()
		}
	case Later:
		if n > 0 {
			line += " color=" + th.Later.Param()
		}
	}
	return line + "\n"
}

// bar는 ▇ 반복. 너무 길어지지 않게 12개에서 멈춘다.
func bar(n int) string {
	if n > 12 {
		return strings.Repeat("▇", 12) + "…"
	}
	return strings.Repeat("▇", n)
}

// pad는 한글 폭을 고려해 라벨 뒤를 공백으로 채운다 (Menlo 고정폭 기준, 한글 1자 = 2칸).
func pad(s string, width int) string {
	w := 0
	for _, r := range s {
		if r > 0x1100 {
			w += 2
		} else {
			w++
		}
	}
	if w >= width*2 {
		return s
	}
	return s + strings.Repeat(" ", width*2-w)
}

func itemLabel(t store.Task, now time.Time) string {
	parts := []string{t.Title}
	if t.Requester != nil && *t.Requester != "" {
		parts = append(parts, *t.Requester)
	}
	if d := relativeDue(t.DueAt, now); d != "" {
		parts = append(parts, d)
	}
	return strings.Join(parts, " · ")
}

// relativeDue는 항목 뒤에 붙는 짧은 기한: -2일 / 오늘 / 내일 / 9/29(화)
func relativeDue(dueAt *string, now time.Time) string {
	if dueAt == nil {
		return ""
	}
	d, err := time.ParseInLocation(time.DateOnly, *dueAt, now.Location())
	if err != nil {
		return *dueAt
	}
	days := int(d.Sub(midnight(now)).Hours() / 24)
	switch {
	case days < 0:
		return fmt.Sprintf("%d일", days)
	case days == 0:
		return "오늘"
	case days == 1:
		return "내일"
	default:
		return fmt.Sprintf("%d/%d(%s)", int(d.Month()), d.Day(),
			[...]string{"일", "월", "화", "수", "목", "금", "토"}[d.Weekday()])
	}
}

// dueChoices는 '기한 ▸' 하위메뉴. Claude를 부르지 않고 Go가 바로 계산한다.
var dueChoices = []struct{ label, spec string }{
	{"오늘", "+0d"},
	{"내일", "+1d"},
	{"이번 주 금요일", "eow"},
	{"다음 주", "next_eow"},
	{"기한 없음", "none"},
}

func settingsMenu(cfg store.Config, th Theme, exe string) string {
	var sb strings.Builder
	sb.WriteString("테마\n")
	for _, t := range Themes {
		line := "--" + check(t.Key == cfg.Theme) + t.Name +
			" | bash=\"" + exe + "\" param1=theme param2=" + t.Key + " terminal=false refresh=true"
		if t.Key == cfg.Theme {
			line += " color=" + th.Accent.Param()
		}
		sb.WriteString(line + "\n")
	}
	sb.WriteString("캐릭터\n")
	for _, m := range Mascots {
		line := "--" + check(m.Key == cfg.Mascot) + m.Name +
			" | bash=\"" + exe + "\" param1=mascot param2=" + m.Key + " terminal=false refresh=true"
		if m.Key == cfg.Mascot {
			line += " color=" + th.Accent.Param()
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func check(on bool) string {
	if on {
		return "✓ "
	}
	return "   "
}

func action(exe, cmd string, id int) string {
	return fmt.Sprintf("bash=%q param1=%s param2=%d terminal=false refresh=true", exe, cmd, id)
}

func actionDue(exe string, id int, spec string) string {
	return fmt.Sprintf("bash=%q param1=due param2=%d param3=%s terminal=false refresh=true", exe, id, spec)
}

// escape는 SwiftBar가 파라미터 구분자로 쓰는 | 를 없앤다. 줄바꿈도 한 줄로 만든다.
func escape(s string) string {
	s = strings.ReplaceAll(s, "|", "∣")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
