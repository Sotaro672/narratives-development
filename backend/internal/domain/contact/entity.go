// backend/internal/domain/contact/entity.go
package contact

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	ErrInvalidName    = errors.New("invalid name")
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidMessage = errors.New("invalid message")
	ErrInvalidCompany = errors.New("invalid company")
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Status represents the lifecycle status of a contact inquiry.
type Status string

const (
	StatusNew Status = "new"
)

// Contact is a domain entity representing an inquiry from the contact form.
// It is expected to be stored as a document in Firestore collection: "contacts".
type Contact struct {
	// ID is the Firestore document id (or generated id) if you choose to set it after persistence.
	ID string `json:"id"`

	// Input fields from frontend
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"` // optional; allow empty string
	Message string `json:"message"`

	// Metadata
	Status    Status    `json:"status"`
	Source    string    `json:"source"` // e.g. "web-introduction"
	CreatedAt time.Time `json:"createdAt"`
}

// NewContact creates a Contact entity with validation.
// - name, email, message are required.
// - company is optional.
// - createdAt should be set by the application layer (or replaced by server timestamp at persistence time).
func NewContact(
	name string,
	email string,
	company string,
	message string,
	source string,
	createdAt time.Time,
) (*Contact, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if err := validateCompany(company); err != nil {
		return nil, err
	}
	if err := validateMessage(message); err != nil {
		return nil, err
	}
	if source == "" {
		source = "web-introduction"
	}
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	return &Contact{
		Name:      name,
		Email:     email,
		Company:   company,
		Message:   message,
		Status:    StatusNew,
		Source:    source,
		CreatedAt: createdAt,
	}, nil
}

func validateName(name string) error {
	// Basic checks: non-empty and reasonable length
	if name == "" {
		return ErrInvalidName
	}
	if l := len([]rune(name)); l < 1 || l > 100 {
		return fmt.Errorf("%w: length must be 1..100", ErrInvalidName)
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	if l := len(email); l < 3 || l > 254 {
		return fmt.Errorf("%w: length must be 3..254", ErrInvalidEmail)
	}
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func validateCompany(company string) error {
	// Optional field; if provided, check length
	if company == "" {
		return nil
	}
	if l := len([]rune(company)); l > 200 {
		return fmt.Errorf("%w: length must be <= 200", ErrInvalidCompany)
	}
	return nil
}

func validateMessage(message string) error {
	if message == "" {
		return ErrInvalidMessage
	}
	if l := len([]rune(message)); l < 1 || l > 4000 {
		return fmt.Errorf("%w: length must be 1..4000", ErrInvalidMessage)
	}
	return nil
}
