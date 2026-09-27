package main

import (
	"encoding/json"
	"fmt"
	"os"
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

// jsonError는 화면이 읽을 수 있는 실패 형식.
func jsonError(err error) {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	fmt.Println(string(b))
}
