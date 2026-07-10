import AppKit
import SwiftUI

/// Shared Cron Editor window (Option A) for create and update.
/// Fields: name, command, workingDir?, schedule (interval|cron), timeout (default 1h), enabled.
/// Cron expression is local wall time in the form; converted to UTC before API.
public struct CronEditorView: View {
    public enum Mode: Equatable {
        case create
        case edit(id: String)
    }

    public var mode: Mode
    public var initial: CronTaskDefinition?
    /// When editing, stored UTC cronExpr (for unsafe pass-through).
    public var storedUTCCronExpr: String
    /// True when UTC→local was unsafe; show UTC indication and pass-through on save.
    public var cronExprIsUTCPassThrough: Bool
    public var onSave: (CronTaskDefinition) async throws -> Void
    public var onCancel: () -> Void
    public var onSaved: () -> Void

    @State private var name: String = ""
    @State private var command: String = ""
    @State private var workingDir: String = ""
    @State private var scheduleMode: String = "interval"
    @State private var interval: String = "5m"
    @State private var cronExpr: String = ""
    @State private var timeout: String = "1h"
    @State private var enabled: Bool = true
    @State private var cronIsUTC: Bool = false
    @State private var isSaving = false
    @State private var errorMessage: String?

    public init(
        mode: Mode,
        initial: CronTaskDefinition? = nil,
        storedUTCCronExpr: String = "",
        cronExprIsUTCPassThrough: Bool = false,
        onSave: @escaping (CronTaskDefinition) async throws -> Void,
        onCancel: @escaping () -> Void = {},
        onSaved: @escaping () -> Void = {}
    ) {
        self.mode = mode
        self.initial = initial
        self.storedUTCCronExpr = storedUTCCronExpr
        self.cronExprIsUTCPassThrough = cronExprIsUTCPassThrough
        self.onSave = onSave
        self.onCancel = onCancel
        self.onSaved = onSaved
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.headline)

            Form {
                TextField("Name", text: $name)
                TextField("Command", text: $command)
                TextField("Working dir (optional)", text: $workingDir)

                Picker("Schedule", selection: $scheduleMode) {
                    Text("Interval").tag("interval")
                    Text("Cron").tag("cron")
                }
                .pickerStyle(.segmented)

                if scheduleMode == "interval" {
                    TextField("Interval (e.g. 5m)", text: $interval)
                } else {
                    TextField(cronIsUTC ? "Cron expression (UTC)" : "Cron expression (local)", text: $cronExpr)
                    if cronIsUTC {
                        Text("Showing stored UTC (could not convert to local)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }

                TextField("Timeout", text: $timeout)
                Toggle("Enabled", isOn: $enabled)
            }

            if let errorMessage {
                Text(errorMessage)
                    .foregroundStyle(.red)
                    .font(.caption)
            }

            HStack {
                Spacer()
                Button("Cancel") {
                    onCancel()
                }
                .keyboardShortcut(.cancelAction)
                .disabled(isSaving)

                Button("Save") {
                    Task { await save() }
                }
                .keyboardShortcut(.defaultAction)
                .disabled(isSaving || !canSave)
            }
        }
        .padding(20)
        .frame(minWidth: 420)
        .onAppear(perform: loadInitial)
    }

    private var title: String {
        switch mode {
        case .create:
            return "New Cron Task"
        case .edit:
            return "Edit Cron Task"
        }
    }

    private var canSave: Bool {
        !name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !command.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
            && !timeout.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
    }

    private var isNew: Bool {
        if case .create = mode { return true }
        return false
    }

    private func loadInitial() {
        guard let initial else {
            cronIsUTC = false
            return
        }
        name = initial.name
        command = initial.command
        workingDir = initial.workingDir ?? ""
        scheduleMode = initial.scheduleMode.isEmpty ? "interval" : initial.scheduleMode
        interval = initial.interval ?? "5m"
        timeout = (initial.timeout?.isEmpty == false) ? (initial.timeout ?? "1h") : "1h"
        enabled = initial.enabled ?? true
        if scheduleMode == "cron" {
            if cronExprIsUTCPassThrough {
                cronExpr = storedUTCCronExpr.isEmpty ? (initial.cronExpr ?? "") : storedUTCCronExpr
                cronIsUTC = true
            } else {
                cronExpr = initial.cronExpr ?? ""
                cronIsUTC = false
            }
        }
    }

    private func save() async {
        errorMessage = nil
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedCommand = command.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedTimeout = timeout.trimmingCharacters(in: .whitespacesAndNewlines)
        let trimmedWorkingDir = workingDir.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty, !trimmedCommand.isEmpty else {
            errorMessage = "Name and command are required"
            return
        }
        guard !trimmedTimeout.isEmpty else {
            errorMessage = "Timeout is required"
            return
        }

        var def = CronTaskDefinition(
            id: nil,
            name: trimmedName,
            command: trimmedCommand,
            workingDir: trimmedWorkingDir.isEmpty ? nil : trimmedWorkingDir,
            scheduleMode: scheduleMode,
            interval: nil,
            cronExpr: nil,
            timeout: trimmedTimeout,
            enabled: enabled
        )
        if case .edit(let id) = mode {
            def.id = id
        }

        if scheduleMode == "interval" {
            let iv = interval.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !iv.isEmpty else {
                errorMessage = "Interval is required"
                return
            }
            def.interval = iv
        } else {
            let expr = cronExpr.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !expr.isEmpty else {
                errorMessage = "Cron expression is required"
                return
            }
            if cronIsUTC {
                // Pass-through stored UTC (edit open was unsafe).
                def.cronExpr = expr
            } else {
                do {
                    def.cronExpr = try CronConvert.convertLocalCronToUTC(expr)
                } catch {
                    errorMessage = error.localizedDescription
                    return
                }
            }
        }

        isSaving = true
        defer { isSaving = false }
        do {
            try await onSave(def)
            onSaved()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}

/// Hosts CronEditorView in an AppKit window for menu-bar apps.
public enum CronEditorWindow {
    private static var retained: NSWindow?

    public static func openCreate(
        onSave: @escaping (CronTaskDefinition) async throws -> Void,
        onSaved: @escaping () -> Void
    ) {
        open(
            mode: .create,
            initial: nil,
            storedUTCCronExpr: "",
            cronExprIsUTCPassThrough: false,
            onSave: onSave,
            onSaved: onSaved
        )
    }

    public static func openEdit(
        task: CronTaskStatus,
        onSave: @escaping (CronTaskDefinition) async throws -> Void,
        onSaved: @escaping () -> Void
    ) {
        var displayCron = task.cronExpr
        var passThrough = false
        if task.scheduleMode == "cron", !task.cronExpr.isEmpty {
            do {
                displayCron = try CronConvert.convertUTCCronToLocal(task.cronExpr)
            } catch {
                displayCron = task.cronExpr
                passThrough = true
            }
        }
        let initial = CronTaskDefinition(
            id: task.id,
            name: task.name,
            command: task.command,
            workingDir: task.workingDir.isEmpty ? nil : task.workingDir,
            scheduleMode: task.scheduleMode.isEmpty ? "interval" : task.scheduleMode,
            interval: task.interval.isEmpty ? nil : task.interval,
            cronExpr: displayCron.isEmpty ? nil : displayCron,
            timeout: task.timeout.isEmpty ? "1h" : task.timeout,
            enabled: task.enabled
        )
        open(
            mode: .edit(id: task.id),
            initial: initial,
            storedUTCCronExpr: task.cronExpr,
            cronExprIsUTCPassThrough: passThrough,
            onSave: onSave,
            onSaved: onSaved
        )
    }

    private static func open(
        mode: CronEditorView.Mode,
        initial: CronTaskDefinition?,
        storedUTCCronExpr: String,
        cronExprIsUTCPassThrough: Bool,
        onSave: @escaping (CronTaskDefinition) async throws -> Void,
        onSaved: @escaping () -> Void
    ) {
        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 460, height: 420),
            styleMask: [.titled, .closable],
            backing: .buffered,
            defer: false
        )
        window.title = "Cron Editor"
        window.isReleasedWhenClosed = false
        window.center()

        let close = { [weak window] in
            window?.close()
            if retained === window {
                retained = nil
            }
        }

        let view = CronEditorView(
            mode: mode,
            initial: initial,
            storedUTCCronExpr: storedUTCCronExpr,
            cronExprIsUTCPassThrough: cronExprIsUTCPassThrough,
            onSave: onSave,
            onCancel: close,
            onSaved: {
                onSaved()
                close()
            }
        )
        window.contentView = NSHostingView(rootView: view)
        retained = window
        NSApp.activate(ignoringOtherApps: true)
        window.makeKeyAndOrderFront(nil)
    }
}
