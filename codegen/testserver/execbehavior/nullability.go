package execbehavior

// Nullability binds non-null GraphQL types to Go types that can hold nil.
type Nullability struct {
	Optional       *string
	Required       *string
	RequiredElems  []*string
	RequiredList   []*string
	Nested         [][]*string
	RequiredObject *Dog
	RequiredDogs   []*Dog
}
