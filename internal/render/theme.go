package render

// Color는 라이트·다크 두 값. SwiftBar에는 "라이트,다크"로 넘긴다 (미검증, DESIGN.md 9장).
type Color struct{ Light, Dark string }

func (c Color) Param() string { return c.Light + "," + c.Dark }

// Theme은 테마 하나의 색. 이름과 색이 어긋나지 않게 맞춘다 (2026-09-23).
type Theme struct {
	Key, Name                         string
	Late, TodayC, Soon, Later, Accent Color
}

var Themes = []Theme{
	{"tide", "물빛", // 청록 물빛에 코랄 포인트
		Color{"#D9573F", "#FF8B76"}, Color{"#0B8583", "#4FD1CC"},
		Color{"#5F948D", "#86BDB6"}, Color{"#8FA3A1", "#6E8481"},
		Color{"#0B8583", "#4FD1CC"}},
	{"dusk", "노을", // 해질 때 하늘 — 붉은 노을과 주황
		Color{"#C0392B", "#FF7F6E"}, Color{"#E2662A", "#FFA05C"},
		Color{"#C58E6A", "#E0B48C"}, Color{"#A08C7E", "#8A776B"},
		Color{"#E2662A", "#FFA05C"}},
	{"moss", "쑥빛", // 이끼 초록에 탄 주황
		Color{"#C4531A", "#FF8C4A"}, Color{"#4F7F2A", "#93D063"},
		Color{"#83865A", "#B7B98A"}, Color{"#8E9179", "#787B66"},
		Color{"#4F7F2A", "#93D063"}},
	{"pop", "진달래", // 진달래 분홍이 주인공
		Color{"#C0392B", "#FF8574"}, Color{"#D2417E", "#FF7FB4"},
		Color{"#A8748C", "#C9A0B4"}, Color{"#9C8A93", "#85737C"},
		Color{"#D2417E", "#FF7FB4"}},
	{"mono", "잉걸", // 재 속에 숯불 하나
		Color{"#E8590C", "#FF7B33"}, Color{"#1D1D1F", "#F2F2F4"},
		Color{"#8A8D94", "#8E9198"}, Color{"#A9ACB2", "#72767D"},
		Color{"#E8590C", "#FF7B33"}},
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
