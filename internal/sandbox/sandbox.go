// Package sandbox holds constants shared across the tool.
package sandbox

// Label is attached to every container the tool creates. ps and rm refuse to
// touch any container that does not carry this label, so the tool can never
// list or remove an unrelated container.
const Label = "container-sandbox"

// LabelName carries the sandbox config name, for informational purposes only.
const LabelName = "container-sandbox.name"
