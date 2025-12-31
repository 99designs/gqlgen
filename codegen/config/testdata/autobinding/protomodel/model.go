package protomodel

// ProtoMessage simulates a protobuf editions message with getters and hasers
type ProtoMessage struct {
	name        *string
	description *string
	count       int32
}

// GetName is a protobuf-style getter
func (m *ProtoMessage) GetName() string {
	if m.name == nil {
		return ""
	}
	return *m.name
}

// HasName is a protobuf-style haser
func (m *ProtoMessage) HasName() bool {
	return m.name != nil
}

// GetDescription is a protobuf-style getter
func (m *ProtoMessage) GetDescription() string {
	if m.description == nil {
		return ""
	}
	return *m.description
}

// HasDescription is a protobuf-style haser
func (m *ProtoMessage) HasDescription() bool {
	return m.description != nil
}

// GetCount is a protobuf-style getter for a non-nullable field
func (m *ProtoMessage) GetCount() int32 {
	return m.count
}
