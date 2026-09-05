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

// firestoreString decodes a Firestore string that may be empty.
//
// Contract:
//   - missing field is invalid
//   - nil field is invalid
//   - only string is accepted
//   - empty string is accepted
//   - string content is returned unchanged
func firestoreString(values map[string]any, key string) (string, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", fmt.Errorf("firestore: missing %s", key)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("firestore: %s must be string, got %T", key, value)
	}

	return text, nil
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

// firestoreRequiredString decodes a required Firestore string.
//
// Contract:
//   - missing field is invalid
//   - nil field is invalid
//   - only string is accepted
//   - empty string is invalid
//   - string content is returned unchanged
func firestoreRequiredString(values map[string]any, key string) (string, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return "", fmt.Errorf("firestore: missing %s", key)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("firestore: %s must be string, got %T", key, value)
	}
	if text == "" {
		return "", fmt.Errorf("firestore: %s must not be empty", key)
	}

	return text, nil
}

// firestoreRequiredBool decodes a required Firestore boolean.
//
// Contract:
//   - missing field is invalid
//   - nil field is invalid
//   - only bool is accepted
func firestoreRequiredBool(values map[string]any, key string) (bool, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return false, fmt.Errorf("firestore: missing %s", key)
	}

	boolean, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("firestore: %s must be bool, got %T", key, value)
	}

	return boolean, nil
}

// firestoreRequiredInt64 decodes a required Firestore integer.
//
// Contract:
//   - missing field is invalid
//   - nil field is invalid
//   - only int64 is accepted
func firestoreRequiredInt64(values map[string]any, key string) (int64, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return 0, fmt.Errorf("firestore: missing %s", key)
	}

	number, ok := value.(int64)
	if !ok {
		return 0, fmt.Errorf("firestore: %s must be int64, got %T", key, value)
	}

	return number, nil
}

// firestoreOptionalInt64 decodes an optional Firestore integer.
//
// Contract:
//   - missing field returns nil
//   - nil field returns nil
//   - if present, only int64 is accepted
func firestoreOptionalInt64(values map[string]any, key string) (*int64, error) {
	value, ok := values[key]
	if !ok || value == nil {
		return nil, nil
	}

	number, ok := value.(int64)
	if !ok {
		return nil, fmt.Errorf("firestore: %s must be int64, got %T", key, value)
	}

	return &number, nil
}
