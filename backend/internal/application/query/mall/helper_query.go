// backend/internal/application/query/mall/helper_query.go
package mall

// ============================================================
// shared helpers (query)
// ============================================================

func cloneMeasurements(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}

	return out
}
