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

func (m *Statement) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}

func (m *StatementBody) MarshalJSON() ([]byte, error) {
	return marshalJSON(m)
}
