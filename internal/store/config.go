package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config는 ~/.tasklet/config.json.
type Config struct {
	Theme            string `json:"theme"`
	Mascot           string `json:"mascot"`
	WeekendDueBucket string `json:"weekend_due_bucket"`
}

// 기본값. 파일이 없거나 값이 이상하면 이걸 쓴다 (에러로 메뉴를 깨지 않는다).
var defaultConfig = Config{Theme: "mono", Mascot: "pig", WeekendDueBucket: "later"}

var (
	Themes  = []string{"tide", "dusk", "moss", "pop", "mono"}
	Mascots = []string{"cat", "dog", "bear", "pig", "duck", "owl", "octopus"}
)

func configPath() string { return filepath.Join(Dir(), "config.json") }

// LoadConfig는 설정을 읽는다. 읽기 실패·모르는 값은 조용히 기본값으로 대체한다.
func LoadConfig() Config {
	c := defaultConfig
	b, err := os.ReadFile(configPath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// 파일이 깨졌어도 메뉴는 떠야 한다.
			return c
		}
		return c
	}
	var got Config
	if err := json.Unmarshal(b, &got); err != nil {
		return c
	}
	if contains(Themes, got.Theme) {
		c.Theme = got.Theme
	}
	if contains(Mascots, got.Mascot) {
		c.Mascot = got.Mascot
	}
	if got.WeekendDueBucket == "later" || got.WeekendDueBucket == "this_week" {
		c.WeekendDueBucket = got.WeekendDueBucket
	}
	return c
}

// SaveConfig는 임시파일 → rename으로 바꾼다. 쓰는 곳이 theme·mascot뿐이라 잠금은 두지 않는다.
func SaveConfig(c Config) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	tmp, err := os.CreateTemp(Dir(), "config-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), configPath())
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
