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
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(sidebarID: SkillsPickerFormatter.sidebarAll), "Search skills, templates & files")
        XCTAssertEqual(SkillsPickerHotKey.defaultKeyCode, 41)
    }

    func testSidebarTitlesAndNormalize() {
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarAll), "All")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarSkills), "Skills")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarTemplates), "Templates")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarTitle(id: SkillsPickerFormatter.sidebarFiles), "Files")
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID(nil), SkillsPickerFormatter.sidebarAll)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("bogus"), SkillsPickerFormatter.sidebarAll)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("templates"), SkillsPickerFormatter.sidebarTemplates)
        XCTAssertEqual(SkillsPickerFormatter.normalizeSidebarID("files"), SkillsPickerFormatter.sidebarFiles)
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "all"), "square.grid.2x2")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "skills"), "wrench.and.screwdriver")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "templates"), "doc.text")
        XCTAssertEqual(SkillsPickerFormatter.formatSidebarSymbol(id: "files"), "folder")
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

    func testTemplateTitleAndSubtitle() {
        let t = TemplatesPickerItem(
            name: "brainstorm-sink",
            fmName: "brainstorm sink",
            description: "Sink knowledge about X",
            path: "/t/brainstorm-sink.md",
            body: "/brainstorm following SINK.md about X"
        )
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateTitle(t), "brainstorm sink")
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateSubtitle(t), "Sink knowledge about X")
        let noDesc = TemplatesPickerItem(name: "plain", path: "/t/plain.md", body: "hello body")
        XCTAssertEqual(SkillsPickerFormatter.formatTemplateSubtitle(noDesc), "hello body")
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
        let merged = SkillsPickerFormatter.mergeAllItems(skills: skills, templates: templates, files: files)
        XCTAssertEqual(merged.map(\.title), ["beta", "future", "sink", "alpha"])
        XCTAssertEqual(merged[0].kind, .skill)
        XCTAssertEqual(merged[1].kind, .file)
        XCTAssertEqual(merged[2].kind, .template)
        XCTAssertEqual(merged[1].clipboardText, "/f/draft.md")
        XCTAssertEqual(merged[2].clipboardText, "body")
        XCTAssertEqual(merged[0].clipboardText, "/s/beta/SKILL.md")
        XCTAssertEqual(SkillsPickerFormatter.formatKindBadge(.template), "TEMPLATE")
        XCTAssertEqual(SkillsPickerFormatter.formatKindBadge(.file), "FILE")
    }

    func testFileTitleAndMissingSubtitle() {
        let missing = FilesPickerItem(path: "/gone/a.md", name: "a.md", note: "label", exists: false)
        XCTAssertEqual(SkillsPickerFormatter.formatFileTitle(missing), "label")
        XCTAssertTrue(SkillsPickerFormatter.formatFileSubtitle(missing).contains("(missing)"))
        let ok = FilesPickerItem(path: "/ok/b.md", name: "b.md", exists: true)
        XCTAssertEqual(SkillsPickerFormatter.formatFileTitle(ok), "b.md")
        XCTAssertEqual(SkillsPickerFormatter.formatFileSubtitle(ok), "/ok/b.md")
    }

    func testEmptyHintsBySidebar() {
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "skills").contains("my skills"))
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "templates").contains("my templates"))
        XCTAssertTrue(SkillsPickerFormatter.formatEmptyHint(sidebarID: "files").contains("my files"))
        XCTAssertEqual(SkillsPickerFormatter.formatEmptyTitle(sidebarID: "templates"), "No templates registered")
        XCTAssertEqual(SkillsPickerFormatter.formatEmptyTitle(sidebarID: "files"), "No files registered")
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
