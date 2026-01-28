// backend/internal/adapters/in/http/console/handler/model/path.go
package model

import "strings"

// extractSingleID は prefix 配下の単一IDを抽出します。
// 例: path="/models/variations/123", prefix="/models/variations/" => "123"
func extractSingleID(path string, prefix string) (string, bool) {
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	id := strings.TrimPrefix(path, prefix)
	id = strings.Trim(id, "/")
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}
	// /models/variations/{id}/xxx のような余計なパスは弾く
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractBlueprintIDForList は以下のパスから productBlueprintID を抽出します。
// GET /models/by-blueprint/{productBlueprintID}/variations
func extractBlueprintIDForList(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/by-blueprint/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/models/by-blueprint/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "variations" {
		return "", false
	}
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	// productBlueprintID は単一セグメント想定
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractBlueprintIDForCreate は以下のパスから productBlueprintID を抽出します。
// POST /models/{productBlueprintID}/variations
func extractBlueprintIDForCreate(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/") {
		return "", false
	}
	rest := strings.TrimPrefix(path, "/models/")
	rest = strings.Trim(rest, "/")
	parts := strings.Split(rest, "/")
	if len(parts) != 2 || parts[1] != "variations" {
		return "", false
	}
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if strings.Contains(id, "/") {
		return "", false
	}
	return id, true
}

// extractModelID は以下のパスから model variation ID を抽出します。
// GET/PUT/DELETE /models/{id}
func extractModelID(path string) (string, bool) {
	if !strings.HasPrefix(path, "/models/") {
		return "", false
	}
	id := strings.TrimPrefix(path, "/models/")
	id = strings.Trim(id, "/")
	id = strings.TrimSpace(id)
	if id == "" {
		return "", false
	}

	// 念のため誤ルーティング防止
	if strings.HasPrefix(id, "variations/") || id == "variations" {
		return "", false
	}

	// /models/{id}/xxx は想定外
	if strings.Contains(id, "/") {
		return "", false
	}

	return id, true
}
