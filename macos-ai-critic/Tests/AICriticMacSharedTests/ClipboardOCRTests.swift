import AppKit
import XCTest

@testable import AICriticMacShared

final class ClipboardOCRTests: XCTestCase {
    func testRecognizeSyntheticImageWithText() throws {
        let image = try XCTUnwrap(makeTextImage(text: "HelloOCR", size: NSSize(width: 320, height: 80)))
        let text = try ClipboardOCR.recognize(image: image)
        let collapsed = text.replacingOccurrences(of: " ", with: "").lowercased()
        XCTAssertTrue(
            collapsed.contains("helloocr") || collapsed.contains("hello"),
            "OCR text should contain drawn label, got: \(text)"
        )
    }

    func testRecognizeMissingFileThrows() {
        XCTAssertThrowsError(try ClipboardOCR.recognize(filePath: "/tmp/no-such-ocr-fixture-\(UUID().uuidString).png"))
    }

    private func makeTextImage(text: String, size: NSSize) -> NSImage? {
        let image = NSImage(size: size)
        image.lockFocus()
        NSColor.white.setFill()
        NSRect(origin: .zero, size: size).fill()
        let attrs: [NSAttributedString.Key: Any] = [
            .font: NSFont.systemFont(ofSize: 28, weight: .bold),
            .foregroundColor: NSColor.black,
        ]
        let attr = NSAttributedString(string: text, attributes: attrs)
        let textSize = attr.size()
        let origin = NSPoint(
            x: (size.width - textSize.width) / 2,
            y: (size.height - textSize.height) / 2
        )
        attr.draw(at: origin)
        image.unlockFocus()
        return image
    }
}
