import AppKit
import Carbon
import AICriticMacShared

/// Registers a Carbon global hotkey (no Accessibility required for the chord itself).
@available(macOS 15.0, *)
final class ITermSwitcherHotKeyMonitor {
    static let shared = ITermSwitcherHotKeyMonitor()

    private var hotKeyRef: EventHotKeyRef?
    private var skillsHotKeyRef: EventHotKeyRef?
    private var handlerRef: EventHandlerRef?
    private var installed = false

    private static let itermSignature = OSType(0x49545357) // ITSW
    private static let skillsSignature = OSType(0x534B504B) // SKPK

    func start() {
        installHandlerIfNeeded()
        reregister()
        registerSkills()
    }

    func reregister() {
        unregister()
        let keyCode = UInt32(UserDefaults.standard.object(forKey: ITermSwitcherHotKey.keyCodeDefaultsKey) as? Int
            ?? ITermSwitcherHotKey.defaultKeyCode)
        let modifiers = UInt32(UserDefaults.standard.object(forKey: ITermSwitcherHotKey.modifiersDefaultsKey) as? Int
            ?? ITermSwitcherHotKey.defaultModifiers)
        var ref: EventHotKeyRef?
        let id = EventHotKeyID(signature: Self.itermSignature, id: 1)
        let status = RegisterEventHotKey(
            keyCode,
            modifiers,
            id,
            GetEventDispatcherTarget(),
            0,
            &ref
        )
        if status == noErr {
            hotKeyRef = ref
        }
    }

    func unregister() {
        if let hotKeyRef {
            UnregisterEventHotKey(hotKeyRef)
            self.hotKeyRef = nil
        }
    }

    private func registerSkills() {
        if let skillsHotKeyRef {
            UnregisterEventHotKey(skillsHotKeyRef)
            self.skillsHotKeyRef = nil
        }
        var ref: EventHotKeyRef?
        let id = EventHotKeyID(signature: Self.skillsSignature, id: 1)
        let status = RegisterEventHotKey(
            UInt32(SkillsPickerHotKey.defaultKeyCode),
            UInt32(SkillsPickerHotKey.defaultModifiers),
            id,
            GetEventDispatcherTarget(),
            0,
            &ref
        )
        if status == noErr {
            skillsHotKeyRef = ref
        }
    }

    private func installHandlerIfNeeded() {
        guard !installed else { return }
        installed = true
        var eventType = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        InstallEventHandler(GetApplicationEventTarget(), { _, event, _ in
            var hotKeyID = EventHotKeyID()
            if let event {
                _ = GetEventParameter(
                    event,
                    EventParamName(kEventParamDirectObject),
                    EventParamType(typeEventHotKeyID),
                    nil,
                    MemoryLayout<EventHotKeyID>.size,
                    nil,
                    &hotKeyID
                )
            }
            let skills = hotKeyID.signature == OSType(0x534B504B)
            DispatchQueue.main.async {
                if skills {
                    SkillsPickerController.shared.toggle()
                } else {
                    ITermSwitcherController.shared.toggle()
                }
            }
            return noErr
        }, 1, &eventType, nil, &handlerRef)
    }
}
