// Copyright 2025 Poiesic Systems
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.


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
