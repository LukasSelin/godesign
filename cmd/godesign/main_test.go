package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

const cleanDoc = `Prefix(:=<http://example.org/o#>)
Ontology(<http://example.org/o>
    Declaration(Class(:Root))
    Declaration(Class(:Child))
    AnnotationAssertion(rdfs:label :Root "Root"^^xsd:string)
    AnnotationAssertion(rdfs:label :Child "Child"^^xsd:string)
    SubClassOf(:Child :Root)
)
`

const cyclicDoc = `Prefix(:=<http://example.org/o#>)
Ontology(<http://example.org/o>
    Declaration(Class(:A))
    Declaration(Class(:B))
    SubClassOf(:A :B)
    SubClassOf(:B :A)
)
`

// write puts content in a temp file and returns its path.
func write(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// capture runs f with stdout and stderr redirected, returning what it printed.
func capture(t *testing.T, f func() int) (string, int) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = w, w
	defer func() { os.Stdout, os.Stderr = stdout, stderr }()

	code := f()
	w.Close()
	out, _ := io.ReadAll(r)
	return string(out), code
}

func TestUnknownCommand(t *testing.T) {
	_, code := capture(t, func() int { return run([]string{"frobnicate"}) })
	if code != exitProblem {
		t.Errorf("exit = %d, want 2", code)
	}
	_, code = capture(t, func() int { return run(nil) })
	if code != exitProblem {
		t.Errorf("no arguments: exit = %d, want 2", code)
	}
}
