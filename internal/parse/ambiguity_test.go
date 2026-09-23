package parse

import (
	"strings"
	"testing"
	"time"
)

func TestReasons(t *testing.T) {
	today := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC) // 화요일

	tests := []struct {
		name     string
		sentence string
		result   Result
		want     []string // 각 사유에 포함돼야 할 조각
	}{
		{"기한 없고 날짜 표현도 없음 → 확인 안 함",
			"시간 될 때 로그 대시보드 개선",
			Result{Title: "로그 대시보드 개선", Due: Due{Kind: "none"}}, nil},

		{"A1 요일이 있는데 기한을 못 찾음",
			"금요일이나 월요일까지 디자인 시안 검토",
			Result{Title: "디자인 시안 검토", Due: Due{Kind: "none"}}, []string{"기한을 찾지 못했"}},

		{"A1 'N일' 표현",
			"2일 걸리는 마이그레이션 작업",
			Result{Title: "마이그레이션 작업", Due: Due{Kind: "none"}}, []string{"기한을 찾지 못했"}},

		{"A1 '다음 주' 표현",
			"다음 주 중에 회고 준비",
			Result{Title: "회고 준비", Due: Due{Kind: "none"}}, []string{"기한을 찾지 못했"}},

		{"A2 지난 날짜라 다음 달로 넘김",
			"15일까지 서버 인증서 갱신",
			Result{Title: "서버 인증서 갱신", Due: Due{Kind: "date", Value: "15"}}, []string{"이미 지난 날짜"}},

		{"A3 오늘과 같은 요일",
			"화요일까지 회원가입 업무",
			Result{Title: "회원가입 업무", Due: Due{Kind: "weekday", Value: "tue"}}, []string{"오늘이라 다음 주"}},

		{"A4 Haiku가 unsure",
			"그거 처리해두기",
			Result{Title: "그거 처리", Due: Due{Kind: "none"}, Unsure: true}, []string{"여러 뜻"}},

		{"A1 + A4 둘 다",
			"이번 주 중에 그거",
			Result{Title: "그거", Due: Due{Kind: "none"}, Unsure: true}, []string{"기한을 찾지 못했", "여러 뜻"}},

		{"평범한 기한은 확인 안 함",
			"수요일까지 배포 체크리스트 공유",
			Result{Title: "배포 체크리스트 공유", Due: Due{Kind: "weekday", Value: "wed"}}, nil},

		{"기한 없음 + 날짜 표현 없음 + unsure만",
			"뭔가 정리",
			Result{Title: "정리", Due: Due{Kind: "none"}, Unsure: true}, []string{"여러 뜻"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := ResolveDue(tt.result.Due, today)
			if err != nil {
				t.Fatalf("ResolveDue: %v", err)
			}
			got := Reasons(tt.sentence, tt.result, resolved)
			if len(got) != len(tt.want) {
				t.Fatalf("사유 %d개 %q, want %d개 %q", len(got), got, len(tt.want), tt.want)
			}
			for i, want := range tt.want {
				if !strings.Contains(got[i], want) {
					t.Errorf("사유[%d] = %q, %q를 포함해야 한다", i, got[i], want)
				}
			}
		})
	}
}

func TestParseResult(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Result
		err   bool
	}{
		{"코드펜스로 감싼 실제 응답",
			"```json\n{\n  \"title\": \"회원가입 업무 진행\",\n  \"from\": \"성민우 선임\",\n  \"due\": {\"kind\": \"weekday\", \"value\": \"tue\"},\n  \"priority\": \"normal\",\n  \"unsure\": false\n}\n```",
			Result{Title: "회원가입 업무 진행", From: "성민우 선임", Due: Due{"weekday", "tue"}, Priority: "normal"}, false},

		{"코드펜스 없음",
			`{"title":"주간보고 작성","from":null,"due":{"kind":"eow","value":null},"priority":"normal","unsure":true}`,
			Result{Title: "주간보고 작성", Due: Due{Kind: "eow"}, Priority: "normal", Unsure: true}, false},

		{"설명이 앞뒤에 붙은 경우",
			"다음과 같습니다:\n{\"title\":\"PR 리뷰\",\"due\":{\"kind\":\"none\",\"value\":null},\"priority\":\"normal\"}\n이상입니다.",
			Result{Title: "PR 리뷰", Due: Due{Kind: "none"}, Priority: "normal"}, false},

		{"JSON이 아예 없음", "잘 모르겠습니다", Result{}, true},
		{"깨진 JSON", `{"title": "어쩌고"`, Result{}, true},
		{"title이 빈 값", `{"title":"","due":{"kind":"none","value":null}}`, Result{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseResult(tt.input)
			if tt.err {
				if err == nil {
					t.Fatalf("에러를 기대했지만 %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("예상치 못한 에러: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestFallback(t *testing.T) {
	long := strings.Repeat("가", 60)
	got := Fallback(long)
	if len([]rune(got.Title)) != 40 {
		t.Errorf("title 길이 = %d, want 40 (앞 40자)", len([]rune(got.Title)))
	}
	if got.Due.Kind != "none" || !got.Unsure {
		t.Errorf("파싱 실패 기본값이 아니다: %+v", got)
	}
}
