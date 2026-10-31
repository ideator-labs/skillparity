package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestExpectationsMatchYAML re-derives every scored expectation with the client
// guide's algorithm (delimiter lines, then a YAML 1.2 parse) so case.json is
// not just someone's opinion.
func TestExpectationsMatchYAML(t *testing.T) {
	cases, err := readCases("../../corpus")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range cases {
		if c.Expect.Load == "either" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("../../corpus", c.ID, c.SkillDir, "SKILL.md"))
		if err != nil {
			t.Fatal(err)
		}
		front, body, ok := splitOnDelimiterLines(strings.ReplaceAll(string(raw), "\r\n", "\n"))
		var m map[string]any
		parsed := ok && yaml.Unmarshal([]byte(front), &m) == nil
		desc, _ := m["description"].(string)
		load := parsed && desc != ""
		if want := c.Expect.Load == "yes"; load != want {
			t.Errorf("%s: spec algorithm load=%v, case.json says %q", c.ID, load, c.Expect.Load)
			continue
		}
		if !load {
			continue
		}
		name, _ := m["name"].(string)
		got := Reading{Name: name, Description: desc, Body: body}
		if f := diff(c.Expect, got); len(f) > 0 {
			t.Errorf("%s: case.json disagrees with YAML on %v", c.ID, f)
		}
	}
}

func splitOnDelimiterLines(s string) (front, body string, ok bool) {
	lines := strings.Split(s, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", "", false
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), true
		}
	}
	return "", "", false
}

func TestJudge(t *testing.T) {
	c := Case{ID: "x", Expect: Expect{Load: "yes", Name: "a", Description: "d", Body: "b"}}
	if v := judge(c, Reading{Loaded: true, Name: "a", Description: "d\n", Body: "b\r\n"}, false).Verdict; v != Match {
		t.Errorf("whitespace and CRLF should be forgiven, got %s", v)
	}
	if v := judge(c, Reading{Loaded: true, Name: "a", Description: "d2", Body: "b"}, false).Verdict; v != Misread {
		t.Errorf("changed description should be MISREAD, got %s", v)
	}
	if v := judge(c, Reading{Loaded: true, Name: "a", Description: "d2", Body: "b"}, true).Verdict; v != Match {
		t.Errorf("load-only mode ignores values, got %s", v)
	}
	if v := judge(Case{Expect: Expect{Load: "no"}}, Reading{Loaded: true}, false).Verdict; v != Accepted {
		t.Errorf("loading a skip case should be ACCEPTED, got %s", v)
	}
}
