package idgen

type Generator interface {
	Generate() string
}

var generator Generator

// Register sets the global ID generator.
func Register(g Generator) {
	generator = g
}

// Generate generates a new ID.
func Generate() string {
	if generator == nil {
		panic("ID generator not registered")
	}
	return generator.Generate()
}
