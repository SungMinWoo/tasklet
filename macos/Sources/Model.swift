import Foundation
import SwiftUI

// Go CLI(tasklet)가 내려주는 상태. 날짜·버킷 계산은 Go가 한다.
struct Item: Identifiable, Decodable, Equatable {
    let id: Int
    let title: String
    var requester: String = ""
    var dueAt: String = ""
    let dueLabel: String
    let bucket: String
    let raw: String
    var createdAt: String = ""   // RFC3339
    var doneAt: String = ""      // RFC3339, 완료한 것만

    enum CodingKeys: String, CodingKey {
        case id, title, requester, raw
        case dueAt = "due_at"
        case dueLabel = "due_label"
        case bucket
        case createdAt = "created_at"
        case doneAt = "done_at"
    }

    // 키가 빠져 있어도 기본값으로 읽는다 (Go가 값을 생략할 수 있다).
    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decode(Int.self, forKey: .id)
        title = try c.decode(String.self, forKey: .title)
        requester = try c.decodeIfPresent(String.self, forKey: .requester) ?? ""
        dueAt = try c.decodeIfPresent(String.self, forKey: .dueAt) ?? ""
        dueLabel = try c.decodeIfPresent(String.self, forKey: .dueLabel) ?? ""
        bucket = try c.decodeIfPresent(String.self, forKey: .bucket) ?? "later"
        raw = try c.decodeIfPresent(String.self, forKey: .raw) ?? ""
        createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        doneAt = try c.decodeIfPresent(String.self, forKey: .doneAt) ?? ""
    }
}

/// Go CLI가 한 번에 내려주는 화면 상태 (SwiftUI의 State와 이름이 겹치지 않게 Snapshot).
struct Snapshot: Decodable, Equatable {
    var counts: [String: Int] = [:]
    var tasks: [Item] = []
    var done: [Item] = []      // 오늘 완료한 것 (최근 순)
    var theme: String = "mono"
    var mascot: String = "duck"
    var reasons: [String] = []
    var changed: [Int] = []   // 방금 추가·수정된 id (한 번에 넣기는 여러 개)

    init() {}

    enum CodingKeys: String, CodingKey { case counts, tasks, done, theme, mascot, reasons, changed }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        counts = try c.decodeIfPresent([String: Int].self, forKey: .counts) ?? [:]
        tasks = try c.decodeIfPresent([Item].self, forKey: .tasks) ?? []
        done = try c.decodeIfPresent([Item].self, forKey: .done) ?? []
        theme = try c.decodeIfPresent(String.self, forKey: .theme) ?? "mono"
        mascot = try c.decodeIfPresent(String.self, forKey: .mascot) ?? "duck"
        reasons = try c.decodeIfPresent([String].self, forKey: .reasons) ?? []
        changed = try c.decodeIfPresent([Int].self, forKey: .changed) ?? []
    }
}

/// 기록의 하루치 (tasklet history --json). 최근 날짜가 먼저 온다.
struct HistoryDay: Identifiable, Decodable, Equatable {
    let date: String    // "2026-10-17"
    let label: String   // "10/17(토)", "10/19(월) 오늘"
    let tasks: [Item]   // 그날 안에서도 최근 것이 위

    var id: String { date }
}

/// 기록. 기간 안의 일을 등록한 날 기준과 완료한 날 기준으로 각각 묶은 것.
struct History: Decodable, Equatable {
    var from = ""   // 실제로 적용된 기간 "2026-09-19" (기본값이면 Go가 정한 최근 1개월)
    var to = ""
    var created: [HistoryDay] = []  // 완료 여부와 상관없이 전부
    var done: [HistoryDay] = []
}

enum Bucket: String, CaseIterable {
    case past, today, tomorrow, this_week, later

    var label: String {
        switch self {
        case .past: return "지남"
        case .today: return "오늘"
        case .tomorrow: return "내일"
        case .this_week: return "이번주"
        case .later: return "나중"
        }
    }
}

/// CLI 호출을 감싼다. 쓰기는 모두 Go를 거치므로 잠금·파싱 규칙이 한 곳에만 있다.
/// @Published 갱신은 전부 메인 큐에서 한다 (Swift 5.8, 엄격한 동시성 검사 없음).
final class Store: ObservableObject {
    @Published var state = Snapshot()
    @Published var busy = false          // Haiku 대기 중 (3~12초)
    @Published var busyNote = ""         // 무엇을 기다리는 중인지 ("3줄 추가하는 중…")
    @Published var errorText = ""
    @Published var pending: (ids: [Int], reasons: [String])? // 확인이 필요한 결과
    @Published var history = History()  // '기록' 탭을 열 때 채운다
    /// 사용자가 고른 기록 기간 ("yyyy-MM-dd"). nil이면 최근 1개월 — 날짜가 바뀌어도 오늘 기준을 따라가게
    /// 값으로 들고 있지 않고 Go가 그때그때 정하게 둔다.
    @Published var historyRange: (from: String, to: String)?
    @Published var historyError = ""

    private let exe: String
    private var timer: Timer?

    init() {
        // .app 안에서 실행되므로 CLI를 절대 경로로 찾는다.
        // ~/Desktop 아래에 있으면 macOS가 폴더 접근 권한을 매번 묻기 때문에
        // 설치본(~/.local/bin)을 먼저 본다 (macos/install.sh).
        let candidates = [
            ProcessInfo.processInfo.environment["TASKLET_BIN"] ?? "",
            NSHomeDirectory() + "/.local/bin/tasklet",
            "/usr/local/bin/tasklet",
            NSHomeDirectory() + "/Desktop/git/tasklet/bin/tasklet",
        ]
        exe = candidates.first { !$0.isEmpty && FileManager.default.isExecutableFile(atPath: $0) }
            ?? candidates[1]
        reload()
        // 파일이 밖에서 바뀔 수도 있으니(터미널에서 add 등) 주기적으로 다시 읽는다.
        timer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            self?.reload()
        }
    }

    func tasks(in bucket: Bucket) -> [Item] {
        state.tasks.filter { $0.bucket == bucket.rawValue }
    }

    func count(_ bucket: Bucket) -> Int { state.counts[bucket.rawValue] ?? 0 }

    var remaining: Int { Bucket.allCases.reduce(0) { $0 + count($1) } }

    var theme: Theme { Theme.byKey(state.theme) }

    func reload() { run(["state"]) }

    func complete(_ id: Int) { run(["done", String(id)]) }

    /// 완료 취소. '완료' 구역에서 잘못 누른 것을 되돌린다.
    func uncomplete(_ id: Int) { run(["undone", String(id)]) }

    /// 삭제. 되돌릴 수 없으므로 확인은 행의 🗑 팝오버에서 받는다.
    func delete(_ id: Int) { run(["delete", String(id)]) }

    /// 여러 건을 한 번에 삭제 ('다시 쓰기'가 방금 넣은 것을 되돌릴 때).
    func delete(ids: [Int]) {
        guard !ids.isEmpty else { return }
        run(["delete"] + ids.map(String.init))
    }

    func setDue(_ id: Int, _ spec: String) { run(["due", String(id), spec]) }

    /// 기록. 상태 JSON에 싣지 않고 '기록' 탭이 필요할 때만 기간만큼 따로 읽는다.
    func loadHistory() {
        let exe = self.exe
        var args = ["history", "--json"]
        if let range = historyRange {
            args += ["--from", range.from, "--to", range.to]
        }
        DispatchQueue.global(qos: .userInitiated).async {
            let result = Self.exec(exe, args, as: History.self)
            DispatchQueue.main.async {
                switch result {
                case .failure(let message):
                    self.historyError = message
                case .success(let history):
                    self.historyError = ""
                    self.history = history
                }
            }
        }
    }

    /// 기록 기간을 바꾸고 다시 읽는다. nil이면 최근 1개월로 돌아간다.
    func setHistoryRange(_ range: (from: String, to: String)?) {
        historyRange = range
        loadHistory()
    }

    func setTheme(_ key: String) { run(["theme", key]) }

    func setMascot(_ key: String) { run(["mascot", key]) }

    /// 한 줄 추가. Haiku를 기다려야 해서 오래 걸린다.
    func add(_ sentence: String) {
        run(["add", sentence], slow: "추가하는 중…")
    }

    /// 여러 줄 한 번에 추가. 줄마다 동시에 파싱하므로 한 줄 추가와 비슷하게 걸린다.
    func addBatch(_ text: String, lines: Int) {
        run(["add", "--batch", text], slow: "\(lines)줄 추가하는 중…")
    }

    /// 고쳐 쓰기. 입력은 팝오버에서 이미 받았으므로 창을 띄우지 않는다.
    func edit(_ id: Int, sentence: String) {
        run(["edit", String(id), "--sentence", sentence], slow: "\(id)번 고쳐 쓰는 중…")
    }

    /// slow에 문구를 주면 기다리는 동안 목록 맨 위에 그 문구를 보여준다.
    private func run(_ args: [String], slow: String? = nil) {
        if let note = slow {
            busy = true
            busyNote = note
        }
        let exe = self.exe
        DispatchQueue.global(qos: .userInitiated).async {
            let result = Self.exec(exe, args + ["--json"], as: Snapshot.self)
            DispatchQueue.main.async {
                if slow != nil {
                    self.busy = false
                    self.busyNote = ""
                }
                switch result {
                case .failure(let message):
                    self.errorText = message
                case .success(let state):
                    self.errorText = ""
                    self.state = state
                    // 애매한 것이 있으면 확인 카드를 띄운다.
                    // 여러 건을 한 번에 넣었을 때는 애매한 게 없어도 몇 건 들어갔는지 알려준다.
                    if !state.changed.isEmpty, !state.reasons.isEmpty || state.changed.count > 1 {
                        self.pending = (state.changed, state.reasons)
                    }
                }
            }
        }
    }

    private enum Outcome<T> { case success(T), failure(String) }

    private static func exec<T: Decodable>(_ exe: String, _ args: [String], as type: T.Type) -> Outcome<T> {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: exe)
        p.arguments = args
        let out = Pipe(), errPipe = Pipe()
        p.standardOutput = out
        p.standardError = errPipe
        do { try p.run() } catch {
            return .failure("tasklet 실행 실패: \(exe)")
        }
        let data = out.fileHandleForReading.readDataToEndOfFile()
        let errData = errPipe.fileHandleForReading.readDataToEndOfFile()
        p.waitUntilExit()

        // 에러를 먼저 본다. Snapshot은 모든 필드가 옵션이라 {"error":...}도 디코드에
        // 성공해 버리고, 그러면 "남은 일 0"인 빈 화면이 그려진다 (실측 2026-09-27).
        if let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
           let message = obj["error"] as? String {
            return .failure(message)
        }
        if let value = try? JSONDecoder().decode(type, from: data) {
            return .success(value)
        }
        let stderr = String(data: errData, encoding: .utf8) ?? ""
        return .failure(stderr.isEmpty ? "응답을 읽지 못했습니다" : stderr)
    }
}
