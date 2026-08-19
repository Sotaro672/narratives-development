package productBlueprintCategory

import (
	"errors"
	"fmt"
)

// ======================================
// Domain errors
// ======================================

var (
	ErrNotFound     = errors.New("productBlueprintCategory: not found")
	ErrConflict     = errors.New("productBlueprintCategory: conflict")
	ErrInvalid      = errors.New("productBlueprintCategory: invalid")
	ErrUnauthorized = errors.New("productBlueprintCategory: unauthorized")
	ErrForbidden    = errors.New("productBlueprintCategory: forbidden")
	ErrInternal     = errors.New("productBlueprintCategory: internal")

	ErrRepositoryInvalidInput = errors.New(
		"productBlueprintCategory: repository invalid input",
	)
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

func IsInvalid(err error) bool {
	return errors.Is(err, ErrInvalid)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}

func WrapInvalid(err error, msg string) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrInvalid,
			msg,
		)
	}

	return fmt.Errorf(
		"%w: %s: %v",
		ErrInvalid,
		msg,
		err,
	)
}

func WrapConflict(err error, msg string) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrConflict,
			msg,
		)
	}

	return fmt.Errorf(
		"%w: %s: %v",
		ErrConflict,
		msg,
		err,
	)
}

func WrapNotFound(err error, msg string) error {
	if err == nil {
		return fmt.Errorf(
			"%w: %s",
			ErrNotFound,
			msg,
		)
	}

	return fmt.Errorf(
		"%w: %s: %v",
		ErrNotFound,
		msg,
		err,
	)
}

// ======================================
// Entity
// ======================================

type ProductBlueprintCategory struct {
	Path []string
}

type Snapshot struct {
	Path []string `json:"path"`
}

// ======================================
// Constructors
// ======================================

func Reconstruct(
	path []string,
) (ProductBlueprintCategory, error) {
	return ProductBlueprintCategory{
		Path: append(
			[]string(nil),
			path...,
		),
	}, nil
}

func (c ProductBlueprintCategory) ToSnapshot() Snapshot {
	return Snapshot{
		Path: append(
			[]string(nil),
			c.Path...,
		),
	}
}

// ======================================
// Validation errors
// ======================================

var (
	ErrInvalidPath = errors.New(
		"productBlueprintCategory: invalid path",
	)
)
