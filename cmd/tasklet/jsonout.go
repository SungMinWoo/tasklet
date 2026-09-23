package main

import (
	"encoding/json"
	"fmt"
	"os"
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
}

type jsonState struct {
	Counts  map[string]int `json:"counts"`
	Tasks   []jsonTask     `json:"tasks"`
	Theme   string         `json:"theme"`
	Mascot  string         `json:"mascot"`
	Reasons []string       `json:"reasons"` // 확인이 필요한 이유 (add·edit 직후)
	Changed int            `json:"changed"` // 방금 추가·수정된 id
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
	return printState(f, nil, 0)
}

func printState(f *store.File, reasons []string, changed int) error {
	cfg := store.LoadConfig()
	now := time.Now()
	buckets := render.Buckets(f.Tasks, now, cfg.WeekendDueBucket)

	st := jsonState{
		Counts:  map[string]int{},
		Tasks:   []jsonTask{}, // null이 아니라 빈 배열로
		Theme:   cfg.Theme,
		Mascot:  cfg.Mascot,
		Reasons: reasons,
		Changed: changed,
	}
	for _, key := range render.BucketOrder {
		st.Counts[bucketName[key]] = len(buckets[key])
		for _, t := range buckets[key] {
			jt := jsonTask{
				ID:       t.ID,
				Title:    t.Title,
				DueLabel: dueLabel(t.DueAt, now),
				Bucket:   bucketName[key],
				Raw:      t.Raw,
			}
			if t.Requester != nil {
				jt.Requester = *t.Requester
			}
			if t.DueAt != nil {
				jt.DueAt = *t.DueAt
			}
			st.Tasks = append(st.Tasks, jt)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(st)
}

// jsonError는 화면이 읽을 수 있는 실패 형식.
func jsonError(err error) {
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	fmt.Println(string(b))
}
