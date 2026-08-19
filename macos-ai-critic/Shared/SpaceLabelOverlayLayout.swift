import Foundation

/// Normalize overlay origin to 0…1 of a screen `visibleFrame` and restore it.
public enum SpaceLabelOverlayLayout {
    public static let defaultInset: CGFloat = 10

    public static func defaultOrigin(size: CGSize, visible: CGRect) -> CGPoint {
        CGPoint(
            x: visible.midX - size.width / 2,
            y: visible.maxY - size.height - defaultInset
        )
    }

    public static func origin(x: Double?, y: Double?, size: CGSize, visible: CGRect) -> CGPoint {
        guard let x, let y else {
            return defaultOrigin(size: size, visible: visible)
        }
        return restore(x: x, y: y, size: size, visible: visible)
    }

    public static func normalize(origin: CGPoint, size: CGSize, visible: CGRect) -> (x: Double, y: Double) {
        let maxX = max(visible.width - size.width, 0)
        let maxY = max(visible.height - size.height, 0)
        let nx: Double
        if maxX == 0 {
            nx = 0.5
        } else {
            nx = Double((origin.x - visible.minX) / maxX)
        }
        let ny: Double
        if maxY == 0 {
            ny = 1.0
        } else {
            ny = Double((origin.y - visible.minY) / maxY)
        }
        return (clamp01(nx), clamp01(ny))
    }

    public static func restore(x: Double, y: Double, size: CGSize, visible: CGRect) -> CGPoint {
        let maxX = max(visible.width - size.width, 0)
        let maxY = max(visible.height - size.height, 0)
        return CGPoint(
            x: visible.minX + CGFloat(clamp01(x)) * maxX,
            y: visible.minY + CGFloat(clamp01(y)) * maxY
        )
    }

    public static func clamp01(_ v: Double) -> Double {
        min(max(v, 0), 1)
    }
}
