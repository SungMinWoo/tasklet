// Package store는 ~/.tasklet/tasks.json을 읽고 쓴다 (DESIGN.md 5장).
// 쓰기는 flock으로 잠그고, 임시파일 → rename으로 교체한다.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

const (
	StatusTodo  = "todo"
	StatusDoing = "doing"
	StatusDone  = "done"
)

// Task는 할 일 하나. 없는 값은 JSON에서 null.
type Task struct {
	ID        int        `json:"id"`
	Title     string     `json:"title"`
	Requester *string    `json:"requester"`
	DueAt     *string    `json:"due_at"` // "2026-09-25", 기한 없으면 nil
	Status    string     `json:"status"`
	Raw       string     `json:"raw"`
	CreatedAt time.Time  `json:"created_at"`
	DoneAt    *time.Time `json:"done_at"`
}

// File은 tasks.json 전체.
type File struct {
	NextID int    `json:"next_id"`
	Tasks  []Task `json:"tasks"`
}

// Dir은 데이터 디렉토리. 테스트는 TASKLET_HOME으로 바꾼다.
func Dir() string {
	if d := os.Getenv("TASKLET_HOME"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".tasklet"
	}
	return filepath.Join(home, ".tasklet")
}

func tasksPath() string { return filepath.Join(Dir(), "tasks.json") }
func lockPath() string  { return filepath.Join(Dir(), "tasks.lock") }

// Read는 잠그지 않고 읽는다 (menu·list용). 파일이 없으면 빈 File.
func Read() (*File, error) {
	b, err := os.ReadFile(tasksPath())
	if errors.Is(err, os.ErrNotExist) {
		return &File{NextID: 1}, nil
	}
	if err != nil {
		return nil, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s 파싱 실패: %w", tasksPath(), err)
	}
	if f.NextID < 1 {
		f.NextID = 1
	}
	return &f, nil
}

// Update는 잠금을 잡고 "읽기 → fn → 쓰기"를 한 번에 한다.
// fn이 에러를 내면 아무것도 쓰지 않는다.
func Update(fn func(*File) error) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	unlock, err := lock()
	if err != nil {
		return err
	}
	defer unlock()

	f, err := Read()
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	return write(f)
}

// Add는 새 할 일을 넣고 부여된 id를 돌려준다.
func (f *File) Add(t Task) int {
	t.ID = f.NextID
	f.NextID++
	if t.Status == "" {
		t.Status = StatusTodo
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	f.Tasks = append(f.Tasks, t)
	return t.ID
}

// Find는 id로 할 일을 찾는다. 수정하려면 반환된 포인터를 고친다.
func (f *File) Find(id int) *Task {
	for i := range f.Tasks {
		if f.Tasks[i].ID == id {
			return &f.Tasks[i]
		}
	}
	return nil
}

// Complete는 완료 처리한다. 이미 완료면 아무것도 하지 않는다.
func (f *File) Complete(id int, now time.Time) error {
	t := f.Find(id)
	if t == nil {
		return fmt.Errorf("%d번 할 일이 없다", id)
	}
	if t.Status == StatusDone {
		return nil
	}
	t.Status = StatusDone
	t.DoneAt = &now
	return nil
}

// Uncomplete는 완료를 취소한다. 실수로 완료한 것을 되돌릴 때 쓴다.
// 완료한 적이 없으면 아무것도 하지 않는다.
func (f *File) Uncomplete(id int) error {
	t := f.Find(id)
	if t == nil {
		return fmt.Errorf("%d번 할 일이 없다", id)
	}
	t.Status = StatusTodo
	t.DoneAt = nil
	return nil
}

// Delete는 할 일을 목록에서 지운다. 되돌릴 수 없다.
// next_id는 줄이지 않는다 — 지운 id를 다시 쓰면 완료 이력과 헷갈린다.
func (f *File) Delete(id int) error {
	for i := range f.Tasks {
		if f.Tasks[i].ID == id {
			f.Tasks = append(f.Tasks[:i], f.Tasks[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("%d번 할 일이 없다", id)
}

// write는 같은 디렉토리 임시파일에 쓴 뒤 rename으로 바꾼다.
// 읽는 쪽이 반쪽 파일을 보지 않게 하기 위함.
func write(f *File) error {
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(Dir(), "tasks-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // rename에 성공하면 사라진 뒤라 무해

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), tasksPath())
}

// lock은 tasks.lock에 배타 잠금을 건다. 반환된 함수로 푼다.
func lock() (func(), error) {
	fd, err := os.OpenFile(lockPath(), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_EX); err != nil {
		fd.Close()
		return nil, fmt.Errorf("잠금 실패: %w", err)
	}
	return func() {
		syscall.Flock(int(fd.Fd()), syscall.LOCK_UN)
		fd.Close()
	}, nil
}
