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
        XCTAssertEqual(SkillsPickerFormatter.formatWindowTitle(), "Skills")
        XCTAssertEqual(SkillsPickerFormatter.formatSearchPrompt(), "Search skills")
        XCTAssertEqual(SkillsPickerHotKey.defaultKeyCode, 41)
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
