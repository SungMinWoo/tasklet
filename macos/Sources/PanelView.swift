import SwiftUI

// 팝오버 A안: 맨 위 스택 바 + 급한 것부터 누르는 목록 (시안 2026-09-23).
struct PanelView: View {
    @ObservedObject var store: Store
    @Environment(\.colorScheme) private var scheme
    @State private var draft = ""
    @State private var editing: Item?
    @State private var editDraft = ""
    @State private var batchMode = false   // 여러 줄 한 번에 넣기
    @State private var batchDraft = ""
    @State private var lastBatch = ""      // 방금 넣은 붙여넣기 원문 ('다시 쓰기'용)
    @AppStorage("doneSectionOpen") private var doneOpen = false
    @FocusState private var focused: Bool

    // 사용자가 오른쪽 아래를 끌어 조절한 크기. 기기별 취향이라 UserDefaults에 둔다.
    @AppStorage("panelWidth") private var width: Double = 348
    @AppStorage("panelListHeight") private var listHeight: Double = 360

    private static let widthRange: ClosedRange<Double> = 280...560
    private static let heightRange: ClosedRange<Double> = 160...620

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
                if store.busy {
                    busyCard
                }
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
                if !store.state.done.isEmpty {
                    doneSection
                }
            }
            .padding(.bottom, 4)
        }
        // 높이를 고정한다. maxHeight만 주면 내용이 짧을 때 세로로 늘어나지 않는다.
        .frame(height: listHeight)
    }

    /// 오늘 완료한 것. 기본은 접혀 있고, 펴면 되돌릴 수 있다.
    /// 완료는 목록에서 사라지는 동작이라, 잘못 눌렀을 때 되돌릴 길이 여기밖에 없다.
    private var doneSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button {
                doneOpen.toggle()
            } label: {
                HStack(spacing: 4) {
                    Image(systemName: doneOpen ? "chevron.down" : "chevron.right")
                        .font(.system(size: 9, weight: .bold))
                    Text("오늘 완료 \(store.state.done.count)")
                        .font(.system(size: 11.5, weight: .bold))
                }
                .foregroundStyle(.secondary)
                .padding(.horizontal, 14)
                .padding(.top, 11).padding(.bottom, 2)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)

            if doneOpen {
                ForEach(store.state.done) { task in
                    DoneRow(task: task, store: store)
                }
            }
        }
    }

    /// Haiku를 기다리는 동안 목록 맨 위에. 발치의 작은 표시는 눈에 안 띈다.
    private var busyCard: some View {
        HStack(spacing: 8) {
            ProgressView().controlSize(.small).scaleEffect(0.7)
            Text(store.busyNote.isEmpty ? "Claude가 읽는 중…" : store.busyNote)
                .font(.system(size: 12, weight: .medium))
            Text("3~12초")
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
        }
        .padding(10)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(store.theme.accent.resolve(scheme).opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 12)
        .padding(.top, 10)
    }

    private func confirmCard(_ pending: (ids: [Int], reasons: [String])) -> some View {
        // 한 건이면 그 내용을, 여러 건이면 몇 건 들어갔는지 보여준다.
        let single = pending.ids.count == 1
            ? store.state.tasks.first { $0.id == pending.ids[0] }
            : nil
        return VStack(alignment: .leading, spacing: 7) {
            Text(single.map(summaryText) ?? "\(pending.ids.count)건 추가했습니다")
                .font(.system(size: 12.5, weight: .medium))
            // 여러 건이면 무엇이 들어갔는지 줄줄이 보여준다 (한 건은 위에 이미 있다).
            if single == nil {
                VStack(alignment: .leading, spacing: 3) {
                    ForEach(added(pending.ids)) { task in
                        Text(summaryText(task))
                            .font(.system(size: 11.5))
                            .foregroundStyle(.primary.opacity(0.85))
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }
            ForEach(pending.reasons, id: \.self) { reason in
                Text("· " + reason)
                    .font(.system(size: 11.5))
                    .foregroundStyle(.secondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
            HStack(spacing: 7) {
                Button("맞아요") { store.pending = nil }
                    .buttonStyle(.borderedProminent).controlSize(.small)
                if let task = single {
                    Button("고쳐 쓰기") {
                        startEdit(task)
                        store.pending = nil
                    }
                    .buttonStyle(.bordered).controlSize(.small)
                } else if !lastBatch.isEmpty {
                    // 여러 건은 하나씩 고치기보다 통째로 다시 쓰는 편이 빠르다.
                    // 방금 넣은 것을 지우고 붙여넣었던 글을 입력창에 도로 채운다.
                    Button("다시 쓰기") {
                        store.delete(ids: pending.ids)
                        batchDraft = lastBatch
                        batchMode = true
                        focused = true
                        store.pending = nil
                    }
                    .buttonStyle(.bordered).controlSize(.small)
                }
            }
        }
        .padding(10)
        .background(store.theme.accent.resolve(scheme).opacity(0.12))
        .clipShape(RoundedRectangle(cornerRadius: 10))
        .padding(.horizontal, 12)
        .padding(.top, 10)
    }

    /// 방금 들어간 항목들. 지워졌거나 못 찾은 id는 건너뛴다.
    private func added(_ ids: [Int]) -> [Item] {
        ids.compactMap { id in store.state.tasks.first { $0.id == id } }
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
            } else if batchMode {
                batchRow
            } else {
                inputRow(placeholder: "할 일을 한 줄로", text: $draft, hint: nil) {
                    let sentence = draft.trimmingCharacters(in: .whitespaces)
                    draft = ""
                    if !sentence.isEmpty { store.add(sentence) }
                }
            }
            HStack(spacing: 6) {
                // 기다리는 표시는 목록 맨 위 busyCard가 맡는다.
                Spacer()
                if editing == nil { batchToggle }
                settingsMenu
                resizeGrip
            }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
    }

    /// 여러 줄 한 번에 넣기. 붙여넣은 글을 줄마다 한 건으로 만든다
    /// (한 줄에 업무가 여러 개면 그 줄에서 여러 건이 나오므로 '줄'로만 센다).
    private var batchRow: some View {
        VStack(alignment: .leading, spacing: 6) {
            TextEditor(text: $batchDraft)
                .font(.system(size: 13))
                .scrollContentBackground(.hidden)
                .focused($focused)
                .frame(height: 88)
                .padding(.horizontal, 6).padding(.vertical, 5)
                .background(Color.primary.opacity(0.07))
                .clipShape(RoundedRectangle(cornerRadius: 9))

            HStack(spacing: 8) {
                Text("한 줄에 하나씩 · ⌘⏎")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
                Spacer()
                Button("취소") {
                    batchMode = false
                    batchDraft = ""
                }
                .buttonStyle(.plain)
                .font(.system(size: 11))
                .foregroundStyle(.secondary)

                Button(batchLines.isEmpty ? "추가" : "\(batchLines.count)줄 추가") {
                    let text = batchDraft
                    let lines = batchLines.count
                    batchDraft = ""
                    batchMode = false
                    lastBatch = text
                    store.addBatch(text, lines: lines)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .disabled(batchLines.isEmpty)
                .keyboardShortcut(.return, modifiers: .command)
            }
        }
    }

    /// 빈 줄을 뺀 줄 목록. 버튼에 개수를 보여주고 빈 입력을 막는 데 쓴다.
    private var batchLines: [String] {
        batchDraft.split(whereSeparator: \.isNewline)
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
    }

    private var batchToggle: some View {
        Button {
            batchMode.toggle()
            focused = batchMode
        } label: {
            Image(systemName: "list.bullet").font(.system(size: 11.5))
        }
        .buttonStyle(.plain)
        .foregroundStyle(batchMode ? store.theme.accent.resolve(scheme) : .secondary)
        .frame(width: 18, height: 18)
        .contentShape(Rectangle())
        .help("여러 줄 한 번에 넣기")
    }

    /// 오른쪽 아래를 끌어 팝오버 크기를 바꾼다. 놓으면 그 크기가 기억된다.
    private var resizeGrip: some View {
        Image(systemName: "arrow.up.left.and.arrow.down.right")
            .font(.system(size: 9, weight: .bold))
            .foregroundStyle(.secondary.opacity(0.7))
            .frame(width: 16, height: 16)
            .contentShape(Rectangle())
            .gesture(
                DragGesture()
                    .onChanged { value in
                        width = min(max(width + value.translation.width / 6, Self.widthRange.lowerBound),
                                    Self.widthRange.upperBound)
                        listHeight = min(max(listHeight + value.translation.height / 3, Self.heightRange.lowerBound),
                                         Self.heightRange.upperBound)
                    }
            )
            .help("끌어서 크기 조절")
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

/// 완료한 일 한 줄. 마우스를 올리면 '되돌리기'가 나온다.
struct DoneRow: View {
    let task: Item
    @ObservedObject var store: Store

    @Environment(\.colorScheme) private var scheme
    @State private var hovering = false

    var body: some View {
        HStack(spacing: 9) {
            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 13))
                .foregroundStyle(.secondary)
            TruncatableText(text: task.title, font: .system(size: 13))
                .foregroundStyle(.secondary)
            Spacer(minLength: 4)
            if hovering {
                Button("되돌리기") { store.uncomplete(task.id) }
                    .buttonStyle(.plain)
                    .font(.system(size: 11))
                    .foregroundStyle(store.theme.accent.resolve(scheme))
                    .contentShape(Rectangle())
            }
        }
        .padding(.horizontal, 14)
        .padding(.vertical, 5)
        .background(hovering ? Color.primary.opacity(0.07) : Color.clear)
        .onHover { hovering = $0 }
    }
}

/// 한 줄로 줄이되, **실제로 잘린 것만** 마우스를 올리면 말풍선으로 전체를 보여주고
/// 누르면 여러 줄로 펼친다. 잘리지 않은 줄은 아무 반응이 없다.
struct TruncatableText: View {
    let text: String
    let font: Font

    @State private var truncated = false
    @State private var expanded = false

    var body: some View {
        Text(text)
            .font(font)
            .lineLimit(expanded ? nil : 1)
            .fixedSize(horizontal: false, vertical: expanded)
            .background(widthProbe)
            .help(truncated ? text : "")
            .onTapGesture {
                guard truncated else { return } // 안 잘린 줄은 눌러도 변화 없음
                expanded.toggle()
            }
    }

    /// 줄이지 않았을 때의 너비와 실제로 주어진 너비를 견줘 잘렸는지 본다.
    /// 재는 쪽은 hidden이라 화면에는 보이지 않는다.
    private var widthProbe: some View {
        GeometryReader { shown in
            Text(text)
                .font(font)
                .fixedSize(horizontal: true, vertical: false)
                .background(GeometryReader { ideal in
                    Color.clear.preference(key: TruncatedKey.self,
                                           value: ideal.size.width > shown.size.width + 0.5)
                })
                .hidden()
        }
        .onPreferenceChange(TruncatedKey.self) { truncated = $0 }
    }
}

private struct TruncatedKey: PreferenceKey {
    static var defaultValue = false
    static func reduce(value: inout Bool, nextValue: () -> Bool) { value = value || nextValue() }
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
    @State private var confirmingDelete = false

    // 달력·삭제 확인이 열려 있는 동안에는 마우스가 벗어나도 버튼을 유지한다.
    private var showActions: Bool { hovering || picking || confirmingDelete }

    var body: some View {
        HStack(spacing: 9) {
            // 버킷 색 세로 띠. 표시만 한다 — 완료는 오른쪽 ✓ 버튼이 맡는다.
            RoundedRectangle(cornerRadius: 1.5)
                .fill(store.theme.color(for: bucket).resolve(scheme))
                .frame(width: 3, height: 16)

            TruncatableText(text: task.title, font: .system(size: 13.5, weight: .medium))
            if !task.requester.isEmpty {
                TruncatableText(text: "· " + task.requester, font: .system(size: 12))
                    .foregroundStyle(.secondary)
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

                HStack(spacing: 6) {
                    Button { picking = true } label: {
                        Image(systemName: "calendar").font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .frame(width: 18, height: 18)
                    .contentShape(Rectangle())
                    .help("기한 바꾸기")
                    .popover(isPresented: $picking, arrowEdge: .bottom) {
                        DuePicker(task: task, store: store) { picking = false }
                    }

                    Button(action: onEdit) {
                        Image(systemName: "pencil").font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .frame(width: 18, height: 18)
                    .contentShape(Rectangle())
                    .help("고쳐 쓰기")

                    Button { store.complete(task.id) } label: {
                        Image(systemName: "checkmark").font(.system(size: 12, weight: .semibold))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .frame(width: 18, height: 18)
                    .contentShape(Rectangle())
                    .help("완료")

                    Button { confirmingDelete = true } label: {
                        Image(systemName: "trash").font(.system(size: 12))
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.secondary)
                    .frame(width: 18, height: 18)
                    .contentShape(Rectangle())
                    .help("삭제")
                    .popover(isPresented: $confirmingDelete, arrowEdge: .bottom) {
                        deleteConfirm
                    }
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

    /// 삭제 확인 — 완료와 달리 되돌릴 수 없으므로 한 번 묻는다.
    private var deleteConfirm: some View {
        VStack(alignment: .leading, spacing: 9) {
            Text("삭제할까요?")
                .font(.system(size: 12.5, weight: .medium))
            Text(task.title)
                .font(.system(size: 11.5))
                .foregroundStyle(.secondary)
                .lineLimit(2)
            Text("되돌릴 수 없습니다")
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
            HStack(spacing: 7) {
                Button("삭제") {
                    confirmingDelete = false
                    store.delete(task.id)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .tint(store.theme.late.resolve(scheme))

                Button("취소") { confirmingDelete = false }
                    .buttonStyle(.bordered)
                    .controlSize(.small)
            }
        }
        .padding(12)
        .frame(width: 190, alignment: .leading)
    }
}
