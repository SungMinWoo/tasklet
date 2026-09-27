package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/SungMinWoo/tasklet/internal/parse"
	"github.com/SungMinWoo/tasklet/internal/store"
)

// 한 번에 넣기 (DESIGN.md 4장·6장).
// 줄바꿈으로 나눠 줄마다 Haiku를 동시에 부른다 — 전체 소요는 가장 느린 한 줄과 같다.
// 한 줄에 서로 다른 업무가 섞여 있으면 그 줄에서 여러 건이 나온다.
const (
	maxBatchLines    = 30 // 붙여넣기 실수로 수백 줄이 들어오는 것을 막는다
	maxBatchParallel = 4  // claude CLI는 프로세스 하나가 무겁다
)

// cmdAddBatch는 여러 줄을 한 번에 추가한다.
// 텍스트는 인자로 받고(팝오버), 없으면 표준입력에서 읽는다(파이프·파일).
func cmdAddBatch(args []string) error {
	text := strings.Join(args, "\n")
	if strings.TrimSpace(text) == "" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("표준입력을 읽지 못했다: %w", err)
		}
		text = string(b)
	}

	lines := splitLines(text)
	if len(lines) == 0 {
		return fmt.Errorf(`추가할 줄이 없다. 예: tasklet add --batch "화요일까지 회원가입 개발\n로그 대시보드 개선"`)
	}
	if len(lines) > maxBatchLines {
		return fmt.Errorf("한 번에 %d줄까지다 (%d줄 들어옴)", maxBatchLines, len(lines))
	}
	if err := parse.LookupClaude(); err != nil {
		return err
	}

	parsed := interpretLines(lines)

	var (
		ids     []int
		reasons []string
	)
	if err := store.Update(func(f *store.File) error {
		for _, p := range parsed {
			for _, res := range p.results {
				resolved, rerr := parse.ResolveDue(res.Due, time.Now())
				if rerr != nil {
					// 스키마에 없는 kind가 오면 기한 없음으로 낮추고 애매한 것으로 표시한다
					// (add 한 건과 같은 처리, DESIGN.md 6장 방어 코드).
					fmt.Fprintf(os.Stderr, "%q 기한을 해석하지 못했다: %v\n", p.line, rerr)
					resolved = parse.Resolved{}
					res.Due = parse.Due{Kind: "none"}
					res.Unsure = true
				}
				// 한 줄에서 여러 건이 나왔으면 원문 대신 제목을 남긴다.
				// 그러지 않으면 '고쳐 쓰기'가 같은 줄을 다시 파싱해 형제 항목과 합쳐진다.
				raw := p.line
				if len(p.results) > 1 {
					raw = res.Title
				}
				ids = append(ids, f.Add(taskOf(raw, res, resolved)))
				for _, r := range parse.Reasons(p.line, res, resolved) {
					reasons = append(reasons, res.Title+" — "+r)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	f, err := store.Read()
	if err != nil {
		return err
	}
	if jsonMode {
		return printState(f, reasons, ids...)
	}
	fmt.Printf("%d건 추가\n", len(ids))
	now := time.Now()
	for _, id := range ids {
		if t := f.Find(id); t != nil {
			fmt.Printf("  %3d  %s\n", id, describe(*t, now))
		}
	}
	// 확인창은 팝오버가 띄운다. 터미널에서는 이유만 알린다 (DESIGN.md 8장).
	for _, r := range reasons {
		fmt.Printf("  · %s\n", r)
	}
	return nil
}

// lineResults는 줄 하나에서 나온 업무들. 입력 순서를 그대로 유지한다.
type lineResults struct {
	line    string
	results []parse.Result
}

// interpretLines는 줄마다 Haiku를 동시에 부른다 (동시 실행은 maxBatchParallel개까지).
// 한 줄이 실패해도 그 줄만 기본값으로 내려가고 나머지는 살린다.
func interpretLines(lines []string) []lineResults {
	out := make([]lineResults, len(lines))
	sem := make(chan struct{}, maxBatchParallel)
	var wg sync.WaitGroup
	today := time.Now()

	for i, line := range lines {
		wg.Add(1)
		go func(i int, line string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			results, err := parse.ExtractMany(context.Background(), line, today)
			if err != nil {
				// 조용히 실패하지 않는다: 기한 없이 저장하고 알린다 (DESIGN.md 6장 방어 코드 3번).
				fmt.Fprintf(os.Stderr, "%q 파싱 실패 — 기한 없이 저장한다: %v\n", line, err)
				results = []parse.Result{parse.Fallback(line)}
			}
			out[i] = lineResults{line: line, results: results}
		}(i, line)
	}
	wg.Wait()
	return out
}

// splitLines는 줄바꿈으로 나누고 빈 줄을 버린다.
func splitLines(text string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// takeBatchFlag는 어디에 있어도 --batch를 빼낸다.
func takeBatchFlag(args []string) ([]string, bool) {
	rest, found := args[:0:0], false
	for _, a := range args {
		if a == "--batch" {
			found = true
			continue
		}
		rest = append(rest, a)
	}
	return rest, found
}
