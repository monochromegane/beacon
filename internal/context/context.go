package context

// Context represents contextual information that can be serialized to JSON.
type Context interface {
	Type() string
	ToJSON() ([]byte, error)
}

// Provider is an interface for obtaining context information.
type Provider interface {
	GetContext() (Context, error)
}
