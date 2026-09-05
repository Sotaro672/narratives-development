// backend/internal/adapters/out/firestore/value_decoder.go
package firestore

import (
	"fmt"
	"time"
)

// firestoreRequiredTime decodes a required Firestore timestamp.
//
// Contract:
//   - missing field is invalid
//   - nil field is invalid
//   - only time.Time is accepted
//   - zero time is invalid
//   - returned time is always normalized to UTC
//
// Firestore timestamp decoding must not silently fall back to time.Time{}.
func firestoreRequiredTime(values map[string]any, key string) (time.Time, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return time.Time{}, fmt.Errorf("firestore: missing %s", key)
	}

	timestamp, ok := value.(time.Time)
	if !ok {
		return time.Time{}, fmt.Errorf("firestore: %s must be time.Time, got %T", key, value)
	}
	if timestamp.IsZero() {
		return time.Time{}, fmt.Errorf("firestore: %s must not be zero", key)
	}

	return timestamp.UTC(), nil
}

// firestoreOptionalTime decodes an optional Firestore timestamp.
//
// Contract:
//   - missing field returns nil
//   - nil field returns nil
//   - if present, only time.Time is accepted
//   - if present, zero time is invalid
//   - returned time is always normalized to UTC
//
// A present but malformed timestamp must not be treated as an absent value.
func firestoreOptionalTime(values map[string]any, key string) (*time.Time, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return nil, nil
	}

	timestamp, ok := value.(time.Time)
	if !ok {
		return nil, fmt.Errorf("firestore: %s must be time.Time, got %T", key, value)
	}
	if timestamp.IsZero() {
		return nil, fmt.Errorf("firestore: %s must not be zero", key)
	}

	timestamp = timestamp.UTC()
	return &timestamp, nil
}

// firestoreOptionalString decodes an optional Firestore string.
//
// Contract:
//   - missing field returns nil
//   - nil field returns nil
//   - if present, only string is accepted
//   - if present, empty string is invalid
//   - string content is returned unchanged
//
// A present but malformed or empty string must not be treated as an absent value.
func firestoreOptionalString(values map[string]any, key string) (*string, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return nil, nil
	}

	text, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("firestore: %s must be string, got %T", key, value)
	}
	if text == "" {
		return nil, fmt.Errorf("firestore: %s must not be empty", key)
	}

	return &text, nil
}
