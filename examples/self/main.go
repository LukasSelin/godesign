// Command self describes godesign's own Go code as an OWL 2 ontology, so that
// gowl's linter has a real, non-toy ontology of this repository to run on.
// Run it from the repository root:
//
//	go run ./examples/self > examples/self/godesign.ofn
//	gowl lint examples/self/godesign.ofn
//
// The generator itself lives in internal/gocode, and "godesign gocode" runs it on
// any other Go module. See that package for the mapping.
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/LukasSelin/godesign/internal/gocode"
)

// pkgPrefix is the name godesign's own packages are abbreviated under
const pkgPrefix = "godesign"

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	o, err := gocode.BuildPrefixed(root, pkgPrefix)
	if err != nil {
		fmt.Fprintln(os.Stderr, "self:", err)
		os.Exit(1)
	}
	w := bufio.NewWriter(os.Stdout)
	if err := o.WriteFunctional(w); err != nil {
		fmt.Fprintln(os.Stderr, "self:", err)
		os.Exit(1)
	}
	if err := w.Flush(); err != nil {
		fmt.Fprintln(os.Stderr, "self:", err)
		os.Exit(1)
	}
}
