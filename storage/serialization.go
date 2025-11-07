package storage

import (
	"github.com/poiesic/memorit/core"
)

// MarshalID serializes an ID to bytes.
func MarshalID(id core.ID) []byte {
	buf := make([]byte, core.IDMUS.Size(id))
	core.IDMUS.Marshal(id, buf)
	return buf
}

// UnmarshalID deserializes an ID from bytes.
func UnmarshalID(data []byte) (core.ID, error) {
	var id core.ID
	id, _, err := core.IDMUS.Unmarshal(data)
	return id, err
}

// MarshalChatRecord serializes a ChatRecord to bytes.
func MarshalChatRecord(record *core.ChatRecord) []byte {
	buf := make([]byte, core.ChatRecordMUS.Size(*record))
	core.ChatRecordMUS.Marshal(*record, buf)
	return buf
}

// UnmarshalChatRecord deserializes a ChatRecord from bytes.
func UnmarshalChatRecord(data []byte) (*core.ChatRecord, error) {
	record, _, err := core.ChatRecordMUS.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// MarshalConcept serializes a Concept to bytes.
func MarshalConcept(concept *core.Concept) []byte {
	buf := make([]byte, core.ConceptMUS.Size(*concept))
	core.ConceptMUS.Marshal(*concept, buf)
	return buf
}

// UnmarshalConcept deserializes a Concept from bytes.
func UnmarshalConcept(data []byte) (*core.Concept, error) {
	concept, _, err := core.ConceptMUS.Unmarshal(data)
	if err != nil {
		return nil, err
	}
	return &concept, nil
}
