// backend\internal\domain\inventory\entity.go
package inventory

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ========================================
// Types (mirror TS)
// ========================================

type InventoryModel struct {
	ModelNumber string
	Quantity    int
}

type InventoryStatus string

const (
	StatusInspecting InventoryStatus = "inspecting"
	StatusInspected  InventoryStatus = "inspected"
	StatusListed     InventoryStatus = "listed"
	StatusDiscarded  InventoryStatus = "discarded"
	StatusDeleted    InventoryStatus = "deleted"
)

// Centralized error definitions (moved from entity.go)
var (
	ErrInvalidID             = errors.New("inventory: invalid id")
	ErrInvalidConnectedToken = errors.New("inventory: invalid connectedToken")
	ErrInvalidModels         = errors.New("inventory: invalid models")
	ErrInvalidModelNumber    = errors.New("inventory: invalid modelNumber")
	ErrInvalidQuantity       = errors.New("inventory: invalid quantity")
	ErrInvalidLocation       = errors.New("inventory: invalid location")
	ErrInvalidStatus         = errors.New("inventory: invalid status")
	ErrInvalidCreatedAt      = errors.New("inventory: invalid createdAt")
	ErrInvalidUpdatedAt      = errors.New("inventory: invalid updatedAt")
	ErrInvalidCreatedBy      = errors.New("inventory: invalid createdBy")
	ErrInvalidUpdatedBy      = errors.New("inventory: invalid updatedBy")
)

// Validation (moved from entity.go)
func (i Inventory) validate() error {
	if i.ID == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(i.Location) == "" {
		return ErrInvalidLocation
	}
	if err := validateModels(i.Models); err != nil {
		return err
	}
	if !IsValidStatus(i.Status) {
		return ErrInvalidStatus
	}
	if strings.TrimSpace(i.CreatedBy) == "" {
		return ErrInvalidCreatedBy
	}
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if strings.TrimSpace(i.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	if i.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	// ConnectedToken is optional; when present must be non-empty
	if i.ConnectedToken != nil && strings.TrimSpace(*i.ConnectedToken) == "" {
		return ErrInvalidConnectedToken
	}
	return nil
}

// moved from entity.go
func validateModels(models []InventoryModel) error {
	seen := make(map[string]struct{}, len(models))
	for _, m := range models {
		if strings.TrimSpace(m.ModelNumber) == "" {
			return ErrInvalidModelNumber
		}
		if m.Quantity < 0 {
			return ErrInvalidQuantity
		}
		if _, ok := seen[m.ModelNumber]; ok {
			// aggregateModels should have deduped; treat duplicates as error here.
			return ErrInvalidModels
		}
		seen[m.ModelNumber] = struct{}{}
	}
	return nil
}

// ========================================
// Behavior (moved from entity.go)
// ========================================

func (i *Inventory) ConnectToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidConnectedToken
	}
	i.ConnectedToken = &token
	return nil
}

func (i *Inventory) DisconnectToken() {
	i.ConnectedToken = nil
}

func (i *Inventory) UpdateLocation(location string) error {
	location = strings.TrimSpace(location)
	if location == "" {
		return ErrInvalidLocation
	}
	i.Location = location
	return nil
}

func (i *Inventory) UpdateStatus(status InventoryStatus) error {
	if !IsValidStatus(status) {
		return ErrInvalidStatus
	}
	i.Status = status
	return nil
}

func (i *Inventory) TouchUpdated(now time.Time, by string) error {
	by = strings.TrimSpace(by)
	if by == "" {
		return ErrInvalidUpdatedBy
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	now = now.UTC()
	if !i.CreatedAt.IsZero() && now.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	i.UpdatedAt = now
	i.UpdatedBy = by
	return nil
}

func (i *Inventory) SetModelQuantity(modelNumber string, qty int) error {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" {
		return ErrInvalidModelNumber
	}
	if qty < 0 {
		return ErrInvalidQuantity
	}
	found := false
	for idx := range i.Models {
		if i.Models[idx].ModelNumber == modelNumber {
			i.Models[idx].Quantity = qty
			found = true
			break
		}
	}
	if !found {
		i.Models = append(i.Models, InventoryModel{ModelNumber: modelNumber, Quantity: qty})
	}
	i.compact()
	return nil
}

func (i *Inventory) IncrementModel(modelNumber string, delta int) error {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" {
		return ErrInvalidModelNumber
	}
	found := false
	for idx := range i.Models {
		if i.Models[idx].ModelNumber == modelNumber {
			newQ := i.Models[idx].Quantity + delta
			if newQ < 0 {
				return ErrInvalidQuantity
			}
			i.Models[idx].Quantity = newQ
			found = true
			break
		}
	}
	if !found {
		if delta < 0 {
			return ErrInvalidQuantity
		}
		i.Models = append(i.Models, InventoryModel{ModelNumber: modelNumber, Quantity: delta})
	}
	i.compact()
	return nil
}

func (i *Inventory) RemoveModel(modelNumber string) {
	modelNumber = strings.TrimSpace(modelNumber)
	if modelNumber == "" || len(i.Models) == 0 {
		return
	}
	out := i.Models[:0]
	for _, m := range i.Models {
		if m.ModelNumber != modelNumber {
			out = append(out, m)
		}
	}
	i.Models = out
}

func (i *Inventory) ReplaceModels(models []InventoryModel) error {
	m := aggregateModels(models)
	if err := validateModels(m); err != nil {
		return err
	}
	i.Models = m
	return nil
}
func IsValidStatus(s InventoryStatus) bool {
	switch s {
	case StatusInspecting, StatusInspected, StatusListed, StatusDiscarded, StatusDeleted:
		return true
	default:
		return false
	}
}

type Inventory struct {
	ID             string
	ConnectedToken *string // null allowed
	Models         []InventoryModel
	Location       string
	Status         InventoryStatus
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdatedBy      string
}

// ========================================
// Constructors
// ========================================

func New(
	id string,
	models []InventoryModel,
	connectedToken *string,
	location string,
	status InventoryStatus,
	createdBy string,
	createdAt time.Time,
	updatedBy string,
	updatedAt time.Time,
) (Inventory, error) {
	inv := Inventory{
		ID:             strings.TrimSpace(id),
		ConnectedToken: normalizeStringPtr(connectedToken),
		Models:         aggregateModels(models),
		Location:       strings.TrimSpace(location),
		Status:         status,
		CreatedBy:      strings.TrimSpace(createdBy),
		CreatedAt:      createdAt.UTC(),
		UpdatedBy:      strings.TrimSpace(updatedBy),
		UpdatedAt:      updatedAt.UTC(),
	}
	if err := inv.validate(); err != nil {
		return Inventory{}, err
	}
	return inv, nil
}

func NewFromStrings(
	id string,
	models []InventoryModel,
	connectedToken *string,
	location string,
	status InventoryStatus,
	createdBy string,
	createdAt string,
	updatedBy string,
	updatedAt string,
) (Inventory, error) {
	ca, err := parseTime(createdAt)
	if err != nil {
		return Inventory{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
	}
	ua, err := parseTime(updatedAt)
	if err != nil {
		return Inventory{}, fmt.Errorf("%w: %v", ErrInvalidUpdatedAt, err)
	}
	return New(id, models, connectedToken, location, status, createdBy, ca, updatedBy, ua)
}

func (i *Inventory) compact() {
	// Remove negative (shouldn't exist) and merge duplicates
	i.Models = aggregateModels(i.Models)
}

func aggregateModels(models []InventoryModel) []InventoryModel {
	type agg struct{ qty int }
	buf := make(map[string]agg, len(models))
	order := make([]string, 0, len(models))
	for _, m := range models {
		num := strings.TrimSpace(m.ModelNumber)
		if num == "" {
			continue
		}
		if m.Quantity < 0 {
			continue
		}
		if _, ok := buf[num]; !ok {
			order = append(order, num)
		}
		a := buf[num]
		a.qty += m.Quantity
		if a.qty < 0 {
			a.qty = 0
		}
		buf[num] = a
	}
	out := make([]InventoryModel, 0, len(buf))
	for _, num := range order {
		out = append(out, InventoryModel{ModelNumber: num, Quantity: buf[num].qty})
	}
	return out
}

func normalizeStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, ErrInvalidCreatedAt
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}
