package adapter

// Factory creates a new Adapter instance.
type Factory func() Adapter

var factories map[string]Factory

// Register adds an adapter factory under the given name.
func Register(name string, factory Factory) {
	if factories == nil {
		factories = make(map[string]Factory)
	}
	factories[name] = factory
}

// GetAdapter returns a new Adapter instance by name.
func GetAdapter(name string) (Adapter, bool) {
	if factories == nil {
		return nil, false
	}
	f, ok := factories[name]
	if !ok {
		return nil, false
	}
	return f(), true
}

// AllAdapters returns a new instance of every registered adapter.
func AllAdapters() []Adapter {
	if factories == nil {
		return nil
	}
	result := make([]Adapter, 0, len(factories))
	for _, f := range factories {
		result = append(result, f())
	}
	return result
}

// AllAdapterNames returns the names of all registered adapters.
func AllAdapterNames() []string {
	if factories == nil {
		return nil
	}
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	return names
}
