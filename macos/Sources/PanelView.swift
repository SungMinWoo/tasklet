import SwiftUI

// 팝오버 A안: 맨 위 스택 바 + 급한 것부터 누르는 목록 (시안 2026-09-23).
struct PanelView: View {
    @ObservedObject var store: Store
    @Environment(\.colorScheme) private var scheme
    @State private var draft = ""
    @State private var editing: Item?
    @State private var editDraft = ""
    @FocusState private var focused: Bool

    private let width: CGFloat = 348

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider().opacity(0.6)
            list
            Divider().opacity(0.6)
            footer
        }
        .frame(width: width)
        .background(.regularMaterial)
        .background(store.theme.accent.resolve(scheme).opacity(0.05))
    }

    // MARK: 머리 — 캐릭터, 남은 개수, 스택 바

    private var header: some View {
        VStack(alignment: .leading, spacing: 9) {
            HStack(spacing: 8) {
                MascotView(key: store.state.mascot, size: 20)
                Text("남은 일 \(store.remaining)")
                    .font(.system(size: 14, weight: .semibold))
                Spacer()
                if store.count(.past) > 0 {
                    Text("\(store.count(.past))건 지남")
                        .font(.system(size: 11.5))
                        .foregroundStyle(store.theme.late.resolve(scheme))
                }
            }
            stackBar
            legend
        }
        .padding(.horizontal, 14)
        .padding(.top, 12)
        .padding(.bottom, 10)
    }

    private var stackBar: some View {
        GeometryReader { geo in
            HStack(spacing: 0) {
                ForEach(Bucket.allCases, id: \.self) { bucket in
                    let n = store.count(bucket)
                    if n > 0 {
                        Rectangle()
                            .fill(store.theme.color(for: bucket).resolve(scheme))
                            // 1건이어도 눈에 보이게 최소 폭을 준다
                            .frame(width: max(10, geo.size.width * CGFloat(n) / CGFloat(max(store.remaining, 1))))
                    }
                }
            }
        }
        .frame(height: 10)
        .background(Color.primary.opacity(0.14))
        .clipShape(RoundedRectangle(cornerRadius: 3))
    }

    private var legend: some View {
        HStack(spacing: 10) {
            ForEach(Bucket.allCases, id: \.self) { bucket in
                if store.count(bucket) > 0 {
                    HStack(spacing: 4) {
                        Circle()
                            .fill(store.theme.color(for: bucket).resolve(scheme))
                            .frame(width: 7, height: 7)
                        Text("\(bucket.label) \(store.count(bucket))")
                            .font(.system(size: 11.5, weight: .medium))
                            .foregroundStyle(.primary.opacity(0.85))
                    }
                }
            }
        }
    }

    // MARK: 목록 — 버킷별 구역, 급한 것이 위

    private var list: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                if let pending = store.pending {
                    confirmCard(pending)
                }
                if !store.errorText.isEmpty {
                    Text(store.errorText)
                        .font(.system(size: 11.5))
                        .foregroundStyle(store.theme.late.resolve(scheme))
                        .padding(.horizontal, 14).padding(.vertical, 8)
                }
                if store.remaining == 0 {
                    Text("남은 일이 없습니다")
                        .font(.system(size: 12.5))
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 14).padding(.vertical, 18)
                }
                ForEach(Bucket.allCases, id: \.self) { bucket in
                    let tasks = store.tasks(in: bucket)
                    if !tasks.isEmpty {
                        Text(bucket.label)
                            .font(.system(size: 11.5, weight: .bold))
                            .foregroundStyle(store.theme.color(for: bucket).resolve(scheme))
                            .padding(.horizontal, 14)
                            .padding(.top, 9).padding(.bottom, 2)
                        ForEach(tasks) { task in
                            TaskRow(task: task, bucket: bucket, store: store,
                                    onEdit: { startEdit(task) })
                        }
                    }
                }
            }
            .padding(.bottom, 4)
        }
        .frame(maxHeight: 360)
    }

    private func confirmCard(_ pending: (id: Int, reasons: [String])) -> some View {
        VStack(alignment: .leading, spacing: 7) {
            Text(store.state.tasks.first { $0.id == pending.id }.map(summaryText) ?? "저장했습니다")
                .font(.system(size: 12.5, weight: .medium))
            ForEach(pending.reasons, id: \.self) { reason in
                Text("· " + reason)
                    .font(.system(size: 11.5))
                    .foregroundStyle(.secondary)
            }
            HStack(spacing: 7) {
                Button("맞아요") { store.pending = nil }
                    .buttonStyle(.borderedProminent).controlSize(.small)
                Button("고쳐 쓰기") {
                    if let task = store.state.tasks.first(where: { $0.id == pending.id }) {
                        startEdit(task)
                    }
                    store.pending = nil
                }
                .buttonStyle(.bordered).controlSize(.small)
            }
        }
        .padding(10)
        .background(store.theme.accent.resolve(scheme).opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 12)
        .padding(.top, 10)
    }

    private func summaryText(_ task: Item) -> String {
        var s = task.title
        if !task.requester.isEmpty { s += " · " + task.requester }
        return s + " · " + task.dueLabel
    }

    // MARK: 발 — 한 줄 추가, 설정

    private var footer: some View {
        VStack(spacing: 8) {
            if let task = editing {
                inputRow(placeholder: "고쳐 쓰기", text: $editDraft, hint: "\(task.id)번 수정") {
                    let sentence = editDraft.trimmingCharacters(in: .whitespaces)
                    editing = nil
                    if !sentence.isEmpty, sentence != task.raw {
                        store.edit(task.id, sentence: sentence)
                    }
                }
            } else {
                inputRow(placeholder: "할 일을 한 줄로", text: $draft, hint: nil) {
                    let sentence = draft.trimmingCharacters(in: .whitespaces)
                    draft = ""
                    if !sentence.isEmpty { store.add(sentence) }
                }
            }
            HStack(spacing: 6) {
                if store.busy {
                    ProgressView().controlSize(.small).scaleEffect(0.7)
                    Text("Claude가 읽는 중… 3~12초")
                        .font(.system(size: 11)).foregroundStyle(.secondary)
                }
                Spacer()
                settingsMenu
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
    }

    private func inputRow(placeholder: String, text: Binding<String>, hint: String?,
                          submit: @escaping () -> Void) -> some View {
        HStack(spacing: 7) {
            if let hint {
                Text(hint).font(.system(size: 11)).foregroundStyle(.secondary)
            } else {
                Image(systemName: "plus")
                    .font(.system(size: 11, weight: .semibold))
                    .foregroundStyle(store.theme.accent.resolve(scheme))
            }
            TextField(placeholder, text: text)
                .textFieldStyle(.plain)
                .font(.system(size: 13))
                .focused($focused)
                .onSubmit(submit)
            if editing != nil {
                Button("취소") { editing = nil }
                    .buttonStyle(.plain)
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.horizontal, 10).padding(.vertical, 7)
        .background(Color.primary.opacity(0.07))
        .clipShape(RoundedRectangle(cornerRadius: 9))
    }

    private var settingsMenu: some View {
        Menu {
            Menu("테마") {
                ForEach(Theme.all, id: \.key) { theme in
                    Button(theme.key == store.state.theme ? "✓ " + theme.name : theme.name) {
                        store.setTheme(theme.key)
                    }
                }
            }
            Menu("캐릭터") {
                ForEach(mascotNames, id: \.key) { m in
                    Button(m.key == store.state.mascot ? "✓ " + m.name : m.name) {
                        store.setMascot(m.key)
                    }
                }
            }
            Divider()
            Button("새로 읽기") { store.reload() }
            Button("종료") { NSApplication.shared.terminate(nil) }
        } label: {
            Image(systemName: "gearshape")
                .font(.system(size: 12))
        }
        .menuStyle(.borderlessButton)
        .menuIndicator(.hidden)
        .frame(width: 22)
    }

    private func startEdit(_ task: Item) {
        editing = task
        editDraft = task.raw
        focused = true
    }
}

/// 할 일 한 줄. 마우스를 올리면 기한·수정 버튼이 나온다.
struct TaskRow: View {
    let task: Item
    let bucket: Bucket
    @ObservedObject var store: Store
    let onEdit: () -> Void

    @Environment(\.colorScheme) private var scheme
    @State private var hovering = false
    @State private var picking = false

    // 달력이 열려 있는 동안에는 마우스가 벗어나도 버튼을 유지한다.
    private var showActions: Bool { hovering || picking }

    var body: some View {
        HStack(spacing: 9) {
            Button {
                store.complete(task.id)
            } label: {
                Circle()
                    .strokeBorder(store.theme.color(for: bucket).resolve(scheme), lineWidth: 2)
                    .frame(width: 15, height: 15)
            }
            .buttonStyle(.plain)
            .help("완료")

            Text(task.title)
                .font(.system(size: 13.5, weight: .medium))
                .lineLimit(1)
            if !task.requester.isEmpty {
                Text("· " + task.requester)
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer(minLength: 4)

            // 기한 라벨과 버튼을 겹쳐 두고 투명도만 바꾼다.
            // 조건부로 없애면 마우스가 벗어날 때 달력 팝오버가 같이 닫힌다.
            ZStack(alignment: .trailing) {
                Text(task.dueLabel)
                    .font(.system(size: 11.5, weight: bucket == .past ? .semibold : .regular))
                    .foregroundStyle(bucket == .past
                                     ? store.theme.late.resolve(scheme)
                                     : Color.primary.opacity(0.7))
                    .opacity(showActions ? 0 : 1)

                HStack(spacing: 10) {
                    Button { picking = true } label: {
                        Image(systemName: "calendar").font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .help("기한 바꾸기")
                    .popover(isPresented: $picking, arrowEdge: .bottom) {
                        DuePicker(task: task, store: store) { picking = false }
                    }

                    Button(action: onEdit) {
                        Image(systemName: "pencil").font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .help("고쳐 쓰기")
                }
                .opacity(showActions ? 1 : 0)
                .allowsHitTesting(showActions)
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 6)
        .background(showActions ? Color.primary.opacity(0.07) : Color.clear)
        .onHover { hovering = $0 }
    }
}


/// 기한 고르기 — 진짜 달력에서 날짜를 찍는다. 자주 쓰는 값은 위에 버튼으로.
struct DuePicker: View {
    let task: Item
    @ObservedObject var store: Store
    let done: () -> Void

    @State private var date = Date()

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 6) {
                quick("오늘", "+0d")
                quick("내일", "+1d")
                quick("금요일", "eow")
                quick("다음 주", "next_eow")
            }
            DatePicker("", selection: $date, displayedComponents: .date)
                .datePickerStyle(.graphical)
                .labelsHidden()
                .frame(width: 260, height: 240)
            HStack {
                Button("기한 없음") {
                    store.setDue(task.id, "none")
                    done()
                }
                .buttonStyle(.bordered).controlSize(.small)
                Spacer()
                Button("이 날짜로") {
                    let f = DateFormatter()
                    f.dateFormat = "yyyy-MM-dd"
                    store.setDue(task.id, f.string(from: date))
                    done()
                }
                .buttonStyle(.borderedProminent).controlSize(.small)
            }
        }
        .padding(12)
        .onAppear {
            // 기한이 있으면 그 날짜에서 시작한다.
            let f = DateFormatter()
            f.dateFormat = "yyyy-MM-dd"
            if let d = f.date(from: task.dueAt) { date = d }
        }
    }

    private func quick(_ label: String, _ spec: String) -> some View {
        Button(label) {
            store.setDue(task.id, spec)
            done()
        }
        .buttonStyle(.bordered)
        .controlSize(.small)
    }
}
