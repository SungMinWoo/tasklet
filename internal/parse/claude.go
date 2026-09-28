package parse

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed prompt.txt
var systemPrompt string

// Result는 Haiku가 문장에서 뽑아낸 것.
type Result struct {
	Title    string `json:"title"`
	From     string `json:"from"`
	Due      Due    `json:"due"`
	Priority string `json:"priority"`
	Unsure   bool   `json:"unsure"`
}

// meta는 claude -p --output-format json의 바깥 구조. 답변은 result에 문자열로 들어온다.
type meta struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
	Subtype string `json:"subtype"`
}

// Timeout은 claude 호출 상한. 실측 보통 3~4초, 느릴 때 12초.
const Timeout = 40 * time.Second

var weekdayKoShort = [...]string{"일", "월", "화", "수", "목", "금", "토"}

// Extract는 문장 하나를 Haiku로 보내 업무 하나를 뽑는다.
func Extract(ctx context.Context, sentence string, today time.Time) (Result, error) {
	out, err := call(ctx, sentence, today, "JSON 객체 하나만 출력하라. 설명·코드펜스 금지.")
	if err != nil {
		return Result{}, err
	}
	return parseResult(out)
}

// ExtractMany는 줄 하나에서 업무를 여러 개까지 뽑는다 ('한 번에 넣기').
// 필드 규칙은 Extract와 같은 prompt.txt를 쓰고, 출력 형태만 배열로 바꾼다.
func ExtractMany(ctx context.Context, sentence string, today time.Time) ([]Result, error) {
	out, err := call(ctx, sentence, today,
		"JSON 배열만 출력하라. 서로 다른 업무가 섞여 있으면 객체를 여러 개 담고, 하나면 객체 하나만 담아라. 설명·코드펜스 금지.")
	if err != nil {
		return nil, err
	}
	return parseResults(out)
}

// call은 claude CLI를 부르고 모델 답변 문자열을 돌려준다.
// tail은 출력 형태 지시 — 필드 규칙(prompt.txt)은 건드리지 않는다.
func call(ctx context.Context, sentence string, today time.Time, tail string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()

	prompt := systemPrompt + "\n" + tail
	msg := fmt.Sprintf("오늘은 %s(%s)이다. 문장: %s",
		today.Format(time.DateOnly), weekdayKoShort[today.Weekday()], sentence)

	bin, err := claudeBin()
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, bin,
		"-p", "--model", "haiku", "--output-format", "json",
		"--tools", "", "--strict-mcp-config", "--no-session-persistence", "--setting-sources", "",
		"--settings", `{"alwaysThinkingEnabled":false}`,
		"--system-prompt", prompt,
		msg)
	// stdin을 막지 않으면 claude -p가 stdin을 읽어 프롬프트에 섞는다.
	cmd.Stdin = nil
	var stderr strings.Builder
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("claude 실행 실패: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	var m meta
	if err := json.Unmarshal(out, &m); err != nil {
		return "", fmt.Errorf("claude 응답이 JSON이 아니다: %w", err)
	}
	if m.IsError || m.Subtype != "success" {
		return "", fmt.Errorf("claude 실패: subtype=%s", m.Subtype)
	}
	return m.Result, nil
}

// parseResult는 모델 답변 문자열에서 JSON만 잘라 읽는다.
// "코드펜스 금지"를 지시해도 ```json 으로 감싸서 오므로 첫 { ~ 마지막 } 만 쓴다.
func parseResult(s string) (Result, error) {
	a, b := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if a < 0 || b < a {
		return Result{}, fmt.Errorf("답변에 JSON이 없다: %q", truncate(s, 80))
	}
	var r Result
	if err := json.Unmarshal([]byte(s[a:b+1]), &r); err != nil {
		return Result{}, fmt.Errorf("답변 JSON 파싱 실패: %w (%q)", err, truncate(s, 80))
	}
	if r.Title == "" {
		return Result{}, fmt.Errorf("title이 비었다: %q", truncate(s, 80))
	}
	return r, nil
}

// parseResults는 답변에서 JSON 배열을 잘라 읽는다.
// 여러 개를 요청해도 객체 하나로 답하는 일이 있어, 그때는 한 건으로 받는다.
func parseResults(s string) ([]Result, error) {
	a := strings.IndexAny(s, "[{")
	if a < 0 {
		return nil, fmt.Errorf("답변에 JSON이 없다: %q", truncate(s, 80))
	}
	if s[a] == '{' {
		r, err := parseResult(s)
		if err != nil {
			return nil, err
		}
		return []Result{r}, nil
	}

	b := strings.LastIndex(s, "]")
	if b < a {
		return nil, fmt.Errorf("답변의 배열이 닫히지 않았다: %q", truncate(s, 80))
	}
	var rs []Result
	if err := json.Unmarshal([]byte(s[a:b+1]), &rs); err != nil {
		return nil, fmt.Errorf("답변 JSON 파싱 실패: %w (%q)", err, truncate(s, 80))
	}

	var out []Result
	for _, r := range rs {
		// 빈 객체나 title 없는 항목을 끼워 넣는 일이 있다. 조용히 버린다.
		if strings.TrimSpace(r.Title) == "" {
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("업무를 찾지 못했다: %q", truncate(s, 80))
	}
	return out, nil
}

// Fallback은 파싱이 실패했을 때 쓸 기본값.
// 조용히 넘어가지 않고 호출한 쪽이 사용자에게 알린다.
func Fallback(sentence string) Result {
	return Result{
		Title:    truncate(strings.TrimSpace(sentence), 40),
		Due:      Due{Kind: "none"},
		Priority: "normal",
		Unsure:   true,
	}
}

// truncate는 룬 단위로 자른다 (한글이 깨지지 않게).
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// LookupClaude는 claude CLI가 설치돼 있는지 본다.
func LookupClaude() error {
	_, err := claudeBin()
	return err
}

// claudeBin은 claude CLI 경로를 찾는다.
// PATH만 봐서는 안 된다: 메뉴바 앱이 launchd로 뜨면 PATH가
// /usr/bin:/bin:/usr/sbin:/sbin 뿐이라 ~/.local/bin/claude를 놓친다 (실측 2026-09-27).
// 앱이 tasklet을 찾는 순서(macos/Sources/Model.swift)와 같은 방식이다.
func claudeBin() (string, error) {
	if p := os.Getenv("CLAUDE_BIN"); p != "" {
		if isExecutable(p) {
			return p, nil
		}
		return "", fmt.Errorf("CLAUDE_BIN이 실행 파일이 아니다: %s", p)
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p, nil
	}
	candidates := []string{"/opt/homebrew/bin/claude", "/usr/local/bin/claude"}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append([]string{filepath.Join(home, ".local", "bin", "claude")}, candidates...)
	}
	for _, p := range candidates {
		if isExecutable(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("claude CLI를 찾을 수 없다 (PATH=%s, %s 에도 없음)",
		os.Getenv("PATH"), strings.Join(candidates, " "))
}

func isExecutable(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0
}
