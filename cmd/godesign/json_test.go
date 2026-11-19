package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// decode parses the captured stdout as a single JSON document. Every -json run
// must produce exactly one, so anything else is a failure worth naming.
func decode[T any](t *testing.T, out string) T {
	t.Helper()
	var v T
	dec := json.NewDecoder(strings.NewReader(out))
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if dec.More() {
		t.Fatalf("expected exactly one JSON document, got more:\n%s", out)
	}
	return v
}

func TestJSONErrorsAreDocumentsOnStdout(t *testing.T) {
	dir := tinyModule(t)
	path := write(t, "bad.ofn", "Ontology(Frobnicate(<http://x/A>))")
	out, code := capture(t, func() int { return run([]string{"invariants", "-json", "-file", path, dir}) })
	if code != exitProblem {
		t.Errorf("exit = %d, want %d", code, exitProblem)
	}
	doc := decode[errorDoc](t, out)
	if !strings.Contains(doc.Error, "Frobnicate") {
		t.Errorf("error = %q, should name the offending input", doc.Error)
	}
}

func TestJSONBadFlagValueIsAlsoADocument(t *testing.T) {
	dir := tinyModule(t)
	out, code := capture(t, func() int {
		return run([]string{"invariants", "-json", "-fail-on", "nonsense", dir})
	})
	if code != exitProblem {
		t.Errorf("exit = %d, want %d", code, exitProblem)
	}
	if doc := decode[errorDoc](t, out); doc.Error == "" {
		t.Error("want a non-empty error message")
	}
}

func TestJSONDoesNotEscapeAngleBrackets(t *testing.T) {
	// Ontology text is full of <, > and &. Go's encoder escapes those by
	// default, which would make every full IRI unreadable.
	dir := tinyModule(t)
	path := write(t, "bad.ofn", "Ontology(Frobnicate(<http://x/A>))")
	out, _ := capture(t, func() int { return run([]string{"invariants", "-json", "-file", path, dir}) })
	if strings.Contains(out, `<`) || strings.Contains(out, `>`) {
		t.Errorf("output contains HTML-escaped angle brackets:\n%s", out)
	}
}

func TestHookUsageMistakesDoNotBlock(t *testing.T) {
	// Exit status 2 would make Claude Code refuse the tool call.
	for _, args := range [][]string{{"hook"}, {"hook", "nonsense"}, {"hook", "ontology-check"}} {
		if _, code := capture(t, func() int { return run(args) }); code != exitFailed {
			t.Errorf("%v: exit = %d, want %d", args, code, exitFailed)
		}
	}
}
