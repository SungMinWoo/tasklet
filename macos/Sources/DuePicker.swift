import SwiftUI

/// 기한 고르기 — 시스템 DatePicker 대신 팝오버 톤에 맞춰 직접 그린 월 달력.
/// 월요일 시작(우리 주 규칙과 같음), 기한이 있는 날에는 점을 찍어 알려준다.
struct DuePicker: View {
    let task: Item
    @ObservedObject var store: Store
    let done: () -> Void

    @Environment(\.colorScheme) private var scheme
    @State private var month: Date = Date()

    private let cell: CGFloat = 32
    private let weekdays = ["월", "화", "수", "목", "금", "토", "일"]

    private var cal: Calendar {
        var c = Calendar(identifier: .gregorian)
        c.firstWeekday = 2 // 월요일
        return c
    }

    private var accent: Color { store.theme.accent.resolve(scheme) }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            quickRow
            Divider().opacity(0.5)
            monthHeader
            weekdayRow
            grid
            footer
        }
        .padding(12)
        .frame(width: cell * 7 + 24)
        .background(.regularMaterial)
        .onAppear {
            if let d = date(from: task.dueAt) { month = d }
        }
    }

    // MARK: 자주 쓰는 값

    private var quickRow: some View {
        HStack(spacing: 6) {
            quick("오늘", "+0d")
            quick("내일", "+1d")
            quick("금요일", "eow")
            quick("다음 주", "next_eow")
        }
    }

    private func quick(_ label: String, _ spec: String) -> some View {
        Button(label) {
            store.setDue(task.id, spec)
            done()
        }
        .buttonStyle(.plain)
        .font(.system(size: 11.5, weight: .medium))
        .padding(.horizontal, 8).padding(.vertical, 4)
        .background(accent.opacity(0.14))
        .foregroundStyle(accent)
        .clipShape(RoundedRectangle(cornerRadius: 7))
    }

    // MARK: 월 이동

    private var monthHeader: some View {
        HStack {
            stepButton("chevron.left", months: -1)
            Spacer()
            Text(monthTitle)
                .font(.system(size: 13, weight: .semibold))
            Spacer()
            stepButton("chevron.right", months: 1)
        }
    }

    private func stepButton(_ symbol: String, months: Int) -> some View {
        Button {
            if let next = cal.date(byAdding: .month, value: months, to: month) {
                month = next
            }
        } label: {
            Image(systemName: symbol).font(.system(size: 11, weight: .semibold))
        }
        .buttonStyle(.plain)
        .foregroundStyle(.secondary)
        .frame(width: 22, height: 22)
        .contentShape(Rectangle())
    }

    private var monthTitle: String {
        let f = DateFormatter()
        f.locale = Locale(identifier: "ko_KR")
        f.dateFormat = "yyyy년 M월"
        return f.string(from: month)
    }

    private var weekdayRow: some View {
        HStack(spacing: 0) {
            ForEach(Array(weekdays.enumerated()), id: \.offset) { i, name in
                Text(name)
                    .font(.system(size: 10.5, weight: .medium))
                    .foregroundStyle(i >= 5 ? Color.secondary.opacity(0.7) : Color.secondary)
                    .frame(width: cell)
            }
        }
    }

    // MARK: 날짜 격자

    private var grid: some View {
        let days = monthDays()
        return VStack(spacing: 2) {
            ForEach(0..<(days.count / 7), id: \.self) { row in
                HStack(spacing: 0) {
                    ForEach(0..<7, id: \.self) { col in
                        dayCell(days[row * 7 + col])
                    }
                }
            }
        }
    }

    private func dayCell(_ day: Date?) -> some View {
        Group {
            if let day {
                let isToday = cal.isDateInToday(day)
                let isSelected = task.dueAt == string(from: day)
                let inMonth = cal.isDate(day, equalTo: month, toGranularity: .month)
                let count = store.state.tasks.filter { $0.dueAt == string(from: day) }.count

                Button {
                    store.setDue(task.id, string(from: day))
                    done()
                } label: {
                    VStack(spacing: 1) {
                        Text("\(cal.component(.day, from: day))")
                            .font(.system(size: 12.5, weight: isToday ? .bold : .regular))
                        Circle()
                            .fill(count > 0 ? accent : .clear)
                            .frame(width: 3.5, height: 3.5)
                    }
                    .frame(width: cell, height: cell - 4)
                    .background(isSelected ? accent : .clear)
                    .foregroundStyle(cellForeground(selected: isSelected, today: isToday, inMonth: inMonth))
                    .clipShape(RoundedRectangle(cornerRadius: 8))
                    .overlay {
                        if isToday && !isSelected {
                            RoundedRectangle(cornerRadius: 8).strokeBorder(accent, lineWidth: 1.4)
                        }
                    }
                }
                .buttonStyle(.plain)
            } else {
                Color.clear.frame(width: cell, height: cell - 4)
            }
        }
    }

    private func cellForeground(selected: Bool, today: Bool, inMonth: Bool) -> Color {
        if selected { return .white }
        if !inMonth { return .primary.opacity(0.25) }
        if today { return accent }
        return .primary.opacity(0.85)
    }

    /// 월요일 시작 격자. 앞뒤 빈칸은 이전·다음 달 날짜로 채운다.
    private func monthDays() -> [Date?] {
        guard let first = cal.date(from: cal.dateComponents([.year, .month], from: month)),
              let range = cal.range(of: .day, in: .month, for: first) else { return [] }

        let leading = (cal.component(.weekday, from: first) - cal.firstWeekday + 7) % 7
        var days: [Date?] = []
        for i in 0..<leading {
            days.append(cal.date(byAdding: .day, value: i - leading, to: first))
        }
        for d in 0..<range.count {
            days.append(cal.date(byAdding: .day, value: d, to: first))
        }
        while days.count % 7 != 0 {
            days.append(cal.date(byAdding: .day, value: days.count - leading, to: first))
        }
        return days
    }

    // MARK: 발

    private var footer: some View {
        HStack {
            Button("기한 없음") {
                store.setDue(task.id, "none")
                done()
            }
            .buttonStyle(.plain)
            .font(.system(size: 11.5))
            .foregroundStyle(.secondary)

            Spacer()

            if !task.dueAt.isEmpty {
                Text("지금 " + task.dueLabel)
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }
        }
    }

    // MARK: 날짜 문자열 (Go와 주고받는 형식)

    private func string(from date: Date) -> String {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f.string(from: date)
    }

    private func date(from string: String) -> Date? {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f.date(from: string)
    }
}
