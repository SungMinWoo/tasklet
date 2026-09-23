package store

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReadMissingFile(t *testing.T) {
	t.Setenv("TASKLET_HOME", t.TempDir())
	f, err := Read()
	if err != nil {
		t.Fatalf("파일이 없을 때는 빈 목록이어야 한다: %v", err)
	}
	if f.NextID != 1 || len(f.Tasks) != 0 {
		t.Fatalf("got %+v", f)
	}
}

func TestAddFindComplete(t *testing.T) {
	t.Setenv("TASKLET_HOME", t.TempDir())
	req, due := "성민우 선임", "2026-09-29"
	var id int
	if err := Update(func(f *File) error {
		id = f.Add(Task{Title: "회원가입 기능 개발", Requester: &req, DueAt: &due, Raw: "원문"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("id = %d, want 1", id)
	}

	now := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	if err := Update(func(f *File) error { return f.Complete(id, now) }); err != nil {
		t.Fatal(err)
	}

	f, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	got := f.Find(id)
	if got == nil {
		t.Fatal("저장된 할 일을 못 찾음")
	}
	if got.Status != StatusDone || got.DoneAt == nil || !got.DoneAt.Equal(now) {
		t.Errorf("완료 처리 안 됨: %+v", got)
	}
	if got.Requester == nil || *got.Requester != req || got.DueAt == nil || *got.DueAt != due {
		t.Errorf("필드 유실: %+v", got)
	}
	if got.Raw != "원문" || got.CreatedAt.IsZero() {
		t.Errorf("raw·created_at 유실: %+v", got)
	}
	if f.NextID != 2 {
		t.Errorf("next_id = %d, want 2", f.NextID)
	}
}

func TestCompleteUnknownID(t *testing.T) {
	t.Setenv("TASKLET_HOME", t.TempDir())
	err := Update(func(f *File) error { return f.Complete(99, time.Now()) })
	if err == nil {
		t.Fatal("없는 id는 에러여야 한다")
	}
	if _, statErr := os.Stat(filepath.Join(Dir(), "tasks.json")); statErr == nil {
		t.Error("fn이 에러를 내면 파일을 쓰지 않아야 한다")
	}
}

func TestNullsInJSON(t *testing.T) {
	t.Setenv("TASKLET_HOME", t.TempDir())
	if err := Update(func(f *File) error {
		f.Add(Task{Title: "기한 없는 일", Raw: "시간 될 때"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(Dir(), "tasks.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"requester": null`, `"due_at": null`, `"done_at": null`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("%s 가 JSON에 없다:\n%s", want, b)
		}
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("사람이 고칠 수 있는 JSON이어야 한다: %v", err)
	}
}

// 같은 프로세스 안에서 여러 고루틴이 동시에 써도 유실되지 않아야 한다.
func TestConcurrentUpdates(t *testing.T) {
	t.Setenv("TASKLET_HOME", t.TempDir())
	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- Update(func(f *File) error {
				f.Add(Task{Title: "일", Raw: "원문"})
				return nil
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	f, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Tasks) != n {
		t.Fatalf("할 일 %d개, want %d — 동시 쓰기에서 유실", len(f.Tasks), n)
	}
	seen := map[int]bool{}
	for _, task := range f.Tasks {
		if seen[task.ID] {
			t.Fatalf("id %d 중복", task.ID)
		}
		seen[task.ID] = true
	}
}

// 다른 프로세스가 동시에 써도 유실되지 않아야 한다 (flock의 진짜 목적).
func TestConcurrentProcesses(t *testing.T) {
	if os.Getenv("TASKLET_CHILD") == "1" {
		if err := Update(func(f *File) error {
			time.Sleep(20 * time.Millisecond) // 잠금을 잡은 채로 겹치게 만든다
			f.Add(Task{Title: "자식이 넣은 일", Raw: "원문"})
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	t.Setenv("TASKLET_HOME", dir)

	const n = 5
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=TestConcurrentProcesses")
			cmd.Env = append(os.Environ(), "TASKLET_CHILD=1", "TASKLET_HOME="+dir)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("자식 프로세스 실패: %v\n%s", err, out)
			}
		}()
	}
	wg.Wait()

	f, err := Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Tasks) != n {
		t.Fatalf("할 일 %d개, want %d — 프로세스 간 잠금이 안 걸렸다", len(f.Tasks), n)
	}
}
