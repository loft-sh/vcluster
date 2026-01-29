package provider

// Constructor is a function that takes a name and returns an Importable interface
type Constructor func(name string) Importable

var providerRegistry = map[string]Constructor{}

// Register registers a constructor for the given scheme
func Register(scheme string, constructor Constructor) {
	providerRegistry[scheme] = constructor
}

// Get returns the constructor for the given type
// The second return value is true if the type is registered, false otherwise
func Get(providerType string) Constructor {
	return providerRegistry[providerType]
}
