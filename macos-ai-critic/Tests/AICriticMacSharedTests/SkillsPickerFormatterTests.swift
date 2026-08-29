import XCTest
@testable import AICriticMacShared

final class SkillsPickerFormatterTests: XCTestCase {
    func testTitlePrefersFrontmatter() {
        let skill = SkillsPickerItem(
            name: "brainstorm",
            fmName: "Brainstorm",
            dir: "/s/brainstorm",
            path: "/s/brainstorm/SKILL.md"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatTitle(skill), "Brainstorm")
    }

    func testTitleFallsBackToName() {
        let skill = SkillsPickerItem(name: "followup", dir: "/s/followup", path: "/s/followup/SKILL.md")
        XCTAssertEqual(SkillsPickerFormatter.formatTitle(skill), "followup")
    }

    func testToastAndHotKey() {
        XCTAssertEqual(SkillsPickerFormatter.formatCopiedToast(), "Copied")
        XCTAssertEqual(SkillsPickerFormatter.formatHotKey(), "⌘⇧;")
        XCTAssertEqual(SkillsPickerFormatter.formatWindowTitle(), "Insert")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarSkills), "Search skills")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarTemplates), "Search templates")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarFiles), "Search files")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarCommands), "Search commands")
        XCTAssertEqual(
            SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarAll),
            "Search skills, templates, files & commands"
        )
        XCTAssertEqual(SkillsPickerHotKey.defaultKeyCode, 41)
        // cmdKey | shiftKey (Carbon)
        XCTAssertEqual(SkillsPickerHotKey.defaultModifiers, 256 | 512)
    }

    func testSidebarTitlesAndNormalize() {
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarAll), "All")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarSkills), "Skills")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarTemplates), "Templates")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarFiles), "Files")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarCommands), "Commands")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarClipboard), "Clipboard")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarAdhoc), "Adhoc text")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarConvert), "Convert text")
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID(nil), SkillsPickerFormatter.sidebarAll)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("bogus"), SkillsPickerFormatter.sidebarAll)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("templates"), SkillsPickerFormatter.sidebarTemplates)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("files"), SkillsPickerFormatter.sidebarFiles)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("commands"), SkillsPickerFormatter.sidebarCommands)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("clipboard"), SkillsPickerFormatter.sidebarClipboard)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("adhoc"), SkillsPickerFormatter.sidebarAdhoc)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("convert"), SkillsPickerFormatter.sidebarConvert)
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "all"), "square.grid.2x2")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "skills"), "wrench.and.screwdriver")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "templates"), "doc.text")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "files"), "folder")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "commands"), "terminal")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "clipboard"), "doc.on.clipboard")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "adhoc"), "note.text")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "convert"), "arrow.triangle.2.circlepath")
        XCTAssertTrue(SkillsPickerFormatter.isSearchableSidebar("all"))
        XCTAssertTrue(SkillsPickerFormatter.isSearchableSidebar("files"))
        XCTAssertTrue(SkillsPickerFormatter.isSearchableSidebar("commands"))
        XCTAssertFalse(SkillsPickerFormatter.isSearchableSidebar("clipboard"))
        XCTAssertFalse(SkillsPickerFormatter.isSearchableSidebar("adhoc"))
        XCTAssertFalse(SkillsPickerFormatter.isSearchableSidebar("convert"))
        XCTAssertEqual(SkillsPickerFormatter.sidebarOrder.count, 8)
        XCTAssertEqual(SkillsPickerFormatter.sidebarOrder[4], SkillsPickerFormatter.sidebarCommands)
        XCTAssertEqual(SkillsPickerFormatter.sidebarOrder[SkillsPickerFormatter.sidebarOrder.count - 2], SkillsPickerFormatter.sidebarAdhoc)
        XCTAssertEqual(SkillsPickerFormatter.sidebarOrder.last, SkillsPickerFormatter.sidebarConvert)
    }

    func testClipboardAndAdhocFormatters() {
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardKind("TEXT"), "text")
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardKind("empty"), "empty")
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardSize(bytes: 0), "0 B")
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardSize(bytes: 512), "512 B")
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardSize(bytes: 184 * 1024), "184.0 KB")
        XCTAssertFalse(SkillsPickerFormatter.canDumpClipboard(kind: "empty"))
        XCTAssertFalse(SkillsPickerFormatter.canDumpClipboard(kind: "unsupported"))
        XCTAssertTrue(SkillsPickerFormatter.canDumpClipboard(kind: "text"))
        XCTAssertTrue(SkillsPickerFormatter.canDumpClipboard(kind: "image"))
        XCTAssertEqual(SkillsPickerFormatter.formatDumpToFileTitle(), "Dump to file")
        XCTAssertEqual(SkillsPickerFormatter.formatCopyFilePathTitle(), "Copy file path")
        XCTAssertEqual(SkillsPickerFormatter.formatPathCopiedToast(), "Path copied")
        XCTAssertEqual(SkillsPickerFormatter.adhocSaveDebounceNanoseconds, 400_000_000)
        XCTAssertEqual(SkillsPickerFormatter.convertDebounceNanoseconds, 400_000_000)
        XCTAssertEqual(SkillsPickerFormatter.formatConvertHeading(), "Convert text")
        XCTAssertEqual(SkillsPickerFormatter.formatConvertResultHeading(), "Converted")
        XCTAssertEqual(SkillsPickerFormatter.formatConvertingStatus(), "Converting…")
        XCTAssertEqual(SkillsPickerFormatter.formatCopyConvertedTextTitle(), "Copy converted text")
        XCTAssertFalse(SkillsPickerFormatter.formatConvertEmptyHint().isEmpty)
        XCTAssertEqual(TextConvertConverter.shellSingleLine, "shell-single-line")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: "clipboard"), "")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: "adhoc"), "")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: "convert"), "")
        XCTAssertEqual(SkillsPickerFormatter.clipboardPathPrefixDefaultsKey, "insertPickerClipboardPathPrefix")
        XCTAssertEqual(SkillsPickerFormatter.clipboardAppendOCRDefaultsKey, "insertPickerClipboardAppendOCR")
        XCTAssertEqual(SkillsPickerFormatter.formatPathPrefixLabel(), "Path prefix")
        XCTAssertEqual(SkillsPickerFormatter.formatOCRHeading(), "OCR")
        XCTAssertEqual(SkillsPickerFormatter.formatOCRCheckboxTitle(), "OCR")
        XCTAssertTrue(SkillsPickerFormatter.isClipboardImageKind("image"))
        XCTAssertTrue(SkillsPickerFormatter.isClipboardImageKind("IMAGE"))
        XCTAssertFalse(SkillsPickerFormatter.isClipboardImageKind("text"))
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(prefix: "image ", path: "/tmp/a.png"),
            "image /tmp/a.png"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(prefix: "", path: "/tmp/a.png"),
            "/tmp/a.png"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(prefix: "image ", path: "  /tmp/a.png\n"),
            "image /tmp/a.png"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatClipboardCopyText(prefix: "image ", path: ""), "")
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(
                prefix: "image ",
                path: "/tmp/a.png",
                appendOCR: true,
                ocrText: "hello\nworld"
            ),
            "image /tmp/a.png\nhello\nworld"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(
                prefix: "image ",
                path: "/tmp/a.png",
                appendOCR: true,
                ocrText: "  \n"
            ),
            "image /tmp/a.png"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatClipboardCopyText(
                prefix: "image ",
                path: "/tmp/a.png",
                appendOCR: false,
                ocrText: "hello"
            ),
            "image /tmp/a.png"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatCopyWillUsePreview(prefix: "image ", path: "/tmp/a.png"),
            "Copy will use: image /tmp/a.png"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatCopyWillUsePreview(prefix: "x", path: ""), "")
    }

    func testSidebarWidthTitleReveal() {
        XCTAssertFalse(SkillsPickerFormatter.shouldShowSidebarTitles(width: 71))
        XCTAssertTrue(SkillsPickerFormatter.shouldShowSidebarTitles(width: 72))
        XCTAssertTrue(SkillsPickerFormatter.shouldShowSidebarTitles(width: 100))
        XCTAssertEqual(SkillsPickerFormatter.sidebarIconWidth, 52)
        XCTAssertEqual(SkillsPickerFormatter.sidebarTitleRevealWidth, 72)
        XCTAssertEqual(SkillsPickerFormatter.clampSidebarWidth(10), SkillsPickerFormatter.sidebarMinWidth)
        XCTAssertEqual(SkillsPickerFormatter.clampSidebarWidth(999), SkillsPickerFormatter.sidebarMaxWidth)
        XCTAssertEqual(SkillsPickerFormatter.clampSidebarWidth(-1), SkillsPickerFormatter.sidebarIconWidth)
        XCTAssertEqual(SkillsPickerFormatter.clampSidebarWidth(.nan), SkillsPickerFormatter.sidebarIconWidth)
        XCTAssertEqual(SkillsPickerFormatter.clampSidebarWidth(90), 90)
    }

    func testNextListSelectionIndexFromNilAndEnds() {
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: nil, delta: 1), 0)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: nil, delta: -1), 2)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 0, delta: -1), 0)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 2, delta: 1), 2)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 1, delta: 1), 2)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 1, delta: -1), 0)
        XCTAssertNil(SkillsPickerFormatter.nextListSelectionIndex(count: 0, current: nil, delta: 1))
        XCTAssertNil(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 0, delta: 0))
        // Out-of-range current behaves like nil
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: 99, delta: 1), 0)
        XCTAssertEqual(SkillsPickerFormatter.nextListSelectionIndex(count: 3, current: -5, delta: -1), 2)
    }

    func testTemplateTitleDescriptionAndBodyPreview() {
        let t = TemplatesPickerItem(
            name: "brainstorm-sink",
            fmName: "brainstorm sink",
            description: "Sink knowledge about X",
            path: "/t/brainstorm-sink.md",
            body: "/brainstorm following SINK.md about X"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateTitle(t), "brainstorm sink")
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateDescription(t), "Sink knowledge about X")
        XCTAssertEqual(
            SkillsPickerFormatter.formatTemplateBodyPreview(t),
            "/brainstorm following SINK.md about X"
        )
        // Subtitle is the body preview (second list line), not description.
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateSubtitle(t), t.body)
        let noDesc = TemplatesPickerItem(name: "plain", path: "/t/plain.md", body: "hello body")
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateDescription(noDesc), "")
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateSubtitle(noDesc), "hello body")
        let item = InsertPickerItem(kind: .template, skill: nil, template: t, file: nil)
        XCTAssertEqual(item.trailingDescription, "Sink knowledge about X")
        XCTAssertEqual(item.subtitle, t.body)
    }

    func testTemplateBodyPreviewCollapsesNewlinesAndTruncates() {
        let multiline = TemplatesPickerItem(
            name: "m",
            path: "/t/m.md",
            body: "line one\n\n  line   two"
        )
        XCTAssertEqual(
            SkillsPickerFormatter.formatTemplateBodyPreview(multiline),
            "line one line two"
        )
        XCTAssertTrue(SkillsPickerFormatter.formatTemplateBodyPreviewSpans(multiline).isEmpty)

        let long = String(repeating: "a", count: 120)
        let longItem = TemplatesPickerItem(name: "l", path: "/t/l.md", body: long)
        let preview = SkillsPickerFormatter.formatTemplateBodyPreview(longItem)
        XCTAssertEqual(preview.count, SkillsPickerFormatter.templateBodyPreviewMaxChars)
        XCTAssertTrue(preview.hasSuffix("…"))

        let spanned = TemplatesPickerItem(
            name: "s",
            path: "/t/s.md",
            body: "hello world",
            bodySpans: [
                FuzzySpan(text: "hello", matched: true),
                FuzzySpan(text: " world", matched: false),
            ]
        )
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateBodyPreviewSpans(spanned).count, 2)
    }

    func testMergeAllItemsRanksByUseCount() {
        let skills = [
            SkillsPickerItem(name: "alpha", dir: "/s/alpha", path: "/s/alpha/SKILL.md", useCount: 1),
            SkillsPickerItem(name: "beta", dir: "/s/beta", path: "/s/beta/SKILL.md", useCount: 5),
        ]
        let templates = [
            TemplatesPickerItem(name: "sink", path: "/t/sink.md", body: "body", useCount: 3),
        ]
        let files = [
            FilesPickerItem(path: "/f/draft.md", name: "draft.md", note: "future", useCount: 4),
        ]
        let commands = [
            CommandsPickerItem(
                command: "kool iterm2 tab-set run services",
                name: "tab-set",
                note: "tab-set run services",
                useCount: 6
            ),
        ]
        let merged = SkillsPickerFormatter.mergeAllItems(
            skills: skills,
            templates: templates,
            files: files,
            commands: commands
        )
        XCTAssertEqual(merged.map(\.title), ["tab-set run services", "beta", "future", "sink", "alpha"])
        XCTAssertEqual(merged[0].kind, .command)
        XCTAssertEqual(merged[1].kind, .skill)
        XCTAssertEqual(merged[2].kind, .file)
        XCTAssertEqual(merged[3].kind, .template)
        XCTAssertEqual(merged[0].clipboardText, "kool iterm2 tab-set run services")
        XCTAssertEqual(merged[2].clipboardText, "/f/draft.md")
        XCTAssertEqual(merged[3].clipboardText, "body")
        XCTAssertEqual(merged[1].clipboardText, "/s/beta/SKILL.md")
        XCTAssertEqual(SkillsPickerFormatter.formatKindBadge(.template), "TEMPLATE")
        XCTAssertEqual(SkillsPickerFormatter.formatKindBadge(.file), "FILE")
        XCTAssertEqual(SkillsPickerFormatter.formatKindBadge(.command), "CMD")
    }

    func testFileTitleAndMissingSubtitle() {
        let missing = FilesPickerItem(path: "/gone/a.md", name: "a.md", note: "label", exists: false)
        XCTAssertEqual(SkillsPickerFormatter.formatFileTitle(missing), "label")
        XCTAssertTrue(SkillsPickerFormatter.formatFileSubtitle(missing).contains("(missing)"))
        let ok = FilesPickerItem(path: "/ok/b.md", name: "b.md", exists: true)
        XCTAssertEqual(SkillsPickerFormatter.formatFileTitle(ok), "b.md")
        XCTAssertEqual(SkillsPickerFormatter.formatFileSubtitle(ok), "/ok/b.md")
    }

    func testCommandTitleAndSubtitle() {
        let noted = CommandsPickerItem(
            command: "kool iterm2 sessions save --file ~/tmp/iterm2-session-spaces-all.json",
            name: "iTerm2 save sessions",
            note: "iTerm2 save sessions"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatCommandTitle(noted), "iTerm2 save sessions")
        XCTAssertEqual(
            SkillsPickerFormatter.formatCommandSubtitle(noted),
            "kool iterm2 sessions save --file ~/tmp/iterm2-session-spaces-all.json"
        )
        let bare = CommandsPickerItem(command: "echo hello")
        XCTAssertEqual(SkillsPickerFormatter.formatCommandTitle(bare), "echo hello")
    }

    func testEmptyHintsBySidebar() {
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "skills").contains("my skills"))
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "templates").contains("Add a template"))
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "files").contains("Add a file"))
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "commands").contains("my commands --add"))
        XCTAssertEqual(SkillsPickerFormatter.formatEmptyTitle(sidebarID: "templates"), "No templates registered")
        XCTAssertEqual(SkillsPickerFormatter.formatEmptyTitle(sidebarID: "files"), "No files registered")
        XCTAssertEqual(SkillsPickerFormatter.formatEmptyTitle(sidebarID: "commands"), "No commands registered")
        XCTAssertEqual(SkillsPickerFormatter.formatAddButtonTitle(sidebarID: "templates"), "New template")
        XCTAssertEqual(SkillsPickerFormatter.formatAddButtonTitle(sidebarID: "files"), "Add file or folder")
        XCTAssertEqual(SkillsPickerFormatter.formatAddButtonTitle(sidebarID: "commands"), "Add command")
        XCTAssertTrue(SkillsPickerFormatter.shouldShowAddButton(sidebarID: "templates"))
        XCTAssertTrue(SkillsPickerFormatter.shouldShowAddButton(sidebarID: "files"))
        XCTAssertTrue(SkillsPickerFormatter.shouldShowAddButton(sidebarID: "commands"))
        XCTAssertFalse(SkillsPickerFormatter.shouldShowAddButton(sidebarID: "skills"))
        XCTAssertFalse(SkillsPickerFormatter.shouldShowAddButton(sidebarID: "all"))
    }

    func testSlugifyTemplateFilename() {
        XCTAssertEqual(SkillsPickerFormatter.slugifyTemplateFilename("brainstorm sink"), "brainstorm-sink.md")
        XCTAssertEqual(SkillsPickerFormatter.slugifyTemplateFilename("  Hello $AI  "), "Hello-AI.md")
        XCTAssertEqual(SkillsPickerFormatter.slugifyTemplateFilename(""), "template.md")
    }

    func testDecodeTemplatesListWithRoots() throws {
        let json = """
        {"templates":[],"missing_roots":[],"roots":[{"path":"/t/root","note":"stubs"}]}
        """
        let resp = try JSONDecoder().decode(TemplatesListResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.roots.count, 1)
        XCTAssertEqual(resp.roots[0].path, "/t/root")
        XCTAssertEqual(resp.roots[0].note, "stubs")
    }

    func testDisplaySpansFallbackWhenEmpty() {
        let spans = SkillsPickerFormatter.displaySpans([], fallback: "followup")
        XCTAssertEqual(spans, [FuzzySpan(text: "followup", matched: false)])
    }

    func testDecodeQuerySpansSnakeCase() throws {
        let json = """
        {"skills":[{"name":"speak-in-human-words","path":"/s/speak-in-human-words/SKILL.md","title_spans":[{"text":"speak","matched":true},{"text":"-in-human-words","matched":false}],"path_spans":[{"text":"/s/speak-in-human-words/SKILL.md","matched":false}]}],"missing_roots":[]}
        """
        let resp = try JSONDecoder().decode(SkillsListResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.skills.count, 1)
        XCTAssertEqual(resp.skills[0].titleSpans.first?.text, "speak")
        XCTAssertEqual(resp.skills[0].titleSpans.first?.matched, true)
    }

    func testDecodeTemplatesList() throws {
        let json = """
        {"templates":[{"name":"brainstorm sink","fm_name":"brainstorm sink","description":"Sink X","path":"/t/a.md","body":"/brainstorm …","use_count":2,"title_spans":[{"text":"sink","matched":true}]}],"missing_roots":[]}
        """
        let resp = try JSONDecoder().decode(TemplatesListResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.templates.count, 1)
        XCTAssertEqual(resp.templates[0].body, "/brainstorm …")
        XCTAssertEqual(resp.templates[0].useCount, 2)
        XCTAssertEqual(resp.templates[0].titleSpans.first?.matched, true)
    }

    func testDecodeFilesList() throws {
        let json = """
        {"files":[{"path":"/gone/draft.md","name":"future","note":"future","exists":false,"is_dir":false,"use_count":1,"title_spans":[{"text":"draft","matched":true}]}]}
        """
        let resp = try JSONDecoder().decode(FilesListResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.files.count, 1)
        XCTAssertEqual(resp.files[0].path, "/gone/draft.md")
        XCTAssertFalse(resp.files[0].exists)
        XCTAssertEqual(resp.files[0].useCount, 1)
        XCTAssertEqual(resp.files[0].titleSpans.first?.matched, true)
    }

    func testDecodeCommandsList() throws {
        let json = """
        {"commands":[{"command":"kool iterm2 tab-set run services","name":"tab-set","note":"tab-set","use_count":2,"title_spans":[{"text":"tab","matched":true}],"command_spans":[{"text":"kool","matched":true}]}]}
        """
        let resp = try JSONDecoder().decode(CommandsListResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.commands.count, 1)
        XCTAssertEqual(resp.commands[0].command, "kool iterm2 tab-set run services")
        XCTAssertEqual(resp.commands[0].useCount, 2)
        XCTAssertEqual(resp.commands[0].titleSpans.first?.matched, true)
        XCTAssertEqual(resp.commands[0].commandSpans.first?.text, "kool")
    }

    func testDecodeTextConvertResponse() throws {
        let json = """
        {"text":"cmd --flag","converter":"shell-single-line"}
        """
        let resp = try JSONDecoder().decode(TextConvertResponse.self, from: Data(json.utf8))
        XCTAssertEqual(resp.text, "cmd --flag")
        XCTAssertEqual(resp.converter, TextConvertConverter.shellSingleLine)
    }

    func testDisplaySpansKeepsServerHighlights() {
        let raw = [
            FuzzySpan(text: "aid", matched: true),
            FuzzySpan(text: "-", matched: false),
            FuzzySpan(text: "user", matched: true),
            FuzzySpan(text: "-do-human-verifications", matched: false),
        ]
        let spans = SkillsPickerFormatter.displaySpans(raw, fallback: "ignored")
        XCTAssertEqual(SkillsPickerFormatter.joinSpans(spans), "aid-user-do-human-verifications")
        XCTAssertTrue(spans[0].matched)
        XCTAssertFalse(spans[1].matched)
        XCTAssertEqual(SkillsPickerFormatter.searchDebounceNanoseconds, 150_000_000)
    }

    func testIgnorableSearchErrorIsCancellation() {
        XCTAssertTrue(SkillsPickerFormatter.isIgnorableSearchError(CancellationError()))
        XCTAssertTrue(SkillsPickerFormatter.isIgnorableSearchError(URLError(.cancelled)))
        XCTAssertTrue(SkillsPickerFormatter.isIgnorableSearchError(
            NSError(domain: NSURLErrorDomain, code: NSURLErrorCancelled)
        ))
        XCTAssertFalse(SkillsPickerFormatter.isIgnorableSearchError(URLError(.timedOut)))
        XCTAssertFalse(SkillsPickerFormatter.isIgnorableSearchError(
            NSError(domain: NSCocoaErrorDomain, code: 4865)
        ))
    }

    func testUseCountHiddenWhenZero() {
        XCTAssertEqual(SkillsPickerFormatter.formatUseCount(0), "")
        XCTAssertEqual(SkillsPickerFormatter.formatUseCount(4), "4")
    }
}
