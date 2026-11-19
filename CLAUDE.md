# godesign

godesign uses itself: this repository's `.claude/settings.json` enables the
godesign plugin from `plugin/`, and in this repository its hooks build
`./cmd/godesign` from the working tree instead of running the one on PATH. A
change to a check is therefore held to the check as changed.

## Plan Go changes before writing them

Before adding or reshaping exported Go types, struct fields, interfaces, packages
or imports between packages, write the plan as an ontology in
`design/<change>.ofn`, check it with `godesign plan`, and then implement to it.
The plugin's `design-first` skill has the steps. A change that adds no exported
structure, such as most bug fixes, needs no plan.

`design/invariants.ofn` is different: it holds the rules the code keeps on every
change (layers, which packages may do I/O), and it is never retired.
`godesign plan` checks each plan against it. A new package has to be placed in
one of its groups.

The plugin's hooks hold this in place:

- `design-gate` refuses an edit that adds an exported type no plan names.
- `conform-check` runs `godesign conform` when a turn ends and reports drift
  between the plan and the code.
- `invariants-check` runs `godesign invariants` when a turn ends and reports a
  rule the code breaks.

If a plan turns out to be wrong, change it and tell the user what changed and why.
Do not edit a plan only to silence a finding, and do not weaken an invariant to
get past one: changing a rule is a decision for the user to hear about.

## Changing the plugin

Anything changed under `plugin/` needs a higher `version` in
`plugin/.claude-plugin/plugin.json`, or installed copies never update;
`scripts/plugin-version-check.sh` checks it, in CI too. The ontologies are read
and reasoned over by gowl, imported at a released version: a change gowl needs
goes to gowl first.
