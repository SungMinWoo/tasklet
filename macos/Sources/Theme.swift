import SwiftUI

// 테마 5종. 값은 DESIGN.md 7장 표와 같아야 한다.
struct Theme {
    let key: String
    let name: String
    let late: DualColor
    let today: DualColor
    let soon: DualColor
    let accent: DualColor

    static let all: [Theme] = [
        Theme(key: "tide", name: "타이드",
              late: .init("#D9573F", "#FF8B76"), today: .init("#0B8583", "#4FD1CC"),
              soon: .init("#5F948D", "#86BDB6"), accent: .init("#0B8583", "#4FD1CC")),
        Theme(key: "dusk", name: "더스크",
              late: .init("#D0632F", "#FF9B6A"), today: .init("#6444D0", "#A791FF"),
              soon: .init("#8878B8", "#A79CD6"), accent: .init("#6444D0", "#A791FF")),
        Theme(key: "moss", name: "모스",
              late: .init("#C4531A", "#FF8C4A"), today: .init("#437A22", "#93D063"),
              soon: .init("#83865A", "#B7B98A"), accent: .init("#437A22", "#93D063")),
        Theme(key: "pop", name: "팝",
              late: .init("#C8286A", "#FF6FA8"), today: .init("#9A6B00", "#F2C53D"),
              soon: .init("#8A7A5C", "#C2AF86"), accent: .init("#C8286A", "#FF6FA8")),
        Theme(key: "mono", name: "원 포인트",
              late: .init("#E8590C", "#FF7B33"), today: .init("#1D1D1F", "#F2F2F4"),
              soon: .init("#8A8D94", "#8E9198"), accent: .init("#E8590C", "#FF7B33")),
    ]

    static func byKey(_ key: String) -> Theme {
        all.first { $0.key == key } ?? all[all.count - 1]
    }

    func color(for bucket: Bucket) -> DualColor {
        switch bucket {
        case .past: return late
        case .today: return today
        case .tomorrow, .this_week: return soon
        case .later: return .init("#9B9EA6", "#7E828B")
        }
    }
}

/// 라이트·다크 두 값을 가진 색. SwiftUI는 시스템 외양에 따라 알아서 고른다.
struct DualColor {
    let light: Color
    let dark: Color

    init(_ light: String, _ dark: String) {
        self.light = Color(hex: light)
        self.dark = Color(hex: dark)
    }

    func resolve(_ scheme: ColorScheme) -> Color { scheme == .dark ? dark : light }
}

extension Color {
    init(hex: String) {
        let s = hex.hasPrefix("#") ? String(hex.dropFirst()) : hex
        var v: UInt64 = 0
        Scanner(string: s).scanHexInt64(&v)
        self.init(
            .sRGB,
            red: Double((v >> 16) & 0xFF) / 255,
            green: Double((v >> 8) & 0xFF) / 255,
            blue: Double(v & 0xFF) / 255,
            opacity: 1
        )
    }
}

// 캐릭터 7종. 그림은 Mascot.swift의 벡터.
let mascotNames: [(key: String, name: String)] = [
    ("cat", "고양이"), ("dog", "강아지"), ("bear", "곰"), ("pig", "돼지"),
    ("duck", "오리"), ("owl", "부엉이"), ("octopus", "문어"),
]
