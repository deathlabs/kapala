package watcher

// Operation identifies how a Markdown file changed.
type Operation string

const (
	// OperationCreate identifies a newly created Markdown file.
	OperationCreate Operation = "create"
	// OperationRemove identifies a removed Markdown file.
	OperationRemove Operation = "remove"
	// OperationRename identifies a renamed Markdown file.
	OperationRename Operation = "rename"
	// OperationWrite identifies a modified Markdown file.
	OperationWrite Operation = "write"
)

// Event describes a change to a Markdown file.
type Event struct {
	Operation Operation `mapstructure:"operation" json:"operation" yaml:"operation"`
	Path      string    `mapstructure:"path"      json:"path"      yaml:"path"`
}

// Handler processes a debounced Markdown file change.
type Handler func(Event) error
