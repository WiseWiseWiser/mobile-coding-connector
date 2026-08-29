import AppKit
import SwiftUI

/// Monospaced plain-text editor with smart quotes / substitutions / Writing Tools off.
/// Use for shell/code pads where `TextEditor` would rewrite quotes and punctuation.
public struct PlainCodeTextEditor: NSViewRepresentable {
    @Binding var text: String
    /// When set true, becomes first responder once, then reset to false.
    @Binding var requestFocus: Bool
    public var accessibilityIdentifier: String?

    public init(
        text: Binding<String>,
        requestFocus: Binding<Bool> = .constant(false),
        accessibilityIdentifier: String? = nil
    ) {
        self._text = text
        self._requestFocus = requestFocus
        self.accessibilityIdentifier = accessibilityIdentifier
    }

    public func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    public func makeNSView(context: Context) -> NSScrollView {
        let scroll = NSScrollView()
        scroll.hasVerticalScroller = true
        scroll.hasHorizontalScroller = false
        scroll.autohidesScrollers = true
        scroll.borderType = .noBorder
        scroll.drawsBackground = false

        let textView = NSTextView()
        textView.delegate = context.coordinator
        textView.isRichText = false
        textView.allowsUndo = true
        textView.isEditable = true
        textView.isSelectable = true
        textView.usesFontPanel = false
        textView.usesRuler = false
        textView.isAutomaticQuoteSubstitutionEnabled = false
        textView.isAutomaticDashSubstitutionEnabled = false
        textView.isAutomaticTextReplacementEnabled = false
        textView.isAutomaticSpellingCorrectionEnabled = false
        textView.isAutomaticDataDetectionEnabled = false
        textView.isAutomaticLinkDetectionEnabled = false
        textView.isContinuousSpellCheckingEnabled = false
        textView.isGrammarCheckingEnabled = false
        if #available(macOS 15.0, *) {
            textView.writingToolsBehavior = .none
        }
        textView.font = NSFont.monospacedSystemFont(ofSize: NSFont.systemFontSize, weight: .regular)
        textView.textColor = NSColor.textColor
        textView.backgroundColor = NSColor.textBackgroundColor
        textView.drawsBackground = true
        textView.string = text
        textView.minSize = NSSize(width: 0, height: 0)
        textView.maxSize = NSSize(width: CGFloat.greatestFiniteMagnitude, height: CGFloat.greatestFiniteMagnitude)
        textView.isVerticallyResizable = true
        textView.isHorizontallyResizable = false
        textView.autoresizingMask = [.width]
        textView.textContainer?.containerSize = NSSize(width: scroll.contentSize.width, height: CGFloat.greatestFiniteMagnitude)
        textView.textContainer?.widthTracksTextView = true
        if let accessibilityIdentifier {
            textView.setAccessibilityIdentifier(accessibilityIdentifier)
        }

        scroll.documentView = textView
        context.coordinator.textView = textView
        return scroll
    }

    public func updateNSView(_ scroll: NSScrollView, context: Context) {
        context.coordinator.parent = self
        guard let textView = scroll.documentView as? NSTextView else { return }
        if textView.string != text {
            let selected = textView.selectedRanges
            textView.string = text
            textView.selectedRanges = selected
        }
        if requestFocus {
            DispatchQueue.main.async {
                scroll.window?.makeFirstResponder(textView)
                if requestFocus {
                    requestFocus = false
                }
            }
        }
    }

    public final class Coordinator: NSObject, NSTextViewDelegate {
        var parent: PlainCodeTextEditor
        weak var textView: NSTextView?

        init(_ parent: PlainCodeTextEditor) {
            self.parent = parent
        }

        public func textDidChange(_ notification: Notification) {
            guard let textView = notification.object as? NSTextView else { return }
            parent.text = textView.string
        }
    }
}
