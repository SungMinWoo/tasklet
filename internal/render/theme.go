package render

// Color는 라이트·다크 두 값. SwiftBar에는 "라이트,다크"로 넘긴다 (미검증, DESIGN.md 9장).
type Color struct{ Light, Dark string }

func (c Color) Param() string { return c.Light + "," + c.Dark }

// Theme은 테마 하나의 색 네 가지 (DESIGN.md 7장).
type Theme struct {
	Key, Name                  string
	Late, TodayC, Soon, Accent Color
}

var Themes = []Theme{
	{"tide", "타이드",
		Color{"#D9573F", "#FF8B76"}, Color{"#0B8583", "#4FD1CC"},
		Color{"#5F948D", "#86BDB6"}, Color{"#0B8583", "#4FD1CC"}},
	{"dusk", "더스크",
		Color{"#D0632F", "#FF9B6A"}, Color{"#6444D0", "#A791FF"},
		Color{"#8878B8", "#A79CD6"}, Color{"#6444D0", "#A791FF"}},
	{"moss", "모스",
		Color{"#C4531A", "#FF8C4A"}, Color{"#437A22", "#93D063"},
		Color{"#83865A", "#B7B98A"}, Color{"#437A22", "#93D063"}},
	{"pop", "팝",
		Color{"#C8286A", "#FF6FA8"}, Color{"#9A6B00", "#F2C53D"},
		Color{"#8A7A5C", "#C2AF86"}, Color{"#C8286A", "#FF6FA8"}},
	{"mono", "원 포인트",
		Color{"#E8590C", "#FF7B33"}, Color{"#1D1D1F", "#F2F2F4"},
		Color{"#8A8D94", "#8E9198"}, Color{"#E8590C", "#FF7B33"}},
}

// ThemeByKey는 모르는 키면 기본 테마(mono)를 돌려준다.
func ThemeByKey(key string) Theme {
	for _, t := range Themes {
		if t.Key == key {
			return t
		}
	}
	return Themes[len(Themes)-1]
}

// Mascot은 메뉴바 캐릭터 목록. 그림은 아직 없다 (DESIGN.md 7장 '캐릭터').
var Mascots = []struct{ Key, Name string }{
	{"cat", "고양이"}, {"dog", "강아지"}, {"bear", "곰"}, {"pig", "돼지"},
	{"duck", "오리"}, {"owl", "부엉이"}, {"octopus", "문어"},
}

func MascotName(key string) string {
	for _, m := range Mascots {
		if m.Key == key {
			return m.Name
		}
	}
	return key
}
