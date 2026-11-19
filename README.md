# godesign

godesign describes a Go module's code as an OWL 2 ontology and holds changes
to that code to a design: a plan written before the code, and invariants the
code keeps on every change. It ships a Claude Code plugin that applies both
while an agent writes the code.

It is built on [gowl](https://github.com/LukasSelin/gowl), the OWL 2 library
for Go that reads, writes and reasons over the ontologies, and it started
inside gowl's repository. It moved out so that gowl stays an OWL library.
gowl's own command lints, classifies, diffs and formats the files godesign
writes.

```bash
go install github.com/LukasSelin/godesign/cmd/godesign@latest
```

| Command | What it does |
|---|---|
| `godesign gocode` | describes a module's types, fields, interfaces and imports as an ontology |
| `godesign sprawl` | reports a change that grows the module beside its types instead of through them |
| `godesign plan` | checks a plan, `design/<change>.ofn`, before any code exists |
| `godesign conform` | compares the code with the plan, both ways |
| `godesign invariants` | checks the code against `design/invariants.ofn` |
| `godesign hook` | runs the plugin's Claude Code hooks |

## The Claude Code plugin

This repository is a Claude Code plugin marketplace. With `godesign` on your
`PATH`:

```bash
claude plugin marketplace add LukasSelin/godesign
claude plugin install godesign@godesign
```

The plugin, in [`plugin/`](plugin), adds two skills and the hooks behind them:

- `/godesign:design-first` plans a Go change as an ontology, checks the plan,
  builds to it and verifies the result.
- `/godesign:new-term` adds a class or property to an ontology with the
  reasoner in the loop.
- Before an `Edit` or `Write`, `design-gate` refuses a new exported type no plan
  names.
- After an edit to a `.ofn` file, gowl's linter and EL reasoner report what the
  edit changed, and block one that breaks the ontology.
- When a turn ends, `godesign sprawl`, `conform` and `invariants` hand anything
  they find back to the agent, once each.

The Go checks apply only to a module with a `design/` directory, so installing
the plugin for your user changes nothing in a repository that never asked for
plans. Create the directory, or write the first plan into it, to opt in. The
ontology checks apply to any `.ofn` file.

The hooks run the `godesign` on your `PATH`. When it is missing they skip their
checks, and a session in an opted-in module says so when it starts. In this
repository they build `./cmd/godesign` from the working tree instead, so a
change to a check is held to the check as changed.

## How Go code maps to OWL

[`examples/self`](examples/self/main.go) describes this repository, and
[`examples/self/godesign.ofn`](examples/self/godesign.ofn) is the result; its
tests fail when the file no longer matches the code, when gowl's linter finds a
warning in it, or when the code breaks its own invariants.

- Each exported Go type becomes a class.
- A type that satisfies one of the module's interfaces becomes a `SubClassOf`
  that interface.
- Struct fields become object or data properties, with domains and ranges.
  An embedded field counts, under its type's name.
- Packages become `:Package` individuals, linked by `:imports` to the packages
  they import. A package from outside the module, such as `fmt`, is an
  `:ExternalPackage`.
- An interface lists the methods it declares, with the annotation `:method`.

## Describing a Go module

The generator behind `examples/self` lives in
[`internal/gocode`](internal/gocode/gocode.go), and `godesign gocode` runs it on any
Go module. Give it the directory that holds the module's `go.mod`; it defaults
to the current directory:

```bash
godesign gocode ../doppel > doppel.ofn
godesign gocode -o doppel.ofn ../doppel   # the same, without shell redirection
gowl lint doppel.ofn
```

To put the command on your `PATH`:

```bash
go install github.com/LukasSelin/godesign/cmd/godesign@latest
```

The ontology is named by the module path, here `<https://github.com/LukasSelin/doppel>`.
The mapping is the one [How Go code maps to OWL](#how-go-code-maps-to-owl) describes. The module has to build. Its own packages
are read from source. Everything they import is read from the export data that
`go list -export` produces inside that module, standard library and
third-party modules alike, so its dependencies must already be in the module
cache. Directories that hold their own `go.mod` are other modules and are
skipped, as the go tool skips them.

Besides its types, the description records each package's direct imports, and
the methods of each interface:

- A package the module imports from outside, standard library or dependency,
  is an `:ExternalPackage` individual, such as `:package.net.http`. Only the
  module's own imports are recorded, not what those packages import in turn.
  Whether a package is the module's own is decided by the packages found under
  its root, so a nested module counts as outside.
- An interface lists each method it declares with the annotation `:method`, as
  `Name(parameter types) result types`, for example
  `AnnotationAssertion(:method :owl.Entity "Kind() owl.Kind")`. Parameter names
  are left out, a type is qualified by its package's name as the classes are,
  and `byte`, `rune` and `interface{}` are written `uint8`, `int32` and `any`.
  An unexported method counts, because one such as `owl.Axiom`'s `isAxiom`
  seals the interface: only its own package can implement it. An embedded
  interface of the module is a `SubClassOf` instead, and an embedded one from
  outside it, such as `io.Reader`, adds its methods.
- Imports are read from the files that build on the machine running gocode, so
  an import that only a file such as `x_windows.go` makes shows up on Windows
  alone.

One importer serves the standard library and the dependencies, so each package
is loaded once. A cobra command's `context.Context` is therefore the same type
as the module's own. Loading the standard library and third-party code through
two importers would produce two different `context` packages, and the module
would fail to type-check.

## Catching sprawl

A codebase sprawls when a change adds new types next to the existing ones
instead of building on them: a struct copied under a new name, or a type
that nothing else uses. `godesign sprawl` looks for both in a change. It runs
`gocode` on the module twice, once on the working tree and once at `-base`
(default `HEAD`), and compares the two ontologies. The base version is copied
out with `git archive`, so the working tree, the index and other worktrees are
not touched.

- **`same-shape-type`** (warning) reports a type the change added or reshaped
  whose fields match another type's. Fields are compared by type and not by
  name, because a copy is usually renamed. The type may have the same fields
  as the other one (reuse it), every field of it plus more (embed it), or only
  fields it also has (use it instead). A match has to share at least two
  fields, and one of them has to be of a type the module defines.
- **`new-orphan`** (info) reports a new type that connects to no type that
  existed before. It satisfies none of the module's interfaces, no type
  satisfies it, and no field links it to an existing type in either direction.

On the module from the tests, with `Holding` and `Memo` added to `Item` and
`Stock`:

```
$ godesign sprawl
compared with HEAD (ea4c786a2d2d): 2 new type(s), 0 changed
  new:     :store.Holding, :store.Memo

warning: same-shape-type: :store.Holding has the same fields as :store.Stock (:store.Item, xsd:integer); consider reusing it
info: new-orphan: new type :store.Memo connects to no type that existed before: no interface, implementation or field links it to one

2 finding(s)
```

The checks look only at what the change did. Checked as if every type were
new, 32 of gowl's types would match another of its types. Most are OWL
types: `ObjectSomeValuesFrom` and `ObjectAllValuesFrom` have the same fields
because the OWL specification gives them the same arguments. Reporting all of
those on every run would hide the one copy a change has just added. For the
same reason, a match that already held before the change is not reported
again just because one of its two types changed.

The checks see only what `gocode` records:

- exported types;
- fields whose type is a module type (directly, or through a pointer or slice)
  or a basic type.

Maps, functions and methods are not recorded. A type that is only ever a
function parameter, such as an options struct, therefore looks unconnected,
which is why `new-orphan` is only `info`. Duplicate functions are a job for a
clone detector.

Nothing reaches the default `-fail-on error`, so in CI pass `-fail-on warning`
and a base that covers the whole branch:

```bash
godesign sprawl -base origin/main -fail-on warning
```

A run takes about 3.5 seconds on this repository. Most of that is
type-checking the base copy.

### Before an agent finishes

[`plugin/hooks/sprawl-check.sh`](plugin/hooks/sprawl-check.sh) is a Claude
Code `Stop` hook of the [plugin](#the-claude-code-plugin). In a module that
has a `design/` directory, when a turn ends with Go changes since `HEAD`, or since `GODESIGN_SPRAWL_BASE` if
that is set, the hook runs `godesign sprawl` and passes any findings back to the
agent. The agent then reworks the change to reuse, embed or implement what is
already there, or tells the user why a new type is the right call.

`-seen` records each finding in a per-session file, so each finding is
reported only once. That also means the hook cannot keep the agent looping. A
turn whose Go changes were already checked costs about half a second. If the
tree does not build, the hook skips the check.

## Planning a change before writing it

`godesign sprawl` looks at a change after it is written. `godesign plan` and
`godesign conform` start earlier. The plan for a change is written first, as an
ontology in `design/<change>.ofn` next to the module's `go.mod`, and the code is
then held to it.

A plan uses the names `gocode` writes, so it can be compared with the code. It
names only what the change adds or alters, plus the existing types those
additions point at:

```
Prefix(:=<https://example.com/store#>)
Prefix(xsd:=<http://www.w3.org/2001/XMLSchema#>)

Ontology(<https://example.com/store/design/holdings>
    Declaration(Class(:store.Item))

    Declaration(Class(:store.Holding))
    Declaration(ObjectProperty(:store.Holding.What))
    ObjectPropertyDomain(:store.Holding.What :store.Holding)
    ObjectPropertyRange(:store.Holding.What :store.Item)
    Declaration(DataProperty(:store.Holding.Amount))
    DataPropertyDomain(:store.Holding.Amount :store.Holding)
    DataPropertyRange(:store.Holding.Amount xsd:integer)
)
```

`godesign plan` reads the module as though the plan were already code, so the checks
from `godesign sprawl` run before anything is built. It also reports what the plan
adds to the linter's findings, leaving out `orphan-class` and `missing-label`. A
name the plan uses and never declares, usually a typo, shows up as
`undeclared-entity`.

```
$ godesign plan -fail-on warning design/holdings.ofn
compared with main merge-base (c5bb468eaa49): 1 new type(s), 0 changed
  new:     :store.Holding

warning: same-shape-type: :store.Holding has the same fields as :store.Stock (:store.Item, xsd:integer); consider reusing it

1 finding(s)
```

An interface's methods are its contract, and a plan that adds or changes one
names it. Parameter names and spacing are the writer's own, and a bare type of
the interface's package, such as `Item` in package `store`, means `store.Item`:

```
    Declaration(Class(:store.Priced))
    AnnotationAssertion(:method :store.Priced "Price(item Item) (cents int, err error)")
```

`godesign conform` compares the code with the plan, both ways:

| Rule | Severity | Reports |
|---|---|---|
| `missing-type` | error | a type the plan names and the code does not have |
| `missing-field` | error | a field the plan names and the code does not have |
| `field-mismatch` | error | a field on another struct, or holding another type, than the plan says |
| `missing-interface` | error | a type that does not satisfy an interface the plan says it satisfies |
| `missing-package` | error | a package the plan names and the module does not have |
| `missing-import` | error | an import between packages that the plan names and the code does not have |
| `unplanned-type` | error | a type the change added that no plan names |
| `unplanned-package` | error | a package the change added that no plan names |
| `unplanned-field` | warning | a field the change added or retyped that no plan names |
| `unplanned-import` | warning | an import the change added that no plan names |
| `missing-method` | error | a method the plan gives an interface that the code does not declare |
| `method-mismatch` | error | a method the code declares with another signature than the plan's |
| `unplanned-method` | error | a method the change added to an interface, or changed, that no plan names |
| `malformed-method` | error | a method in a plan that is not a signature |

A plan is about the module's own structure, so the change's imports from outside
the module are not reported: what the module may depend on is a rule for the
[invariants](#invariants-that-outlive-a-change). A plan that does name one, as
`ObjectPropertyAssertion(:imports :package.example.com.store :package.strings)`, has it
checked.

Suppose the code leaves out `Amount` and adds a type nobody planned:

```
$ godesign conform
compared with main merge-base (c5bb468eaa49): 1 plan(s), 4 planned item(s) checked
  plans:     design/holdings.ofn

error: missing-field: the plan names field :store.Holding.Amount, and the code does not have it as an exported field of a module or basic type
error: unplanned-type: type :store.Memo is new, and no plan names it

2 finding(s)
```

The plans are the files under `design/` that differ from the base, or the ones
named with `-design`. The base defaults to where this branch left the default
branch, so a plan committed partway through a branch still counts. Once merged, a
plan is part of the base and is no longer checked: it records a decision, and
later changes write their own. For the same reason, a plan has to reach the
default branch together with its code, or be named with `-design`. A change that
adds no exported structure, such as most bug fixes, passes with no plan at all.

A plan can only say as much as `gocode` records:

- **Structure only.** Interfaces' methods are in it; functions, the methods of
  other types, and behaviour are not, so a plan does not replace tests.
- **Exported types and some of their fields.** A field of a map, a function or a
  type from outside the module is not recorded, so a plan cannot name one and
  `conform` does not see one being added.
- **Additions only.** A plan says what should exist, so removing something is not
  reported. The invariants can fix an interface's methods so that removing one
  is.
- **Go only.** `gocode` reads Go modules.

### Holding an agent to the plan

Three pieces of the [plugin](#the-claude-code-plugin) make this the order Claude
Code works in, in a module that has a `design/` directory.
They run only in Claude Code sessions, so to hold everyone to a plan, also run
`godesign conform -fail-on warning` in CI.

- The [`design-first`](plugin/skills/design-first/SKILL.md) skill walks through
  writing the plan, checking it, building to it and verifying the result.
- [`plugin/hooks/design-gate.sh`](plugin/hooks/design-gate.sh) is a `PreToolUse`
  hook that refuses an `Edit` or `Write` adding an exported type no `design/*.ofn`
  file names. A bug fix adds no types, so it never notices one. It lets through
  anything it cannot read, and `GODESIGN_GATE=off` turns it off.
- [`plugin/hooks/conform-check.sh`](plugin/hooks/conform-check.sh) is a `Stop`
  hook that runs `godesign conform` when a turn ends with Go changes since the base,
  or since `GODESIGN_DESIGN_BASE` if that is set. It reports each finding once, as
  the sprawl hook does. A turn that only wrote a plan has nothing to compare yet,
  and a tree that does not build skips the check.

The gate sees only `Edit` and `Write`, and only types. Anything changed through a
shell command, and any field, import or package, is caught at the end of the turn
instead.

## Invariants that outlive a change

A plan is about one change, says only what should exist, and is retired once it
is merged. Some decisions are not like that: the core does no I/O, `owl` imports
nothing else in the module, a new package is put in a layer on purpose. Those go
in `design/invariants.ofn`, which is never retired. `godesign invariants` checks the
code against it, and `godesign plan` checks every plan against it, so a plan that
would break a rule fails while it is still a few lines of a design.

```
Prefix(:=<https://example.com/shop#>)
Prefix(rdfs:=<http://www.w3.org/2000/01/rdf-schema#>)

Ontology(<https://example.com/shop/design/invariants>
    # Groups: facts the code cannot state.
    ClassAssertion(:Core :package.example.com.shop.core)
    ClassAssertion(:Core :package.example.com.shop.billing)
    ClassAssertion(:Edge :package.example.com.shop.api)
    ClassAssertion(:Effectful :package.os)
    ClassAssertion(:Effectful :package.net.http)

    # A derived property: imports, followed any distance.
    SubObjectPropertyOf(:imports :dependsOn)
    TransitiveObjectProperty(:dependsOn)

    # Rules.
    DisjointClasses(Annotation(rdfs:comment "The core does no I/O")
        :Core ObjectSomeValuesFrom(:dependsOn :Effectful))
    SubClassOf(Annotation(rdfs:comment "Every package is placed on purpose")
        :Package ObjectUnionOf(:Core :Edge))

    # A contract: these are the methods of core.Store, now and later.
    AnnotationAssertion(:method :core.Store "Get(id int) (Thing, error)")
)
```

When billing starts to use the api, the core reaches the network through it:

```
$ godesign invariants
checked against design/invariants.ofn: 3 rule(s)

error: invariant-violated: The core does no I/O: :package.example.com.shop.billing is :Core, and :package.example.com.shop.billing :dependsOn :package.net.http, which is :Effectful (through :package.example.com.shop.billing →:imports :package.example.com.shop.api →:imports :package.net.http)

1 finding(s)
```

The rule comes first, in its own words, then the package that breaks it, and
the imports to blame. That is a counterexample, the way a model checker gives
one, and it is the shortest the facts allow.

**How it reads the file.** A reasoner reads axioms open-world: that a package is
not said to import `os` does not mean it does not. A description of code is
different. It is complete for what it records, so the check is closed-world:
what the code does not do, it does not do. Each axiom has one of four jobs:

- **The code's facts** are which packages there are and what each imports. Only
  the code says so: an assertion about `:Package`, `:ExternalPackage` or
  `:imports` in the file is refused. To forbid an import, use
  `NegativeObjectPropertyAssertion`.
- **Groups** are `ClassAssertion`s with a class of the file's own, such as
  `:Core`. `ClassAssertion(:ExternalPackage …)` declares a package the module
  does not depend on yet, so that a rule can forbid it.
- **Derived properties** come from `SubObjectPropertyOf` between named
  properties and `TransitiveObjectProperty`. They may only define a property
  the facts do not state, so a rule can define `:dependsOn` from `:imports`,
  but cannot add an import.
- **Rules** are `SubClassOf`, `DisjointClasses` and
  `NegativeObjectPropertyAssertion`, over named classes, `owl:Thing`,
  `owl:Nothing`, `ObjectIntersectionOf`, `ObjectUnionOf`, `ObjectComplementOf`,
  `ObjectOneOf`, `ObjectSomeValuesFrom`, `ObjectAllValuesFrom` and
  `ObjectHasValue`, with a property or its `ObjectInverseOf`.

A `:method` annotation fixes an interface's methods: the code must have exactly
those, so a method added, changed or removed is reported on every run, unlike
in a plan.

The subject of a `SubClassOf`, and each operand of a `DisjointClasses`, has to
be bounded by the facts: a named class, `ObjectOneOf`, `ObjectHasValue`,
`ObjectSomeValuesFrom`, or an intersection or union built from those.
`owl:Thing`, `ObjectComplementOf` and `ObjectAllValuesFrom` are refused there,
because who they include depends on which names happen to be mentioned, and a
rule over them could change its answer when an unrelated package is named.

| Rule | Severity | Reports |
|---|---|---|
| `invariant-violated` | error | a rule the code breaks, once for each package that breaks it |
| `unsupported-invariant` | error | an axiom outside what the checker reads, with the reason; it is an error because a rule nobody checks reads as if it held |
| `unknown-package` | error | a package that is no package of the module, the standard library, a dependency or a plan for the change, and is not declared an `:ExternalPackage`: usually a misspelling, which in a rule that forbids a package quietly forbids nothing |
| `empty-term` | warning | a group or property the rules use that nothing defines, so they hold or fail for no package |
| `missing-method`, `method-mismatch`, `unplanned-method` | error | a fixed interface whose methods differ from the file's |

**What it can and cannot see.** It sees packages and their direct imports, and
interfaces' methods. Imports are a proxy for effects, and a coarse one: `fmt`
both formats strings and prints them. Packages from outside the module have no
imports of their own in the facts, so `:dependsOn` follows the module's own
packages and stops at the first package outside it.

**The checker.** [`internal/icv`](internal/icv/icv.go) does the checking. It
knows nothing about Go and does no I/O; [`CheckInvariants`](internal/gocode/invariants.go)
gives it the facts and turns what it reports into findings. Its tests compare
it, on 60 random finite models, with the semantics written out the slow,
obvious way. They also check:

- that a universal is a negated existential of the negation, and that De
  Morgan holds;
- that the derived transitive property is exactly what a search reaches, with
  the shortest path as its proof;
- that every path is a chain of asserted facts;
- that naming an unrelated package changes no answer;
- that the order the rules are written in changes nothing.

As with the reasoner, the batch has to contain both broken and kept rules, so
the comparison cannot pass vacuously.

**godesign's own.** [`design/invariants.ofn`](design/invariants.ofn) holds the
following rules:

- every package is internal or a tool;
- the pure internal packages never reach `os`, the network, `os/exec` or
  logging;
- `internal/icv`, the checker, imports nothing of godesign's: gowl's `owl` and
  the standard library are all it builds on.

gowl keeps its own, about its layers, in its repository.

`TestOwnInvariantsHold` in [`examples/self`](examples/self/main_test.go) keeps
them in `go test ./...`.

[`plugin/hooks/invariants-check.sh`](plugin/hooks/invariants-check.sh) is a
`Stop` hook. When a turn ends with changes to the Go code or to the invariants
themselves, it runs `godesign invariants` and hands any finding back to the agent,
once.


## Layout

| Path | What it holds |
|---|---|
| `cmd/godesign` | the command |
| `internal/gocode` | describing a module, and comparing it with a plan, a base commit or its invariants |
| `internal/icv` | the closed-world checker behind invariants |
| `internal/designgate`, `internal/ontologycheck` | the plugin's hooks, run by `godesign hook` |
| `examples/self` | godesign describing itself |
| `design/` | godesign's invariants, and the plans its features were built to |
| `plugin/` | the Claude Code plugin; `.claude-plugin/marketplace.json` lists it |
| `scripts/plugin-version-check.sh` | fails when `plugin/` changed and its version did not go up |
