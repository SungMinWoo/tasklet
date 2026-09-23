package parse

import (
	"testing"
	"time"
)

func TestResolveDue(t *testing.T) {
	// 기준 요일: 09-19 토, 09-20 일, 09-21 월, 09-22 화, 09-25 금, 09-26 토, 12-30 수
	tests := []struct {
		name   string
		today  string
		due    Due
		want   string // "" = 기한 없음
		reason string // "" = 확인창 없음, "*" = 사유가 있기만 하면 됨
		err    bool
	}{
		{"none", "2026-09-22", Due{"none", ""}, "", "", false},

		{"relative 오늘", "2026-09-22", Due{"relative", "+0d"}, "2026-09-22", "", false},
		{"relative 3일", "2026-09-22", Due{"relative", "+3d"}, "2026-09-25", "", false},
		{"relative 연도 경계", "2026-12-31", Due{"relative", "+1d"}, "2027-01-01", "", false},
		{"relative 부호 없음", "2026-09-22", Due{"relative", "3d"}, "", "", true},
		{"relative 숫자 없음", "2026-09-22", Due{"relative", "+d"}, "", "", true},

		{"weekday 다음 날", "2026-09-22", Due{"weekday", "wed"}, "2026-09-23", "", false},
		{"weekday 다음 주 월", "2026-09-22", Due{"weekday", "mon"}, "2026-09-28", "", false},
		{"weekday 오늘과 같은 요일 → 7일 뒤 (A3)", "2026-09-22", Due{"weekday", "tue"}, "2026-09-29", "화요일 = 오늘이라 다음 주 9/29(화)로 계산", false},
		{"weekday 일요일에 일요일", "2026-09-20", Due{"weekday", "sun"}, "2026-09-27", "*", false},
		{"weekday 토요일에 금요일", "2026-09-19", Due{"weekday", "fri"}, "2026-09-25", "", false},
		{"weekday 연도 경계", "2026-12-30", Due{"weekday", "fri"}, "2027-01-01", "", false},
		{"weekday 잘못된 값", "2026-09-22", Due{"weekday", "tuesday"}, "", "", true},

		{"next_weekday 평일", "2026-09-22", Due{"next_weekday", "wed"}, "2026-09-30", "", false},
		{"next_weekday 평일 월", "2026-09-22", Due{"next_weekday", "mon"}, "2026-09-28", "", false},
		{"next_weekday 월요일에 월", "2026-09-21", Due{"next_weekday", "mon"}, "2026-09-28", "", false},
		{"next_weekday 일요일 → 바로 다음 주", "2026-09-20", Due{"next_weekday", "wed"}, "2026-09-23", "", false},
		{"next_weekday 토요일 → 바로 다음 주", "2026-09-19", Due{"next_weekday", "wed"}, "2026-09-23", "", false},

		{"eow 화요일", "2026-09-22", Due{"eow", ""}, "2026-09-25", "", false},
		{"eow 금요일 당일", "2026-09-25", Due{"eow", ""}, "2026-09-25", "", false},
		{"eow 토요일 → 다음 주 금", "2026-09-26", Due{"eow", ""}, "2026-10-02", "", false},
		{"eow 일요일 → 다음 주 금", "2026-09-20", Due{"eow", ""}, "2026-09-25", "", false},

		{"next_eow 화요일", "2026-09-22", Due{"next_eow", ""}, "2026-10-02", "", false},
		{"next_eow 금요일", "2026-09-25", Due{"next_eow", ""}, "2026-10-02", "", false},
		{"next_eow 토요일 (A5)", "2026-09-26", Due{"next_eow", ""}, "2026-10-02", "주말이라 '이번 주'와 '다음 주'가 같은 10/2(금)로 계산", false},
		{"next_eow 일요일 (A5)", "2026-09-20", Due{"next_eow", ""}, "2026-09-25", "*", false},

		{"eom", "2026-09-22", Due{"eom", ""}, "2026-09-30", "", false},
		{"eom 말일 당일", "2026-09-30", Due{"eom", ""}, "2026-09-30", "", false},
		{"eom 윤년 2월", "2028-02-10", Due{"eom", ""}, "2028-02-29", "", false},
		{"eom 12월", "2026-12-15", Due{"eom", ""}, "2026-12-31", "", false},

		{"date DD 이번 달", "2026-09-22", Due{"date", "25"}, "2026-09-25", "", false},
		{"date DD 오늘 당일은 안 넘김", "2026-09-22", Due{"date", "22"}, "2026-09-22", "", false},
		{"date DD 지남 → 다음 달 (A2)", "2026-09-22", Due{"date", "15"}, "2026-10-15", "이미 지난 날짜라 10/15(목)로 계산", false},
		{"date DD 이번 달에 없는 날 → 말일", "2026-09-22", Due{"date", "31"}, "2026-09-30", "31일이 없는 달이라 말일 9/30(수)로 계산", false},
		{"date DD 1/31에 30일 → 2/28", "2026-01-31", Due{"date", "30"}, "2026-02-28", "*", false},
		{"date DD 1/31에 31일 → 오늘", "2026-01-31", Due{"date", "31"}, "2026-01-31", "", false},
		{"date DD 연도 경계", "2026-12-28", Due{"date", "5"}, "2027-01-05", "이미 지난 날짜라 2027/1/5(화)로 계산", false},
		{"date DD 0", "2026-09-22", Due{"date", "0"}, "", "", true},
		{"date DD 32", "2026-09-22", Due{"date", "32"}, "", "", true},
		{"date 숫자 아님", "2026-09-22", Due{"date", "abc"}, "", "", true},

		{"date MM-DD 올해", "2026-09-22", Due{"date", "10-02"}, "2026-10-02", "", false},
		{"date MM-DD 지남 → 내년", "2026-09-22", Due{"date", "09-01"}, "2027-09-01", "*", false},
		{"date MM-DD 지난 2/29 → 내년 2/28", "2026-09-22", Due{"date", "02-29"}, "2027-02-28", "*", false},
		{"date MM-DD 2/30 → 말일", "2026-01-10", Due{"date", "02-30"}, "2026-02-28", "*", false},
		{"date MM-DD 13월", "2026-09-22", Due{"date", "13-01"}, "", "", true},

		{"date YYYY-MM-DD", "2026-09-22", Due{"date", "2026-10-02"}, "2026-10-02", "", false},
		{"date YYYY-MM-DD 지난 날짜는 그대로", "2026-09-22", Due{"date", "2026-09-01"}, "2026-09-01", "", false},
		{"date YYYY-MM-DD 없는 날짜", "2026-09-22", Due{"date", "2026-02-30"}, "", "", true},

		{"알 수 없는 kind", "2026-09-22", Due{"month", ""}, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			today, err := time.Parse(time.DateOnly, tt.today)
			if err != nil {
				t.Fatal(err)
			}
			// 시각이 섞여 있어도 날짜만 보는지 함께 확인
			got, err := ResolveDue(tt.due, today.Add(15*time.Hour+30*time.Minute))
			if tt.err {
				if err == nil {
					t.Fatalf("에러를 기대했지만 %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("예상치 못한 에러: %v", err)
			}

			gotDate := ""
			if !got.Date.IsZero() {
				gotDate = got.Date.Format(time.DateOnly)
			}
			if gotDate != tt.want {
				t.Errorf("date = %q, want %q", gotDate, tt.want)
			}

			switch tt.reason {
			case "":
				if got.Reason != "" {
					t.Errorf("reason = %q, 확인창 없음을 기대", got.Reason)
				}
			case "*":
				if got.Reason == "" {
					t.Errorf("reason이 비어 있음, 확인창을 기대")
				}
			default:
				if got.Reason != tt.reason {
					t.Errorf("reason = %q, want %q", got.Reason, tt.reason)
				}
			}
		})
	}
}
