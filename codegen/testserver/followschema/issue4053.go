package followschema

type Issue4053Input1 struct {
	Input2 Issue4053Input2
}

type Issue4053Input2 struct {
	Hello            string
	HelloWithDefault string
}
