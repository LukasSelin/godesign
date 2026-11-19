package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LukasSelin/godesign/internal/gocode"
	"github.com/LukasSelin/gowl/lint"
	"github.com/LukasSelin/gowl/owl"
)

// The documents in this file are the machine-readable contract of the CLI:
// -json makes a command write exactly one JSON object to stdout, including when
// it fails. Field names are snake_case, lists are always present rather than
// null, and axioms are rendered with the ontology's own prefixes so they match
// the source file.
//
// -json changes the format, not the content: flags that select how much detail
// to report (diff -summary, profile -v) mean the same thing in both modes.

// writeJSON emits v as an indented JSON document followed by a newline.
func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	// Ontology text is full of <, > and &; HTML escaping would render every
	// IRI and message unreadable for no benefit here.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(os.Stderr, "godesign:", err)
	}
}

// failWith reports an error as JSON when the caller asked for it, so a harness
// always gets one parseable object on stdout. The exit status is unchanged.
func failWith(jsonOut bool, err error) int {
	if jsonOut {
		writeJSON(errorDoc{Error: err.Error()})
		return exitProblem
	}
	return fail(err)
}

type errorDoc struct {
	Error string `json:"error"`
}

// --- lint -------------------------------------------------------------------

type lintDoc struct {
	File     string       `json:"file"`
	Findings []findingDoc `json:"findings"`
	Summary  lintSummary  `json:"summary"`
}

type findingDoc struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Subject  string `json:"subject,omitempty"`
	Message  string `json:"message"`
	Axiom    string `json:"axiom,omitempty"`
}

type lintSummary struct {
	Total int `json:"total"`
	// BySeverity always carries all three keys, so a consumer never has to
	// distinguish "no errors" from "field absent".
	BySeverity map[string]int `json:"by_severity"`
	Worst      string         `json:"worst,omitempty"`
	// Failed reports whether the findings met the -fail-on threshold.
	Failed bool `json:"failed"`
}

func lintJSON(file string, o *owl.Ontology, findings []lint.Finding, threshold lint.Severity) lintDoc {
	docs := make([]findingDoc, 0, len(findings))
	bySeverity := map[string]int{
		lint.Info.String():    0,
		lint.Warning.String(): 0,
		lint.Error.String():   0,
	}
	for _, f := range findings {
		d := findingDoc{
			Rule:     f.Rule,
			Severity: f.Severity.String(),
			Subject:  string(f.Subject),
			Message:  f.Message,
		}
		if f.Axiom != nil {
			d.Axiom = o.Render(f.Axiom)
		}
		docs = append(docs, d)
		bySeverity[f.Severity.String()]++
	}

	summary := lintSummary{Total: len(findings), BySeverity: bySeverity}
	if worst, any := lint.MaxSeverity(findings); any {
		summary.Worst = worst.String()
		summary.Failed = worst >= threshold
	}
	return lintDoc{File: file, Findings: docs, Summary: summary}
}

func renderClasses(o *owl.Ontology, classes []owl.Class) []string {
	out := make([]string, 0, len(classes))
	for _, cl := range classes {
		out = append(out, string(cl))
	}
	return out
}

// --- sprawl -----------------------------------------------------------------

type sprawlDoc struct {
	Dir string `json:"dir"`
	// Base is the -base flag as given; BaseCommit is what it resolved to.
	Base       string `json:"base"`
	BaseCommit string `json:"base_commit"`
	// Added and Changed are class IRIs: the types the change introduced, and
	// the existing types whose fields it changed.
	Added    []string     `json:"added"`
	Changed  []string     `json:"changed"`
	Findings []findingDoc `json:"findings"`
	// AlreadyReported counts the findings -seen left out. The summary covers
	// only the findings reported.
	AlreadyReported int         `json:"already_reported"`
	Summary         lintSummary `json:"summary"`
}

// planDoc is what godesign plan writes: what godesign sprawl would say about the plan
// if it were already code, and which plan that was. Invariants names the file
// the plan was also checked against, or is empty when the module has none;
// InvariantsChecked counts its rules.
type planDoc struct {
	Design            string `json:"design"`
	Invariants        string `json:"invariants"`
	InvariantsChecked int    `json:"invariants_checked"`
	sprawlDoc
}

// invariantsDoc is what godesign invariants writes. Invariants is empty when the
// module has none.
type invariantsDoc struct {
	Dir        string       `json:"dir"`
	Invariants string       `json:"invariants"`
	Checked    int          `json:"checked"`
	Findings   []findingDoc `json:"findings"`
	// AlreadyReported counts the findings -seen left out. The summary covers
	// only the findings reported.
	AlreadyReported int         `json:"already_reported"`
	Summary         lintSummary `json:"summary"`
}

func invariantsJSON(dir, path string, o *owl.Ontology, c gocode.Conformance, findings []lint.Finding, already int, threshold lint.Severity) invariantsDoc {
	l := lintJSON(dir, o, findings, threshold)
	return invariantsDoc{
		Dir:             dir,
		Invariants:      path,
		Checked:         c.Checked,
		Findings:        l.Findings,
		AlreadyReported: already,
		Summary:         l.Summary,
	}
}

// conformDoc is what godesign conform writes.
type conformDoc struct {
	Dir string `json:"dir"`
	// Base is how the base was named; BaseCommit is what it resolved to.
	Base       string `json:"base"`
	BaseCommit string `json:"base_commit"`
	// Designs are the plan files compared with the code.
	Designs []string `json:"designs"`
	// Checked counts the planned items compared with the code. Unchecked lists
	// the terms the plans name outside the module.
	Checked   int          `json:"checked"`
	Unchecked []string     `json:"unchecked"`
	Findings  []findingDoc `json:"findings"`
	// AlreadyReported counts the findings -seen left out. The summary covers
	// only the findings reported.
	AlreadyReported int         `json:"already_reported"`
	Summary         lintSummary `json:"summary"`
}

func conformJSON(dir, base, commit string, paths []string, o *owl.Ontology, c gocode.Conformance, findings []lint.Finding, already int, threshold lint.Severity) conformDoc {
	l := lintJSON(dir, o, findings, threshold)
	designs := make([]string, len(paths))
	for i, p := range paths {
		designs[i] = filepath.ToSlash(p)
	}
	unchecked := make([]string, len(c.Unchecked))
	for i, iri := range c.Unchecked {
		unchecked[i] = string(iri)
	}
	return conformDoc{
		Dir:             dir,
		Base:            base,
		BaseCommit:      commit,
		Designs:         designs,
		Checked:         c.Checked,
		Unchecked:       unchecked,
		Findings:        l.Findings,
		AlreadyReported: already,
		Summary:         l.Summary,
	}
}

func sprawlJSON(dir, base, commit string, o *owl.Ontology, d gocode.Delta, findings []lint.Finding, already int, threshold lint.Severity) sprawlDoc {
	l := lintJSON(dir, o, findings, threshold)
	return sprawlDoc{
		Dir:             dir,
		Base:            base,
		BaseCommit:      commit,
		Added:           renderClasses(o, d.Added),
		Changed:         renderClasses(o, d.Changed),
		Findings:        l.Findings,
		AlreadyReported: already,
		Summary:         l.Summary,
	}
}
