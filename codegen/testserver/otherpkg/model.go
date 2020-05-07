package otherpkg

type Scalar string

type Struct struct {
	Name string
}

type Map map[string]string

func (m Map) Get(key string) string {
	return m[key]
}

type Slice []string

func (s Slice) Get(idx int) string {
	return s[idx]
}
