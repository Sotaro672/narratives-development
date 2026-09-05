// backend\internal\adapters\out\firestore\value_decoder.go
package firestore

import (
	"fmt"
	"time"
)

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
