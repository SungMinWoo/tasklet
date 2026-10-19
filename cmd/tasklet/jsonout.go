package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/SungMinWoo/tasklet/internal/render"
	"github.com/SungMinWoo/tasklet/internal/store"
)

// SwiftUI 팝오버가 읽는 형식. 화면이 날짜·버킷 계산을 다시 하지 않도록
// 라벨과 버킷까지 Go가 계산해서 내려준다.
type jsonTask struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Requester string `json:"requester"`
	DueAt     string `json:"due_at"`
	DueLabel  string `json:"due_label"` // "-2일", "오늘", "9/29(화)", "기한 없음"
	Bucket    string `json:"bucket"`    // past | today | tomorrow | this_week | later
	Raw       string `json:"raw"`
	// 등록·완료 시각. 지금은 '완료' 구역이 쓰고, 나중에 기간으로 찾을 때도 이 값으로 찾는다.
	CreatedAt string `json:"created_at"`        // RFC3339
	DoneAt    string `json:"done_at,omitempty"` // RFC3339, 완료한 것만
}

type jsonState struct {
	Counts  map[string]int `json:"counts"`
	Tasks   []jsonTask     `json:"tasks"`
	Done    []jsonTask     `json:"done"` // 오늘 완료한 것 (최근 순). 되돌리기용
	Theme   string         `json:"theme"`
	Mascot  string         `json:"mascot"`
	Reasons []string       `json:"reasons"` // 확인이 필요한 이유 (add·edit 직후)
	Changed []int          `json:"changed"` // 방금 추가·수정된 id (한 번에 넣기는 여러 개)
}

var bucketName = map[render.BucketKey]string{
	render.Past: "past", render.Today: "today", render.Tomorrow: "tomorrow",
	render.ThisWeek: "this_week", render.Later: "later",
}

// cmdState는 화면에 필요한 모든 것을 한 번에 내려준다 (tasklet state --json).
func cmdState() error {
	f, err := store.Read()
	if err != nil {
		return err
	}
	return printState(f, nil)
}

// changed에는 방금 바뀐 id를 넘긴다. 없으면 생략한다.
func printState(f *store.File, reasons []string, changed ...int) error {
	cfg := store.LoadConfig()
	now := time.Now()
	buckets := render.Buckets(f.Tasks, now, cfg.WeekendDueBucket)

	st := jsonState{
		Counts:  map[string]int{},
		Tasks:   []jsonTask{}, // null이 아니라 빈 배열로
		Done:    []jsonTask{},
		Theme:   cfg.Theme,
		Mascot:  cfg.Mascot,
		Reasons: reasons,
		Changed: changed,
	}
	for _, key := range render.BucketOrder {
		st.Counts[bucketName[key]] = len(buckets[key])
		for _, t := range buckets[key] {
			st.Tasks = append(st.Tasks, jsonTaskOf(t, bucketName[key], now))
		}
	}
	st.Done = doneToday(f.Tasks, now)

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(st)
}

func jsonTaskOf(t store.Task, bucket string, now time.Time) jsonTask {
	jt := jsonTask{
		ID:        t.ID,
		Title:     t.Title,
		DueLabel:  dueLabel(t.DueAt, now),
		Bucket:    bucket,
		Raw:       t.Raw,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
	if t.Requester != nil {
		jt.Requester = *t.Requester
	}
	if t.DueAt != nil {
		jt.DueAt = *t.DueAt
	}
	if t.DoneAt != nil {
		jt.DoneAt = t.DoneAt.Format(time.RFC3339)
	}
	return jt
}

// doneToday는 오늘 완료한 것을 최근 순으로 준다.
// 완료 이력 전체를 내려보내면 쓸수록 무거워지므로 화면에 필요한 만큼만 준다.
// 기간으로 찾는 기능이 생기면 그때는 이 함수 대신 검색 명령을 따로 둔다.
func doneToday(tasks []store.Task, now time.Time) []jsonTask {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	out := []jsonTask{} // null이 아니라 빈 배열로
	for _, t := range tasks {
		if t.Status != store.StatusDone || t.DoneAt == nil || t.DoneAt.Before(today) {
			continue
		}
		out = append(out, jsonTaskOf(t, "done", now))
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].DoneAt > out[j].DoneAt })
	return out
}

type jsonDay struct {
	Date  string     `json:"date"`  // "2026-10-17"
	Label string     `json:"label"` // "10/17(토)", "10/19(월) 오늘"
	Tasks []jsonTask `json:"tasks"`
}

// cmdHistory는 일을 날짜별로 보여준다 (tasklet history [--created] [--from 날짜] [--to 날짜] [--json]).
// 기본은 완료한 날 기준, --created면 등록한 날 기준.
// 기간은 기본 최근 1개월(한 달 전 ~ 오늘). 이력이 쌓여도 화면에 넘기는 양이 일정하게.
// 화면에서는 팝오버의 '기록' 탭이 부르고, JSON에는 두 기준을 다 담는다.
func cmdHistory(args []string) error {
	now := time.Now()
	from, to, err := historyRange(args, now)
	if err != nil {
		return err
	}
	f, err := store.Read()
	if err != nil {
		return err
	}
	weekend := store.LoadConfig().WeekendDueBucket
	created := render.Between(render.CreatedByDay(f.Tasks, now.Location()), from, to)
	done := render.Between(render.DoneByDay(f.Tasks, now.Location()), from, to)

	if jsonMode {
		out := struct {
			From    string    `json:"from"` // 실제로 적용한 기간 (기본값이면 Go가 정한 값)
			To      string    `json:"to"`
			Created []jsonDay `json:"created"` // 등록한 날별. 완료한 것도 들어간다
			Done    []jsonDay `json:"done"`    // 완료한 날별
		}{
			From: from.Format(time.DateOnly), To: to.Format(time.DateOnly),
			Created: jsonDays(created, now, weekend), Done: jsonDays(done, now, weekend),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	byCreated := slices.Contains(args, "--created")
	days := done
	if byCreated {
		days = created
	}
	fmt.Printf("%s ~ %s\n\n", from.Format(time.DateOnly), to.Format(time.DateOnly))
	if len(days) == 0 {
		fmt.Println("기록 없음")
		return nil
	}
	for i, d := range days {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("%s  %d건\n", render.DayLabel(d.Date, now), len(d.Tasks))
		for _, t := range d.Tasks {
			at, mark := t.CreatedAt, "○"
			if t.Status == store.StatusDone {
				mark = "✓"
			}
			if !byCreated {
				at = *t.DoneAt
			}
			who := ""
			if t.Requester != nil && *t.Requester != "" {
				who = "  ·  " + *t.Requester
			}
			fmt.Printf("  %s %s  %3d  %s%s\n", at.In(now.Location()).Format("15:04"), mark, t.ID, t.Title, who)
		}
	}
	return nil
}

// historyRange는 --from·--to를 읽는다. 없으면 한 달 전 ~ 오늘.
func historyRange(args []string, now time.Time) (from, to time.Time, err error) {
	to, from = now, render.MonthBefore(now)
	for _, flag := range []struct {
		name string
		dst  *time.Time
	}{{"--from", &from}, {"--to", &to}} {
		v, given := flagValue(args, flag.name)
		if !given {
			continue
		}
		d, perr := time.ParseInLocation(time.DateOnly, v, now.Location())
		if perr != nil {
			return from, to, fmt.Errorf("%s 값은 2026-10-19 형식이어야 한다: %q", flag.name, v)
		}
		*flag.dst = d
	}
	if to.Before(from) {
		return from, to, fmt.Errorf("기간이 거꾸로다: %s ~ %s", from.Format(time.DateOnly), to.Format(time.DateOnly))
	}
	return from, to, nil
}

// jsonDays는 날짜 묶음을 화면 형식으로. 버킷은 완료면 done, 아니면 지금 기준 버킷 (색 표시용).
func jsonDays(days []render.Day, now time.Time, weekend string) []jsonDay {
	out := []jsonDay{} // null이 아니라 빈 배열로
	for _, d := range days {
		jd := jsonDay{Date: d.Date.Format(time.DateOnly), Label: render.DayLabel(d.Date, now)}
		for _, t := range d.Tasks {
			bucket := "done"
			if t.Status != store.StatusDone {
				bucket = bucketName[render.BucketOf(t, now, weekend)]
			}
			jd.Tasks = append(jd.Tasks, jsonTaskOf(t, bucket, now))
		}
		out = append(out, jd)
	}
	return out
}

// jsonError는 화면이 읽을 수 있는 실패 형식.
func jsonError(err error) {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	fmt.Println(string(b))
}
