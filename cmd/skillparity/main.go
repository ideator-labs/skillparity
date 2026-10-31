// Command skillparity runs one shared SKILL.md corpus through every loader in
// loaders.json and reports, per case and loader, whether the loader read what
// the spec says the file means.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Loader struct {
	ID      string   `json:"id"`
	What    string   `json:"what"`
	Version string   `json:"version"`
	Cmd     []string `json:"cmd"`
}

type Expect struct {
	Load        string `json:"load"` // "yes", "no", or "either" (contested, reported but not scored)
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body,omitempty"`
}

type Case struct {
	ID       string `json:"id"`
	SkillDir string `json:"skill_dir"`
	Expect   Expect `json:"expect"`
	Note     string `json:"note"`
}

type Reading struct {
	Loader      string `json:"loader"`
	Case        string `json:"case"`
	Loaded      bool   `json:"loaded"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Body        string `json:"body,omitempty"`
	Error       string `json:"error,omitempty"`
}

type Verdict string

const (
	Match     Verdict = "MATCH"     // loaded and read the expected values
	Skipped   Verdict = "SKIPPED"   // refused a file the spec says to skip
	Misread   Verdict = "MISREAD"   // loaded, but a value differs from the spec reading
	Refused   Verdict = "REFUSED"   // refused a file the spec says to load
	Accepted  Verdict = "ACCEPTED"  // loaded a file the spec says to skip
	Contested Verdict = "CONTESTED" // spec does not decide; reported, not scored
)

type Cell struct {
	Case    string   `json:"case"`
	Loader  string   `json:"loader"`
	Verdict Verdict  `json:"verdict"`
	Fields  []string `json:"fields,omitempty"`
	Reading Reading  `json:"reading"`
}

type Report struct {
	CorpusDigest string   `json:"corpus_digest"`
	Mode         string   `json:"mode"`
	Loaders      []Loader `json:"loaders"`
	Cases        []Case   `json:"cases"`
	Cells        []Cell   `json:"cells"`
}

func main() {
	corpus := flag.String("corpus", "corpus", "corpus directory")
	registry := flag.String("loaders", "loaders.json", "loader registry")
	only := flag.String("only", "", "comma-separated loader ids to run (default all)")
	loadOnly := flag.Bool("load-only", false, "judge loaders only on load or skip, ignoring the values they read")
	out := flag.String("out", "report.json", "report path")
	flag.Parse()

	loaders, err := readLoaders(*registry, *only)
	check(err)
	cases, err := readCases(*corpus)
	check(err)
	digest, err := corpusDigest(*corpus)
	check(err)

	abs, err := filepath.Abs(*corpus)
	check(err)
	rep := Report{CorpusDigest: digest, Mode: "values", Loaders: loaders, Cases: cases}
	if *loadOnly {
		rep.Mode = "load-only"
	}
	for _, l := range loaders {
		readings, err := run(l, abs)
		check(err)
		for _, c := range cases {
			r, ok := readings[c.ID]
			if !ok {
				check(fmt.Errorf("loader %s returned no reading for %s", l.ID, c.ID))
			}
			rep.Cells = append(rep.Cells, judge(c, r, *loadOnly))
		}
	}

	b, _ := json.MarshalIndent(rep, "", "  ")
	check(os.WriteFile(*out, append(b, '\n'), 0o644))
	printMatrix(rep)
}

func judge(c Case, r Reading, loadOnly bool) Cell {
	cell := Cell{Case: c.ID, Loader: r.Loader, Reading: r}
	switch {
	case c.Expect.Load == "either":
		cell.Verdict = Contested
	case c.Expect.Load == "no" && r.Loaded:
		cell.Verdict = Accepted
	case c.Expect.Load == "no":
		cell.Verdict = Skipped
	case !r.Loaded:
		cell.Verdict = Refused
	default:
		cell.Verdict = Match
		if !loadOnly {
			cell.Fields = diff(c.Expect, r)
			if len(cell.Fields) > 0 {
				cell.Verdict = Misread
			}
		}
	}
	return cell
}

func diff(e Expect, r Reading) []string {
	var fields []string
	if norm(e.Name) != norm(r.Name) {
		fields = append(fields, "name")
	}
	if norm(e.Description) != norm(r.Description) {
		fields = append(fields, "description")
	}
	if norm(e.Body) != norm(r.Body) {
		fields = append(fields, "body")
	}
	return fields
}

// norm forgives what the spec leaves open: line endings and surrounding whitespace.
func norm(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

func run(l Loader, corpus string) (map[string]Reading, error) {
	args := make([]string, len(l.Cmd))
	for i, a := range l.Cmd {
		args[i] = strings.ReplaceAll(a, "{corpus}", corpus)
	}
	cmd := exec.Command(args[0], args[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("loader %s: %v\n%s", l.ID, err, stderr.String())
	}
	readings := map[string]Reading{}
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r Reading
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("loader %s: bad reading %q: %v", l.ID, sc.Text(), err)
		}
		if r.Loader != l.ID {
			return nil, fmt.Errorf("loader %s: reading labelled %q", l.ID, r.Loader)
		}
		readings[r.Case] = r
	}
	return readings, sc.Err()
}

func readLoaders(path, only string) ([]Loader, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var all []Loader
	if err := json.Unmarshal(b, &all); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	if only == "" {
		return all, nil
	}
	want := map[string]bool{}
	for _, id := range strings.Split(only, ",") {
		want[strings.TrimSpace(id)] = true
	}
	var picked []Loader
	for _, l := range all {
		if want[l.ID] {
			picked = append(picked, l)
			delete(want, l.ID)
		}
	}
	if len(want) > 0 {
		return nil, fmt.Errorf("unknown loader ids in -only: %v", keys(want))
	}
	return picked, nil
}

func readCases(dir string) ([]Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var cases []Case
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name(), "case.json"))
		if err != nil {
			return nil, err
		}
		var c Case
		if err := json.Unmarshal(b, &c); err != nil {
			return nil, fmt.Errorf("%s/case.json: %v", e.Name(), err)
		}
		if c.ID != e.Name() {
			return nil, fmt.Errorf("%s/case.json: id %q does not match directory", e.Name(), c.ID)
		}
		switch c.Expect.Load {
		case "yes", "no", "either":
		default:
			return nil, fmt.Errorf("%s: expect.load must be yes, no or either", c.ID)
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), c.SkillDir, "SKILL.md")); err != nil {
			return nil, err
		}
		cases = append(cases, c)
	}
	return cases, nil
}

// corpusDigest hashes every path and byte in the corpus so two reports only
// compare when they ran the same files.
func corpusDigest(dir string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), len(b))
		h.Write(b)
		return nil
	})
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), err
}

func printMatrix(rep Report) {
	short := map[Verdict]string{Match: "ok", Skipped: "skip", Misread: "MISREAD", Refused: "REFUSED", Accepted: "ACCEPTED", Contested: "~"}
	byKey := map[string]Cell{}
	for _, c := range rep.Cells {
		byKey[c.Case+"|"+c.Loader] = c
	}
	fmt.Printf("corpus %s  mode %s\n", rep.CorpusDigest, rep.Mode)
	fmt.Printf("%-28s", "case")
	for _, l := range rep.Loaders {
		fmt.Printf("%-13s", l.ID)
	}
	fmt.Println()
	for _, c := range rep.Cases {
		fmt.Printf("%-28s", c.ID)
		for _, l := range rep.Loaders {
			cell := byKey[c.ID+"|"+l.ID]
			s := short[cell.Verdict]
			if cell.Verdict == Contested {
				s = "~skip"
				if cell.Reading.Loaded {
					s = "~load"
				}
			}
			fmt.Printf("%-13s", s)
		}
		fmt.Println()
	}
}

func keys(m map[string]bool) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "skillparity:", err)
		os.Exit(2)
	}
}
