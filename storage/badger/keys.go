package badger

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/poiesic/memorit/core"
)

// Key prefixes for different data types
const (
	chatRecordPrefix        = "charec"
	chatRecordDatePrefix    = "charecd"
	chatRecordConceptPrefix = "charecc"
	chatRecordIDSeq         = "charecseq"
	conceptRecordPrefix     = "conrec"
	conceptTypeNamePrefix   = "contyna"
	conceptIDSeq            = "conrecseq"
)

// makeChatRecordKey generates a key for a chat record by ID.
func makeChatRecordKey(id core.ID) []byte {
	return []byte(fmt.Sprintf("%s:%d", chatRecordPrefix, id))
}

// makeChatDateKey generates a composite key for the date index.
// Format: prefix:timestamp:id
func makeChatDateKey(timestamp time.Time, id core.ID) []byte {
	prefix := chatRecordDatePrefix + ":"
	prefixBytes := []byte(prefix)
	prefixSize := len(prefixBytes)
	totalSize := prefixSize + 16 // 8 bytes for timestamp + 8 bytes for ID
	buf := make([]byte, totalSize)
	offset := copy(buf, prefixBytes)
	// Write in BigEndian order so lexicographic sort works correctly
	binary.BigEndian.PutUint64(buf[offset:], uint64(timestamp.UnixMicro()))
	offset += 8
	binary.BigEndian.PutUint64(buf[offset:], uint64(id))
	return buf
}

// makePartialChatDateKey generates a partial key for date range queries.
// Format: prefix:timestamp
func makePartialChatDateKey(timestamp time.Time) []byte {
	prefix := chatRecordDatePrefix + ":"
	prefixBytes := []byte(prefix)
	prefixSize := len(prefixBytes)
	totalSize := prefixSize + 8 // 8 bytes for timestamp
	buf := make([]byte, totalSize)
	offset := copy(buf, prefixBytes)
	// Write in BigEndian order so lexicographic sort works correctly
	binary.BigEndian.PutUint64(buf[offset:], uint64(timestamp.UnixMicro()))
	return buf
}

// makeChatConceptKey generates a composite key for the concept index.
// Format: prefix:conceptID:recordID
func makeChatConceptKey(conceptID, recordID core.ID) []byte {
	prefix := chatRecordConceptPrefix + ":"
	prefixBytes := []byte(prefix)
	prefixSize := len(prefixBytes)
	totalSize := prefixSize + 16 // 8 bytes for conceptID + 8 bytes for recordID
	buf := make([]byte, totalSize)
	offset := copy(buf, prefixBytes)
	// Write in BigEndian order so lexicographic sort works correctly
	binary.BigEndian.PutUint64(buf[offset:], uint64(conceptID))
	offset += 8
	binary.BigEndian.PutUint64(buf[offset:], uint64(recordID))
	return buf
}

// makePartialChatConceptKey generates a partial key for concept queries.
// Format: prefix:conceptID
func makePartialChatConceptKey(conceptID core.ID) []byte {
	prefix := chatRecordConceptPrefix + ":"
	prefixBytes := []byte(prefix)
	prefixSize := len(prefixBytes)
	totalSize := prefixSize + 8 // 8 bytes for conceptID
	buf := make([]byte, totalSize)
	offset := copy(buf, prefixBytes)
	// Write in BigEndian order so lexicographic sort works correctly
	binary.BigEndian.PutUint64(buf[offset:], uint64(conceptID))
	return buf
}

// makeConceptKey generates a key for a concept by ID.
func makeConceptKey(id core.ID) []byte {
	return []byte(fmt.Sprintf("%s:%d", conceptRecordPrefix, id))
}

// makeConceptTupleKey generates a composite key for concept lookup by (type, name).
// Format: prefix:type:name
func makeConceptTupleKey(name, conceptType string) []byte {
	prefix := conceptTypeNamePrefix + ":"
	totalSize := len(prefix) + len(conceptType) + len(name)
	buf := make([]byte, totalSize)
	offset := copy(buf, []byte(prefix))
	offset += copy(buf[offset:], []byte(conceptType))
	copy(buf[offset:], []byte(name))
	return buf
}

// makeCheckpointKey generates a key for processor checkpoints.
func makeCheckpointKey(processorType string) []byte {
	return []byte(fmt.Sprintf("%s:chkpt", processorType))
}
