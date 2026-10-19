import SwiftUI

// 팝오버의 '기록' 탭. 기간 안의 일을 날짜별로 모아 보여준다 (최근 날짜가 위).
// '등록한 날'과 '완료한 날' 중 무엇으로 묶을지 고른다. 보기만 한다 — 완료·되돌리기는 '할 일' 탭이 맡는다.

enum HistoryMode: String, CaseIterable {
    case created, done

    var label: String { self == .created ? "등록" : "완료" }
}

/// 기록 탭의 머리: 기간(누르면 달력)과 등록·완료 전환.
struct HistoryHeader: View {
    @ObservedObject var store: Store
    @AppStorage("historyMode") private var mode: HistoryMode = .created
    @State private var picking = false

    var body: some View {
        HStack(spacing: 8) {
            Button { picking = true } label: {
                HStack(spacing: 5) {
                    Image(systemName: "calendar").font(.system(size: 12))
                    Text(rangeLabel).font(.system(size: 12.5, weight: .semibold))
                }
            }
            .buttonStyle(.plain)
            .help("기간 바꾸기")
            .popover(isPresented: $picking, arrowEdge: .bottom) {
                RangePicker(store: store) { picking = false }
            }

            Spacer()

            Picker("", selection: $mode) {
                ForEach(HistoryMode.allCases, id: \.self) { m in
                    Text("\(m.label) \(total(m))").tag(m)
                }
            }
            .pickerStyle(.segmented)
            .labelsHidden()
            .controlSize(.small)
            .fixedSize()
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 11)
    }

    /// "9/19 ~ 10/19". 올해가 아니면 연도를 붙인다.
    private var rangeLabel: String {
        let h = store.history
        guard !h.from.isEmpty else { return "최근 1개월" }
        return short(h.from) + " ~ " + short(h.to)
    }

    private func short(_ date: String) -> String {
        let parts = date.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3 else { return date }
        let thisYear = Calendar.current.component(.year, from: Date())
        return parts[0] == thisYear ? "\(parts[1])/\(parts[2])" : "\(parts[0])/\(parts[1])/\(parts[2])"
    }

    private func total(_ m: HistoryMode) -> Int {
        (m == .created ? store.history.created : store.history.done).reduce(0) { $0 + $1.tasks.count }
    }
}

/// 기록 탭의 목록. 탭을 열 때, 그리고 상태가 바뀔 때(추가·완료·되돌리기) 다시 읽는다.
struct HistoryList: View {
    @ObservedObject var store: Store
    @Environment(\.colorScheme) private var scheme
    @AppStorage("historyMode") private var mode: HistoryMode = .created

    private var days: [HistoryDay] {
        mode == .created ? store.history.created : store.history.done
    }

    var body: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 0) {
                if !store.historyError.isEmpty {
                    Text(store.historyError)
                        .font(.system(size: 11.5))
                        .foregroundStyle(store.theme.late.resolve(scheme))
                        .padding(.horizontal, 14).padding(.vertical, 8)
                }
                if days.isEmpty && store.historyError.isEmpty {
                    Text(mode == .created ? "이 기간에 등록한 일이 없습니다" : "이 기간에 완료한 일이 없습니다")
                        .font(.system(size: 12.5))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 14).padding(.vertical, 18)
                }
                ForEach(days) { day in
                    HStack(spacing: 6) {
                        Text(day.label)
                            .font(.system(size: 11.5, weight: .bold))
                            .foregroundStyle(store.theme.accent.resolve(scheme))
                        Text("\(day.tasks.count)건")
                            .font(.system(size: 11))
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal, 14)
                    .padding(.top, 9).padding(.bottom, 2)

                    ForEach(day.tasks) { task in
                        HistoryRow(task: task, mode: mode, store: store)
                    }
                }
            }
            .padding(.bottom, 4)
        }
        // 기준을 바꾸면 맨 위부터 보이게 한다.
        .id(mode)
        .onAppear { store.loadHistory() }
        .onChange(of: store.state) { _ in store.loadHistory() }
    }
}

/// 기록 한 줄: 시각, 제목, 요청자. 등록 기준이면 오른쪽에 완료 여부나 기한을 붙인다.
/// 마우스를 올리면 원래 문장을 보여준다.
private struct HistoryRow: View {
    let task: Item
    let mode: HistoryMode
    @ObservedObject var store: Store
    @Environment(\.colorScheme) private var scheme

    private var isDone: Bool { task.bucket == "done" }

    var body: some View {
        HStack(spacing: 9) {
            Text(time)
                .font(.system(size: 11).monospacedDigit())
                .foregroundStyle(.secondary)
            TruncatableText(text: task.title, font: .system(size: 13))
                .foregroundStyle(mode == .created && isDone ? .secondary : .primary)
            if !task.requester.isEmpty {
                TruncatableText(text: "· " + task.requester, font: .system(size: 12))
                    .foregroundStyle(.secondary)
            }
            Spacer(minLength: 4)
            if mode == .created {
                status
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 5)
        .help(task.raw)
    }

    @ViewBuilder private var status: some View {
        if isDone {
            Label("완료", systemImage: "checkmark")
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
        } else if let bucket = Bucket(rawValue: task.bucket) {
            Text(task.dueLabel)
                .font(.system(size: 11.5, weight: bucket == .past ? .semibold : .regular))
                .foregroundStyle(store.theme.color(for: bucket).resolve(scheme))
        }
    }

    /// 등록·완료 시각을 이 기기 시간대의 "17:51"로. 날짜 묶음(Go)과 같은 시간대를 쓴다.
    private var time: String {
        let stamp = mode == .created ? task.createdAt : task.doneAt
        guard let date = Self.parser.date(from: stamp) else { return "" }
        return Self.formatter.string(from: date)
    }

    private static let parser = ISO8601DateFormatter()
    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "HH:mm"
        return f
    }()
}

/// 기록 기간 고르기. 시작일과 끝날을 차례로 누른다 (거꾸로 눌러도 앞뒤를 맞춘다).
private struct RangePicker: View {
    @ObservedObject var store: Store
    let done: () -> Void

    @Environment(\.colorScheme) private var scheme
    @State private var month = Date()
    @State private var anchor: String?   // 첫 번째로 누른 날

    private let cell: CGFloat = 32
    private var accent: Color { store.theme.accent.resolve(scheme) }
    private var cal: Calendar { MonthCalendar<EmptyView>.cal }

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                Button("최근 1개월") {
                    store.setHistoryRange(nil)
                    done()
                }
                .buttonStyle(.plain)
                .font(.system(size: 11.5, weight: .medium))
                .padding(.horizontal, 8).padding(.vertical, 4)
                .background(accent.opacity(0.14))
                .foregroundStyle(accent)
                .clipShape(RoundedRectangle(cornerRadius: 7))
                Spacer()
                Text(anchor == nil ? "시작일을 누르세요" : "끝날을 누르세요")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }
            Divider().opacity(0.5)
            MonthCalendar(month: $month, cell: cell) { day in dayCell(day) }
        }
        .padding(12)
        .frame(width: cell * 7 + 24)
        .background(.regularMaterial)
        .onAppear {
            if let d = Self.formatter.date(from: store.history.to) { month = d }
        }
    }

    @ViewBuilder private func dayCell(_ day: Date) -> some View {
        let key = Self.formatter.string(from: day)
        // 고르는 중이면 누른 날 하나만, 아니면 지금 기간을 칠한다.
        let (from, to) = anchor.map { ($0, $0) } ?? (store.history.from, store.history.to)
        let isEdge = key == from || key == to
        let inRange = from <= key && key <= to   // yyyy-MM-dd는 문자열 순서 = 날짜 순서
        let inMonth = cal.isDate(day, equalTo: month, toGranularity: .month)

        Button {
            if let first = anchor {
                store.setHistoryRange((min(first, key), max(first, key)))
                anchor = nil
                done()
            } else {
                anchor = key
            }
        } label: {
            Text("\(cal.component(.day, from: day))")
                .font(.system(size: 12.5, weight: cal.isDateInToday(day) ? .bold : .regular))
                .frame(width: cell, height: cell - 4)
                .background(isEdge ? accent : inRange ? accent.opacity(0.18) : .clear)
                .foregroundStyle(isEdge ? .white : inMonth ? .primary.opacity(0.85) : .primary.opacity(0.25))
                .clipShape(RoundedRectangle(cornerRadius: 8))
        }
        .buttonStyle(.plain)
    }

    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "yyyy-MM-dd"
        return f
    }()
}
