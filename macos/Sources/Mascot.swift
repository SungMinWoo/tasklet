import SwiftUI

// 캐릭터는 벡터로 그린다 (2026-09-23 결정). PNG를 굽지 않으므로 어느 크기에서도 선명하다.
// 좌표는 36×36 격자 기준 — 아티팩트 시안의 SVG와 같은 좌표계.
private let grid: CGFloat = 36

/// 한 겹. fill로 칠하고 stroke가 true면 외곽선을 덧그린다.
private struct Layer {
    let path: (inout Path) -> Void
    let fill: Color?
    let stroke: Bool
    let width: CGFloat

    init(fill: Color? = nil, stroke: Bool = true, width: CGFloat = 3,
         _ path: @escaping (inout Path) -> Void) {
        self.fill = fill
        self.stroke = stroke
        self.width = width
        self.path = path
    }
}

private func circle(_ x: CGFloat, _ y: CGFloat, _ r: CGFloat) -> (inout Path) -> Void {
    { p in p.addEllipse(in: CGRect(x: x - r, y: y - r, width: r * 2, height: r * 2)) }
}

private func ellipse(_ x: CGFloat, _ y: CGFloat, _ rx: CGFloat, _ ry: CGFloat) -> (inout Path) -> Void {
    { p in p.addEllipse(in: CGRect(x: x - rx, y: y - ry, width: rx * 2, height: ry * 2)) }
}

private func triangle(_ a: CGPoint, _ b: CGPoint, _ c: CGPoint) -> (inout Path) -> Void {
    { p in
        p.move(to: a)
        p.addLine(to: b)
        p.addLine(to: c)
        p.closeSubpath()
    }
}

private let ink = Color(hex: "#26221E")
private let cream = Color(hex: "#FBF7F1")
private let white = Color(hex: "#FBFAF8")

private func layers(for key: String) -> [Layer] {
    switch key {
    case "duck":
        return [
            Layer(fill: white, circle(18, 18.5, 12)),
            Layer(fill: ink, stroke: false, ellipse(13.7, 14.8, 2.1, 2.5)),
            Layer(fill: ink, stroke: false, ellipse(22.3, 14.8, 2.1, 2.5)),
            Layer(fill: Color(hex: "#F5C02A"), width: 2.4, ellipse(18, 23.2, 5.6, 3.4)),
        ]
    case "cat":
        return [
            Layer(fill: Color(hex: "#EBCB96"), triangle(CGPoint(x: 10.4, y: 11), CGPoint(x: 9, y: 3.6), CGPoint(x: 16.4, y: 7.4))),
            Layer(fill: Color(hex: "#EBCB96"), triangle(CGPoint(x: 25.6, y: 11), CGPoint(x: 27, y: 3.6), CGPoint(x: 19.6, y: 7.4))),
            Layer(fill: Color(hex: "#EBCB96"), circle(18, 19.5, 11.8)),
            Layer(fill: cream, stroke: false, ellipse(18, 25, 8.4, 5.6)),
            Layer(fill: ink, stroke: false, ellipse(13.9, 20.6, 1.6, 2.0)),
            Layer(fill: ink, stroke: false, ellipse(22.1, 20.6, 1.6, 2.0)),
            Layer(fill: Color(hex: "#E08B8B"), width: 2, triangle(CGPoint(x: 16.6, y: 24.6), CGPoint(x: 19.4, y: 24.6), CGPoint(x: 18, y: 26.4))),
        ]
    case "dog":
        return [
            Layer(fill: Color(hex: "#C98A4B"), circle(8.8, 9.6, 4.8)),
            Layer(fill: Color(hex: "#C98A4B"), circle(27.2, 9.6, 4.8)),
            Layer(fill: Color(hex: "#D9A163"), circle(18, 18.8, 11.6)),
            Layer(fill: Color(hex: "#F7E8D2"), stroke: false, ellipse(18, 24, 9.4, 6.2)),
            Layer(fill: ink, stroke: false, ellipse(13.3, 17, 2.2, 2.9)),
            Layer(fill: ink, stroke: false, ellipse(22.7, 17, 2.2, 2.9)),
            Layer(fill: ink, stroke: false, ellipse(18, 22.4, 3.1, 2.4)),
        ]
    case "bear":
        return [
            Layer(fill: Color(hex: "#C89B6A"), circle(8.5, 10, 4.6)),
            Layer(fill: Color(hex: "#C89B6A"), circle(27.5, 10, 4.6)),
            Layer(fill: Color(hex: "#C89B6A"), circle(18, 20, 11)),
            Layer(fill: Color(hex: "#FBF1E3"), stroke: false, ellipse(18, 24.5, 7, 5.2)),
            Layer(fill: ink, stroke: false, ellipse(13.5, 18, 1.9, 1.9)),
            Layer(fill: ink, stroke: false, ellipse(22.5, 18, 1.9, 1.9)),
            Layer(fill: ink, stroke: false, ellipse(18, 22.8, 2.6, 2.0)),
        ]
    case "pig":
        return [
            Layer(fill: Color(hex: "#F3AFBE"), triangle(CGPoint(x: 10, y: 12), CGPoint(x: 8.4, y: 4.6), CGPoint(x: 16, y: 8))),
            Layer(fill: Color(hex: "#F3AFBE"), triangle(CGPoint(x: 26, y: 12), CGPoint(x: 27.6, y: 4.6), CGPoint(x: 20, y: 8))),
            Layer(fill: Color(hex: "#F3AFBE"), circle(18, 20.5, 11)),
            Layer(fill: ink, stroke: false, ellipse(13.3, 18, 1.9, 1.9)),
            Layer(fill: ink, stroke: false, ellipse(22.7, 18, 1.9, 1.9)),
            Layer(fill: Color(hex: "#E88CA2"), ellipse(18, 24.2, 5.6, 4.2)),
            Layer(fill: ink, stroke: false, ellipse(15.9, 24.2, 1.1, 1.5)),
            Layer(fill: ink, stroke: false, ellipse(20.1, 24.2, 1.1, 1.5)),
        ]
    case "owl":
        return [
            Layer(fill: Color(hex: "#B99A72"), ellipse(18, 18, 11.5, 13.5)),
            Layer(fill: cream, circle(13.2, 16.5, 5.2)),
            Layer(fill: cream, circle(22.8, 16.5, 5.2)),
            Layer(fill: ink, stroke: false, circle(13.2, 16.5, 2.1)),
            Layer(fill: ink, stroke: false, circle(22.8, 16.5, 2.1)),
            Layer(fill: Color(hex: "#E8A33C"), width: 2, triangle(CGPoint(x: 18, y: 20.5), CGPoint(x: 20.4, y: 24), CGPoint(x: 15.6, y: 24))),
        ]
    case "octopus":
        return [
            Layer(fill: Color(hex: "#C79BE0")) { p in
                p.move(to: CGPoint(x: 6, y: 20))
                p.addCurve(to: CGPoint(x: 18, y: 5), control1: CGPoint(x: 6, y: 10.5), control2: CGPoint(x: 11, y: 5))
                p.addCurve(to: CGPoint(x: 30, y: 20), control1: CGPoint(x: 25, y: 5), control2: CGPoint(x: 30, y: 10.5))
                p.addCurve(to: CGPoint(x: 24, y: 22.2), control1: CGPoint(x: 30, y: 23), control2: CGPoint(x: 26.5, y: 21))
                p.addCurve(to: CGPoint(x: 18, y: 22.2), control1: CGPoint(x: 22, y: 23.6), control2: CGPoint(x: 20, y: 21))
                p.addCurve(to: CGPoint(x: 12, y: 22.2), control1: CGPoint(x: 16, y: 21), control2: CGPoint(x: 14, y: 23.6))
                p.addCurve(to: CGPoint(x: 6, y: 20), control1: CGPoint(x: 9.5, y: 21), control2: CGPoint(x: 6, y: 23))
                p.closeSubpath()
            },
            Layer(fill: cream, circle(13.5, 16, 3.6)),
            Layer(fill: cream, circle(22.5, 16, 3.6)),
            Layer(fill: ink, stroke: false, circle(13.5, 16.5, 1.7)),
            Layer(fill: ink, stroke: false, circle(22.5, 16.5, 1.7)),
        ]
    default:
        return layers(for: "duck")
    }
}

/// 메뉴바·팝오버에 쓰는 캐릭터. size는 한 변 길이(pt).
struct MascotView: View {
    let key: String
    let size: CGFloat
    @Environment(\.colorScheme) private var scheme

    var body: some View {
        // 다크에서는 외곽선을 밝게 해야 배경에 묻히지 않는다.
        let outline = scheme == .dark ? Color(hex: "#F4F2EF") : ink
        Canvas { ctx, canvasSize in
            let k = min(canvasSize.width, canvasSize.height) / grid
            ctx.scaleBy(x: k, y: k)
            for layer in layers(for: key) {
                var p = Path()
                layer.path(&p)
                if let fill = layer.fill {
                    ctx.fill(p, with: .color(fill))
                }
                if layer.stroke {
                    ctx.stroke(p, with: .color(outline),
                               style: StrokeStyle(lineWidth: layer.width, lineCap: .round, lineJoin: .round))
                }
            }
        }
        .frame(width: size, height: size)
        .accessibilityLabel(mascotNames.first { $0.key == key }?.name ?? "캐릭터")
    }
}

/// 메뉴바 라벨은 Canvas 같은 직접 그리기를 렌더하지 않는다 (실측 2026-09-23).
/// 그래서 벡터를 그 자리에서 이미지로 구워 넣는다. 원본은 여전히 벡터다.
@MainActor
enum MascotImage {
    private static var cache: [String: NSImage] = [:]

    static func make(_ key: String, size: CGFloat, dark: Bool) -> NSImage {
        let id = "\(key)-\(Int(size))-\(dark)"
        if let hit = cache[id] { return hit }
        let renderer = ImageRenderer(
            content: MascotView(key: key, size: size)
                .environment(\.colorScheme, dark ? .dark : .light)
        )
        renderer.scale = 2
        let image = renderer.nsImage ?? NSImage(size: .init(width: size, height: size))
        cache[id] = image
        return image
    }
}
