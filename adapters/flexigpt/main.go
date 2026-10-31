// Command flexigpt reads every corpus case with github.com/flexigpt/agentskills-go
// and prints one JSON reading per line.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/flexigpt/agentskills-go/document"
)

type reading struct {
	Loader      string `json:"loader"`
	Case        string `json:"case"`
	Loaded      bool   `json:"loaded"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body,omitempty"`
	Error       string `json:"error,omitempty"`
}

func main() {
	root := os.Args[1]
	cases, err := os.ReadDir(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		r := reading{Loader: "flexigpt", Case: c.Name()}
		raw, err := os.ReadFile(skillFile(filepath.Join(root, c.Name())))
		if err == nil {
			var doc document.SkillDocument
			doc, _, err = document.ParseSkillDocument(raw, document.ParseSkillDocumentOptions{})
			if err == nil {
				r.Loaded, r.Name, r.Description, r.Body = true, doc.Name, doc.Description, doc.MarkdownBody
			}
		}
		if err != nil {
			r.Error = err.Error()
		}
		enc.Encode(r)
	}
}

func skillFile(caseDir string) string {
	entries, _ := os.ReadDir(caseDir)
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	if len(dirs) == 0 {
		return filepath.Join(caseDir, "SKILL.md")
	}
	return filepath.Join(caseDir, dirs[0], "SKILL.md")
}
