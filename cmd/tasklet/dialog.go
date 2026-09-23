package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// errCancelled는 사용자가 입력창·확인창에서 취소를 눌렀을 때.
var errCancelled = errors.New("취소")

// ask는 한 줄 입력창을 띄우고 입력값을 돌려준다.
// def가 있으면 그 값이 채워진 채로 뜬다 (수정할 때 원래 문장).
func ask(prompt, def string) (string, error) {
	script := fmt.Sprintf(`display dialog %s with title "tasklet" default answer %s buttons {"취소", "저장"} default button "저장"`,
		quote(prompt), quote(def))
	out, err := osascript(script + `
return text returned of result`)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// confirm은 예/아니오 확인창. 예면 true.
func confirm(message string) (bool, error) {
	out, err := osascript(fmt.Sprintf(
		`display dialog %s with title "tasklet" buttons {"아니오", "예"} default button "예"
return button returned of result`, quote(message)))
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "예", nil
}

// notify는 macOS 알림 한 줄. 실패해도 무시한다 (알림이 안 떠도 본 작업은 끝나야 한다).
func notify(message string) {
	_, _ = osascript(fmt.Sprintf(`display notification %s with title "tasklet"`, quote(message)))
}

func osascript(script string) (string, error) {
	cmd := exec.Command("osascript", "-e", script)
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		// 취소 버튼을 누르면 osascript가 -128로 끝난다.
		if errors.As(err, &ee) && strings.Contains(string(ee.Stderr), "-128") {
			return "", errCancelled
		}
		return "", fmt.Errorf("osascript 실패: %w", err)
	}
	return string(out), nil
}

// quote는 AppleScript 문자열 리터럴로 감싼다.
func quote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `" & return & "`)
	return `"` + s + `"`
}
