//go:generate rm -f generated.go models-gen.go
//go:generate go run ../../../main.go -v

package notweaks

// NewConfig creates a configuration
func NewConfig() Config {
	return Config{
		Resolvers: &Resolver{},
	}
}
