package skill

import (
	"io/fs"
	"strings"
	"testing"
)

// parseIndexTopics extracts topic paths from the SKILL.md "## Topics" table:
// rows whose first cell is exactly one backticked name (`config`).
func parseIndexTopics(t *testing.T, root string) []string {
	t.Helper()
	var topics []string
	inTopics := false
	for _, line := range strings.Split(root, "\n") {
		switch {
		case strings.HasPrefix(line, "## "):
			inTopics = strings.HasPrefix(line, "## Topics")
		case inTopics && strings.HasPrefix(line, "|"):
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if len(cells) == 0 {
				continue
			}
			name := strings.TrimSpace(cells[0])
			if !strings.HasPrefix(name, "`") || !strings.HasSuffix(name, "`") {
				continue
			}
			name = strings.Trim(name, "`")
			if name == "" || strings.Contains(name, " ") {
				continue
			}
			topics = append(topics, name)
		}
	}
	return topics
}

// walkTreeTopics returns topicPath → TOPIC.md content for every embedded
// <topicPath>/TOPIC.md.
func walkTreeTopics(t *testing.T, tree fs.FS) map[string]string {
	t.Helper()
	topics := map[string]string{}
	err := fs.WalkDir(tree, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != "TOPIC.md" {
			return nil
		}
		data, err := fs.ReadFile(tree, p)
		if err != nil {
			return err
		}
		topics[strings.TrimSuffix(p, "/TOPIC.md")] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walk skill tree: %v", err)
	}
	return topics
}

// frontmatterName returns the `name:` value from a skill file's YAML
// frontmatter, failing the test when the frontmatter is missing or nameless.
func frontmatterName(t *testing.T, path, content string) string {
	t.Helper()
	rest, ok := strings.CutPrefix(content, "---\n")
	if !ok {
		t.Fatalf("%s: missing frontmatter", path)
		return ""
	}
	for _, line := range strings.Split(rest, "\n") {
		if line == "---" {
			break
		}
		if name, ok := strings.CutPrefix(line, "name:"); ok {
			return strings.TrimSpace(name)
		}
	}
	t.Fatalf("%s: frontmatter has no name:", path)
	return ""
}

// TestSkillIndexMatchesTree guards the skill tree (SSOT: this directory)
// against internal drift: the SKILL.md index and the
// embedded <topic>/TOPIC.md tree must list exactly the same topics, and
// frontmatter names must follow {skill}/{topic-path}.
func TestSkillIndexMatchesTree(t *testing.T) {
	index := parseIndexTopics(t, skillRoot)
	if len(index) == 0 {
		t.Fatal("no topic rows parsed from SKILL.md ## Topics table")
	}
	tree := walkTreeTopics(t, skillTree)

	indexSet := map[string]bool{}
	for _, topic := range index {
		if indexSet[topic] {
			t.Errorf("SKILL.md lists topic %q twice", topic)
		}
		indexSet[topic] = true
		if _, ok := tree[topic]; !ok {
			t.Errorf("SKILL.md lists topic %q but %s/TOPIC.md is not embedded", topic, topic)
		}
	}
	for topic := range tree {
		if !indexSet[topic] {
			t.Errorf("embedded %s/TOPIC.md is missing from the SKILL.md ## Topics table", topic)
		}
	}

	if name := frontmatterName(t, "SKILL.md", skillRoot); name != skillName {
		t.Errorf("SKILL.md frontmatter name = %q, want %q", name, skillName)
	}
	for topic, content := range tree {
		want := skillName + "/" + topic
		if name := frontmatterName(t, topic+"/TOPIC.md", content); name != want {
			t.Errorf("%s/TOPIC.md frontmatter name = %q, want %q", topic, name, want)
		}
	}
}
