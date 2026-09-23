import SwiftUI

@main
struct TaskletApp: App {
    @StateObject private var store = Store()

    var body: some Scene {
        MenuBarExtra {
            PanelView(store: store)
                .onAppear { store.reload() }
        } label: {
            // 메뉴바: 캐릭터 + 숫자. 지난 일이 있으면 숫자를 강조색으로
            // (캐릭터에 고유 색이 있어 캐릭터 색으로는 알릴 수 없다 — DESIGN.md 7장).
            MenuBarLabel(store: store)
        }
        .menuBarExtraStyle(.window)
    }
}

private struct MenuBarLabel: View {
    @ObservedObject var store: Store
    @Environment(\.colorScheme) private var scheme

    var body: some View {
        HStack(spacing: 3) {
            Image(nsImage: MascotImage.make(store.state.mascot, size: 18, dark: scheme == .dark))
            Text(text)
                .font(.system(size: 12, weight: store.count(.past) > 0 ? .semibold : .regular))
                .foregroundStyle(store.count(.past) > 0
                                 ? store.theme.late.resolve(scheme)
                                 : Color.primary)
        }
    }

    private var text: String {
        let past = store.count(.past), today = store.count(.today)
        if past > 0 { return "\(past) · 오늘 \(today)" }
        if today > 0 { return "오늘 \(today)" }
        if store.remaining > 0 { return "남음 \(store.remaining)" }
        return "클리어"
    }
}
