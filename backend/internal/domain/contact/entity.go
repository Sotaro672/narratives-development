// backend/internal/domain/contact/entity.go
package contact

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const contactAttachmentMarker = "--- 添付ファイル ---"

var (
	ErrInvalidName    = errors.New("invalid name")
	ErrInvalidEmail   = errors.New("invalid email")
	ErrInvalidMessage = errors.New("invalid message")
	ErrInvalidCompany = errors.New("invalid company")
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// Contact is a domain entity representing an inquiry from the contact form.
// It is expected to be stored as a document in Firestore collection: "contacts".
type Contact struct {
	ID string `json:"id"`

	Name               string   `json:"name"`
	Email              string   `json:"email"`
	Company            string   `json:"company"`
	Message            string   `json:"message"`
	AttachmentImageIDs []string `json:"attachmentImageIds"`

	IsRead    bool      `json:"isRead"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"createdAt"`
}

// NewContact creates a Contact entity with validation.
// - name, email, message are required.
// - company is optional.
// - message contains only the inquiry body.
// - attachmentImageIds contains references to attached images.
// - isRead is always initialized to false.
// - createdAt should be set by the application layer or defaults to the current UTC time.
func NewContact(
	name string,
	email string,
	company string,
	message string,
	attachmentImageIDs []string,
	source string,
	createdAt time.Time,
) (*Contact, error) {
	message = stripAttachmentMessage(message)

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
		Name:               name,
		Email:              email,
		Company:            company,
		Message:            message,
		AttachmentImageIDs: append([]string(nil), attachmentImageIDs...),
		IsRead:             false,
		Source:             source,
		CreatedAt:          createdAt,
	}, nil
}

func stripAttachmentMessage(message string) string {
	message = strings.TrimSpace(message)

	markerIndex := strings.Index(message, contactAttachmentMarker)
	if markerIndex < 0 {
		return message
	}

	return strings.TrimSpace(message[:markerIndex])
}

func validateName(name string) error {
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
