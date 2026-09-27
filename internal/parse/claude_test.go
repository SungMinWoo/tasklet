package parse

import "testing"

// 모델 답변에서 JSON을 잘라 읽는 부분만 본다 (claude 호출은 하지 않는다).
func TestParseResults(t *testing.T) {
	tests := []struct {
		name   string
		answer string
		want   []string // 뽑혀야 하는 title 목록
		err    bool
	}{
		{
			name:   "배열 두 건",
			answer: `[{"title":"회원가입 개발","due":{"kind":"weekday","value":"tue"}},{"title":"대심분석","due":{"kind":"none"}}]`,
			want:   []string{"회원가입 개발", "대심분석"},
		},
		{
			name:   "코드펜스로 감싸 옴",
			answer: "```json\n[{\"title\":\"로그 대시보드 개선\",\"due\":{\"kind\":\"none\"}}]\n```",
			want:   []string{"로그 대시보드 개선"},
		},
		{
			name:   "배열 앞뒤에 설명이 붙어 옴",
			answer: `네, 아래와 같습니다. [{"title":"헬스체크 제한","due":{"kind":"none"}}] 이상입니다.`,
			want:   []string{"헬스체크 제한"},
		},
		{
			name:   "배열 대신 객체 하나로 답함",
			answer: `{"title":"회원가입 개발","due":{"kind":"none"}}`,
			want:   []string{"회원가입 개발"},
		},
		{
			name:   "title 없는 항목은 버린다",
			answer: `[{"title":"회원가입 개발","due":{"kind":"none"}},{},{"title":"  ","due":{"kind":"none"}}]`,
			want:   []string{"회원가입 개발"},
		},
		{
			name:   "요청자·기한까지 읽는다",
			answer: `[{"title":"대시보드 개발","from":"이사님","due":{"kind":"weekday","value":"thu"},"unsure":true}]`,
			want:   []string{"대시보드 개발"},
		},
		{name: "JSON이 없다", answer: "무슨 말인지 모르겠습니다", err: true},
		{name: "빈 배열", answer: `[]`, err: true},
		{name: "title이 다 비었다", answer: `[{},{"title":""}]`, err: true},
		{name: "배열이 닫히지 않았다", answer: `[{"title":"회원가입 개발"`, err: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseResults(tc.answer)
			if tc.err {
				if err == nil {
					t.Fatalf("에러여야 한다: got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("%d건, want %d건: %+v", len(got), len(tc.want), got)
			}
			for i, want := range tc.want {
				if got[i].Title != want {
					t.Errorf("[%d] title = %q, want %q", i, got[i].Title, want)
				}
			}
		})
	}
}

// 객체 하나로 답했을 때도 필드가 유실되지 않아야 한다.
func TestParseResultsKeepsFields(t *testing.T) {
	got, err := parseResults(`[{"title":"대시보드 개발","from":"이사님","due":{"kind":"weekday","value":"thu"},"priority":"high","unsure":true}]`)
	if err != nil {
		t.Fatal(err)
	}
	r := got[0]
	if r.From != "이사님" || r.Due.Kind != "weekday" || r.Due.Value != "thu" || r.Priority != "high" || !r.Unsure {
		t.Errorf("필드 유실: %+v", r)
	}
}
