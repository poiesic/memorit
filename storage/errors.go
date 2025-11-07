package storage

import "errors"

var (
	// ErrNotFound indicates that the requested record was not found.
	ErrNotFound = errors.New("record not found")

	// ErrDuplicateKey indicates a duplicate key violation.
	ErrDuplicateKey = errors.New("duplicate key")

	// ErrTransactionFailed indicates that a transaction failed.
	ErrTransactionFailed = errors.New("transaction failed")

	// ErrStorageClosed indicates that the storage backend is closed.
	ErrStorageClosed = errors.New("storage is closed")

	// ErrInvalidQuery indicates invalid query parameters.
	ErrInvalidQuery = errors.New("invalid query parameters")

	// ErrSerializationFailed indicates a serialization/deserialization failure.
	ErrSerializationFailed = errors.New("serialization failed")

	// ErrTruncatedData indicates that data was truncated during reading.
	ErrTruncatedData = errors.New("truncated data")
)
