// ai-critic-screencap streams the main display as concatenated JPEG images on
// stdout, the same wire format ffmpeg produces for the camera-free capture
// paths. It exists because macOS 26 removed the legacy CGDisplay* capture APIs:
// ffmpeg's avfoundation screen input still "works", but it replays the first
// frame forever while the desktop keeps changing, which looks like a frozen
// mirror. ScreenCaptureKit is the only supported capture source left.
//
// Usage: ai-critic-screencap [--width N] [--fps N] [--quality 0..1] [--display N]
//
// Exit codes: 3 = no Screen Recording permission, 4 = stream stopped, 1 = other.

import CoreImage
import CoreMedia
import CoreVideo
import Darwin
import Foundation
import ScreenCaptureKit

struct Options {
    var width = 1280
    var fps = 12
    var quality = 0.55
    var displayIndex = 0
}

func fail(_ message: String, code: Int32 = 1) -> Never {
    FileHandle.standardError.write(Data(("ai-critic-screencap: error: " + message + "\n").utf8))
    exit(code)
}

func parseOptions() -> Options {
    var options = Options()
    let args = Array(CommandLine.arguments.dropFirst())
    func value(_ flag: String) -> String {
        guard let index = args.firstIndex(of: flag), index + 1 < args.count else {
            fail("missing value for \(flag)")
        }
        return args[index + 1]
    }
    if args.contains("--help") || args.contains("-h") {
        print("usage: ai-critic-screencap [--width N] [--fps N] [--quality 0..1] [--display N]")
        exit(0)
    }
    if args.contains("--width"), let n = Int(value("--width")) { options.width = n }
    if args.contains("--fps"), let n = Int(value("--fps")) { options.fps = n }
    if args.contains("--quality"), let q = Double(value("--quality")) { options.quality = q }
    if args.contains("--display"), let n = Int(value("--display")) { options.displayIndex = n }
    if options.width < 160 { options.width = 160 }
    if options.fps < 1 { options.fps = 1 }
    if options.quality <= 0 || options.quality > 1 { options.quality = 0.55 }
    return options
}

// writeAll pushes one JPEG to stdout, tolerating partial writes. A dead reader
// means the server went away, so exiting quietly is correct.
func writeAll(_ data: Data) {
    var offset = 0
    while offset < data.count {
        let wrote = data.withUnsafeBytes { buffer -> Int in
            guard let base = buffer.baseAddress else { return -1 }
            return Darwin.write(STDOUT_FILENO, base + offset, data.count - offset)
        }
        if wrote < 0 {
            if errno == EINTR { continue }
            exit(0)
        }
        if wrote == 0 { exit(0) }
        offset += wrote
    }
}

final class ScreenEncoder: NSObject, SCStreamOutput, SCStreamDelegate {
    private let options: Options
    private let context = CIContext(options: [.useSoftwareRenderer: false])
    private let colorSpace = CGColorSpace(name: CGColorSpace.sRGB)!
    private let lock = NSLock()
    private var previous: Data?
    private var emitted = 0

    init(options: Options) {
        self.options = options
    }

    func stream(_ stream: SCStream, didOutputSampleBuffer sample: CMSampleBuffer, of type: SCStreamOutputType) {
        guard type == .screen, sample.isValid, let pixelBuffer = sample.imageBuffer else { return }

        // ScreenCaptureKit marks frames that carry no visual change as "idle";
        // skipping them keeps the server (and the phone) idle on a still screen.
        if let attachments = CMSampleBufferGetSampleAttachmentsArray(sample, createIfNecessary: false) as? [[SCStreamFrameInfo: Any]],
           let raw = attachments.first?[.status] as? Int,
           let status = SCFrameStatus(rawValue: raw),
           status != .complete {
            return
        }

        let bgra = Self.visibleBytes(of: pixelBuffer)
        lock.lock()
        let unchanged = bgra == previous
        if !unchanged { previous = bgra }
        lock.unlock()
        if unchanged { return }

        let image = CIImage(cvPixelBuffer: pixelBuffer)
        guard let jpeg = context.jpegRepresentation(
            of: image,
            colorSpace: colorSpace,
            options: [kCGImageDestinationLossyCompressionQuality as CIImageRepresentationOption: options.quality]
        ) else {
            return
        }
        lock.lock()
        emitted += 1
        lock.unlock()
        writeAll(jpeg)
    }

    func stream(_ stream: SCStream, didStopWithError error: Error) {
        fail("capture stopped: \(error.localizedDescription)", code: 4)
    }

    // visibleBytes copies the pixel buffer rows without the stride padding so
    // two frames can be compared byte for byte.
    private static func visibleBytes(of pixelBuffer: CVPixelBuffer) -> Data {
        CVPixelBufferLockBaseAddress(pixelBuffer, .readOnly)
        defer { CVPixelBufferUnlockBaseAddress(pixelBuffer, .readOnly) }
        let width = CVPixelBufferGetWidth(pixelBuffer)
        let height = CVPixelBufferGetHeight(pixelBuffer)
        let rowBytes = CVPixelBufferGetBytesPerRow(pixelBuffer)
        guard let base = CVPixelBufferGetBaseAddress(pixelBuffer) else { return Data() }
        let visible = width * 4
        if rowBytes == visible {
            return Data(bytes: base, count: visible * height)
        }
        var data = Data(count: visible * height)
        data.withUnsafeMutableBytes { out in
            guard let destination = out.baseAddress else { return }
            for row in 0..<height {
                memcpy(destination + row * visible, base + row * rowBytes, visible)
            }
        }
        return data
    }
}

func evenHeight(display: SCDisplay, width: Int) -> Int {
    guard display.width > 0 else { return 2 }
    let scaled = Int((Double(display.height) * Double(width) / Double(display.width)).rounded())
    return max(2, scaled & ~1)
}

signal(SIGPIPE, SIG_IGN)
let options = parseOptions()

// Neither the stream nor its output survives being released: without these
// references the setup task's locals are freed on return and capture silently
// stops delivering samples.
var activeStream: SCStream?
var activeEncoder: ScreenEncoder?

Task {
    let content: SCShareableContent
    do {
        content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: true)
    } catch {
        fail("cannot list displays (\(error.localizedDescription)); grant Screen Recording permission to ai-critic", code: 3)
    }
    guard !content.displays.isEmpty else {
        fail("no display available to capture", code: 3)
    }
    let main = CGMainDisplayID()
    let display = content.displays.first(where: { $0.displayID == main })
        ?? content.displays[min(options.displayIndex, content.displays.count - 1)]

    let filter = SCContentFilter(display: display, excludingWindows: [])
    let configuration = SCStreamConfiguration()
    configuration.width = options.width
    configuration.height = evenHeight(display: display, width: options.width)
    configuration.minimumFrameInterval = CMTime(value: 1, timescale: CMTimeScale(options.fps))
    configuration.pixelFormat = kCVPixelFormatType_32BGRA
    configuration.showsCursor = false
    configuration.scalesToFit = true
    configuration.queueDepth = 4

    let encoder = ScreenEncoder(options: options)
    let stream = SCStream(filter: filter, configuration: configuration, delegate: encoder)
    activeStream = stream
    activeEncoder = encoder
    do {
        try stream.addStreamOutput(encoder, type: .screen, sampleHandlerQueue: DispatchQueue(label: "ai-critic.screencap"))
        try await stream.startCapture()
    } catch {
        fail("cannot start capture (\(error.localizedDescription)); grant Screen Recording permission to ai-critic", code: 3)
    }
    FileHandle.standardError.write(
        Data("ai-critic-screencap: info: capturing \(configuration.width)x\(configuration.height) at \(options.fps) fps\n".utf8)
    )
}

RunLoop.main.run()
