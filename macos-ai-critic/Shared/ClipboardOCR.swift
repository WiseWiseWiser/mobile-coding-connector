import AppKit
import Foundation
import Vision

/// In-process Apple Vision OCR for clipboard images (insert picker).
public enum ClipboardOCR {
    public enum OCRError: Error, LocalizedError {
        case noImage
        case loadFailed
        case recognizeFailed(String)

        public var errorDescription: String? {
            switch self {
            case .noImage:
                return "No image on clipboard"
            case .loadFailed:
                return "Failed to load clipboard image"
            case .recognizeFailed(let msg):
                return msg
            }
        }
    }

    /// Recognize text from the current pasteboard image, if any.
    public static func recognizeFromPasteboard() throws -> String {
        guard let image = pasteboardImage() else {
            throw OCRError.noImage
        }
        return try recognize(image: image)
    }

    /// Recognize text from an image file path.
    public static func recognize(filePath: String) throws -> String {
        guard let image = NSImage(contentsOfFile: filePath) else {
            throw OCRError.loadFailed
        }
        return try recognize(image: image)
    }

    public static func recognize(image: NSImage) throws -> String {
        guard let cgImage = cgImage(from: image) else {
            throw OCRError.loadFailed
        }
        return try recognize(cgImage: cgImage)
    }

    public static func recognize(cgImage: CGImage) throws -> String {
        let request = VNRecognizeTextRequest()
        request.recognitionLevel = .accurate
        request.usesLanguageCorrection = true
        if #available(macOS 13.0, *) {
            request.recognitionLanguages = ["en-US", "zh-Hans", "zh-Hant"]
            request.automaticallyDetectsLanguage = true
        }
        do {
            let handler = VNImageRequestHandler(cgImage: cgImage, options: [:])
            try handler.perform([request])
        } catch {
            throw OCRError.recognizeFailed(error.localizedDescription)
        }
        let observations = request.results ?? []
        var lines: [String] = []
        for obs in observations {
            if let candidate = obs.topCandidates(1).first {
                let s = candidate.string.trimmingCharacters(in: .whitespacesAndNewlines)
                if !s.isEmpty {
                    lines.append(s)
                }
            }
        }
        return lines.joined(separator: "\n")
    }

    private static func pasteboardImage() -> NSImage? {
        let pb = NSPasteboard.general
        if let images = pb.readObjects(forClasses: [NSImage.self], options: nil) as? [NSImage],
           let first = images.first
        {
            return first
        }
        if let tiff = pb.data(forType: .tiff), let image = NSImage(data: tiff) {
            return image
        }
        if let png = pb.data(forType: .png), let image = NSImage(data: png) {
            return image
        }
        return nil
    }

    private static func cgImage(from image: NSImage) -> CGImage? {
        guard let tiff = image.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff)
        else {
            return nil
        }
        return rep.cgImage
    }
}
