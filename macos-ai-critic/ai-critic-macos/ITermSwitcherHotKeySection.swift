import SwiftUI
import AppKit
import Carbon
import AICriticMacShared

/// Settings row: show / record / reset the global switcher shortcut.
@available(macOS 15.0, *)
struct ITermSwitcherHotKeySection: View {
    @AppStorage(ITermSwitcherHotKey.keyCodeDefaultsKey) private var keyCode = ITermSwitcherHotKey.defaultKeyCode
    @AppStorage(ITermSwitcherHotKey.modifiersDefaultsKey) private var modifiers = ITermSwitcherHotKey.defaultModifiers
    @State private var recording = false
    @State private var monitor: Any?

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("iTerm Switcher")
                .font(.headline)
            Text("Press the shortcut from any app to jump to a running iTerm tab. Desktop switching needs Accessibility for the local server process.")
                .font(.caption)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)
            Text("Skills picker: \(SkillsPickerFormatter.formatHotKey()) copies a SKILL.md path.")
                .font(.caption)
                .foregroundStyle(.secondary)

            HStack {
                Text(ITermSwitcherFormatter.formatHotKey(keyCode: keyCode, modifiers: modifiers))
                    .font(.body.monospaced())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(Color.secondary.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))

                Button(recording ? "Press keys…" : "Record Shortcut") {
                    startRecording()
                }
                .disabled(recording)

                Button("Reset") {
                    keyCode = ITermSwitcherHotKey.defaultKeyCode
                    modifiers = ITermSwitcherHotKey.defaultModifiers
                    ITermSwitcherHotKeyMonitor.shared.reregister()
                }
            }
        }
        .accessibilityIdentifier("iterm-switcher-hotkey-section")
        .onDisappear { stopRecording() }
    }

    private func startRecording() {
        stopRecording()
        recording = true
        monitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { event in
            if event.keyCode == 53 { // escape
                stopRecording()
                return nil
            }
            var mods = 0
            if event.modifierFlags.contains(.control) { mods |= Int(controlKey) }
            if event.modifierFlags.contains(.option) { mods |= Int(optionKey) }
            if event.modifierFlags.contains(.shift) { mods |= Int(shiftKey) }
            if event.modifierFlags.contains(.command) { mods |= Int(cmdKey) }
            if mods == 0 {
                return nil
            }
            keyCode = Int(event.keyCode)
            modifiers = mods
            ITermSwitcherHotKeyMonitor.shared.reregister()
            stopRecording()
            return nil
        }
    }

    private func stopRecording() {
        if let monitor {
            NSEvent.removeMonitor(monitor)
        }
        monitor = nil
        recording = false
    }
}
