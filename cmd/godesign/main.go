// Command godesign describes a Go module's code as an OWL 2 ontology and holds
// changes to that code to a design: plans written before the code, and
// invariants kept on every change. It also runs the Claude Code hooks of the
// godesign plugin.
//
// Usage:
//
//	godesign gocode     [-o file.ofn] [module-dir]
//	godesign sprawl     [-base ref] [-fail-on severity] [-seen file] [-json] [module-dir]
//	godesign plan       [-base ref] [-invariants file] [-fail-on severity] [-json] plan.ofn [module-dir]
//	godesign conform    [-base ref] [-design plan.ofn] [-fail-on severity] [-seen file] [-json] [module-dir]
//	godesign invariants [-file invariants.ofn] [-fail-on severity] [-seen file] [-json] [module-dir]
//	godesign hook design-gate < hook.json
//	godesign hook ontology-check pre|post < hook.json
//	godesign hook ontology-check compare before.ofn after.ofn
//
// Exit status is 0 on success, 1 when a check fails (sprawl, plan, conform or
// invariants findings at or above the fail threshold), and 2 on a usage or I/O
// error. The hook commands answer the way Claude Code expects of a hook
// instead.
//
// Every command except gocode and hook takes -json, which writes one JSON
// object to stdout instead of text — including on failure, so a caller that
// parses stdout always gets a document. Exit statuses are the same in both
// modes. gocode has no -json because its output is an ontology document, not a
// report. The document shapes live in json.go.
//
// The ontologies themselves are read and written by gowl
// (github.com/LukasSelin/gowl), whose own command lints, classifies, diffs and
// formats them.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LukasSelin/godesign/internal/designgate"
	"github.com/LukasSelin/godesign/internal/ontologycheck"
	"github.com/LukasSelin/gowl/owl"
)

const (
	exitOK      = 0
	exitFailed  = 1
	exitProblem = 2
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return exitProblem
	}
	switch args[0] {
	case "gocode":
		return cmdGoCode(args[1:])
	case "sprawl":
		return cmdSprawl(args[1:])
	case "plan":
		return cmdPlan(args[1:])
	case "conform":
		return cmdConform(args[1:])
	case "invariants":
		return cmdInvariants(args[1:])
	case "hook":
		return cmdHook(args[1:])
	case "help", "-h", "--help":
		usage()
		return exitOK
	default:
		fmt.Fprintf(os.Stderr, "godesign: unknown command %q\n\n", args[0])
		usage()
		return exitProblem
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `godesign describes a Go module's code as an OWL 2 ontology, and holds changes
to it to a design.

  godesign gocode     [-o file.ofn] [module-dir]
  godesign sprawl     [-base ref] [-fail-on severity] [-seen file] [-json] [module-dir]
  godesign plan       [-base ref] [-invariants file] [-fail-on severity] [-json] plan.ofn [module-dir]
  godesign conform    [-base ref] [-design plan.ofn] [-fail-on severity] [-seen file] [-json] [module-dir]
  godesign invariants [-file invariants.ofn] [-fail-on severity] [-seen file] [-json] [module-dir]
  godesign hook design-gate | ontology-check pre|post|compare

gocode describes a Go module's code as an ontology: exported types become
classes, struct fields properties, and packages individuals. Give it the
directory holding the module's go.mod (default: the current directory).

sprawl runs gocode on the module and on the same module at -base (default
HEAD), and reports what the change added beside the existing types rather
than through them: a type whose fields repeat another's, and a new type that
connects to nothing that was there.

plan and conform keep a change to a design written before it. A plan is an
ontology naming what the change will add, in the names gocode writes. plan
checks it before any code exists, as though it were already built. conform
compares the code with it both ways: what the plan names that the code lacks,
and what the change added that the plan does not name. Unless -design says
otherwise, the plans are the files under design/ that differ from the base,
which defaults to where this branch left the default branch.

invariants checks the code against design/invariants.ofn: rules about its
packages, what they may import and which interfaces have fixed methods, that
hold on every change rather than for one. plan checks a plan against them too,
so a plan that would break one fails before any code exists.

hook runs the Claude Code hooks of the godesign plugin; see its README.

Every command except gocode and hook takes -json for machine-readable output.
Run "godesign <command> -h" for the flags of one command. To lint, classify,
diff or format an ontology, use gowl.
`)
}

// cmdHook runs one of the plugin's Claude Code hooks. They read the hook
// payload from stdin and answer by Claude Code's rules, where exit status 2
// refuses the tool call. A usage mistake must not do that, so it exits 1, which
// Claude Code shows as an error without blocking anything.
func cmdHook(args []string) int {
	const hookUsage = "usage: godesign hook design-gate < hook.json\n" +
		"       godesign hook ontology-check pre|post < hook.json\n" +
		"       godesign hook ontology-check compare before.ofn after.ofn"
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, hookUsage)
		return exitFailed
	}
	switch args[0] {
	case "design-gate":
		return designgate.Run(os.Stdin, os.Stderr)
	case "ontology-check":
		if len(args) > 1 {
			switch args[1] {
			case "pre":
				return ontologycheck.Pre()
			case "post":
				return ontologycheck.Post()
			case "compare":
				return ontologycheck.Compare(args[2:])
			}
		}
	}
	fmt.Fprintln(os.Stderr, hookUsage)
	return exitFailed
}

// parseArgs parses flags that may appear before, after or between positional
// arguments. Go's flag package stops at the first non-flag argument, which
// makes "godesign sprawl . -json" fail; each round of this loop consumes one
// positional and re-parses the rest, so "-flag value" pairs still bind.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return positional, nil
		}
		positional = append(positional, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// load parses an ontology file, reporting a usable error.
func load(path string) (*owl.Ontology, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	o, err := owl.ParseFunctional(f)
	if err != nil {
		// Parse errors carry "owl: line N"; prefix the filename so editors and
		// CI logs can locate them.
		return nil, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return o, nil
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "godesign:", err)
	return exitProblem
}
