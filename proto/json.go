package proto

import (
	jsonpb "github.com/gogo/protobuf/jsonpb"
	pb "github.com/gogo/protobuf/proto"
)

func marshalJSON(m pb.Message) ([]byte, error) {
	marshaler := jsonpb.Marshaler{}
	str, err := marshaler.MarshalToString(m)
	return []byte(str), err
}

func unmarshalJSON(data []byte, m pb.Message) error {
	return jsonpb.UnmarshalString(string(data), m)
}

func (m *Statement) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}

func (m *StatementBody) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}

func (m *Statement) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
}

func (m *StatementBody) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
}

func (m *Manifest) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}

func (m *ManifestBody) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}

func (m *Manifest) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
}

func (m *ManifestBody) UnmarshalJSON(data []byte) error {
	return unmarshalJSON(data, m)
}
