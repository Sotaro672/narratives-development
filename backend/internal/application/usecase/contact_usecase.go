// backend/internal/application/usecase/contact_usecase.go
package usecase

import (
	"context"
	"fmt"
	"log"
	"time"

	common "narratives/internal/domain/common"
	contact "narratives/internal/domain/contact"
)

type ContactReceiptMailer interface {
	SendContactReceipt(
		ctx context.Context,
		name string,
		email string,
		company string,
		message string,
		source string,
	) error
}

type ContactAdminNotifier interface {
	SendContactAdminNotification(
		ctx context.Context,
		id string,
		name string,
		email string,
		company string,
		message string,
		source string,
		createdAt time.Time,
	) error
}

// ContactUsecase provides application-layer operations for contact inquiries.
type ContactUsecase struct {
	repo          contact.Repository
	receiptMailer ContactReceiptMailer
	adminNotifier ContactAdminNotifier
}

// NewContactUsecase creates a new ContactUsecase.
func NewContactUsecase(
	repo contact.Repository,
	receiptMailer ContactReceiptMailer,
	adminNotifier ContactAdminNotifier,
) *ContactUsecase {
	return &ContactUsecase{
		repo:          repo,
		receiptMailer: receiptMailer,
		adminNotifier: adminNotifier,
	}
}

// CreateInput is the input DTO for creating a contact inquiry.
type CreateInput struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Message string `json:"message"`
	Source  string `json:"source"`
}

// Create sends a receipt mail first, persists the inquiry if the receipt mail succeeds,
// and then sends an admin notification mail after persistence.
func (u *ContactUsecase) Create(ctx context.Context, in CreateInput) (contact.Contact, error) {
	entity, err := contact.NewContact(
		in.Name,
		in.Email,
		in.Company,
		in.Message,
		in.Source,
		time.Now().UTC(),
	)
	if err != nil {
		return contact.Contact{}, err
	}

	if u.receiptMailer == nil {
		return contact.Contact{}, fmt.Errorf("receipt mailer is not configured")
	}

	if err := u.receiptMailer.SendContactReceipt(
		ctx,
		entity.Name,
		entity.Email,
		entity.Company,
		entity.Message,
		entity.Source,
	); err != nil {
		log.Printf("[contact] receipt mail send failed: email=%s err=%v", entity.Email, err)
		return contact.Contact{}, err
	}

	log.Printf("[contact] receipt mail sent: email=%s", entity.Email)

	created, err := u.repo.Create(ctx, *entity)
	if err != nil {
		log.Printf("[contact] firestore create failed after receipt mail sent: email=%s err=%v", entity.Email, err)
		return contact.Contact{}, err
	}

	log.Printf("[contact] firestore create succeeded: id=%s email=%s", created.ID, created.Email)

	if u.adminNotifier != nil {
		if err := u.adminNotifier.SendContactAdminNotification(
			ctx,
			created.ID,
			created.Name,
			created.Email,
			created.Company,
			created.Message,
			created.Source,
			created.CreatedAt,
		); err != nil {
			log.Printf("[contact] admin notification mail send failed: id=%s email=%s err=%v", created.ID, created.Email, err)
		} else {
			log.Printf("[contact] admin notification mail sent: id=%s email=%s", created.ID, created.Email)
		}
	}

	return created, nil
}

// GetByID fetches a contact inquiry by id.
func (u *ContactUsecase) GetByID(ctx context.Context, id string) (contact.Contact, error) {
	return u.repo.GetByID(ctx, id)
}

// List lists contact inquiries with filter/sort/page.
func (u *ContactUsecase) List(
	ctx context.Context,
	filter contact.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[contact.Contact], error) {
	return u.repo.List(ctx, filter, sort, page)
}

// Update updates a contact inquiry by id.
func (u *ContactUsecase) Update(ctx context.Context, id string, patch contact.Patch) (contact.Contact, error) {
	return u.repo.Update(ctx, id, patch)
}

// Delete deletes a contact inquiry by id.
func (u *ContactUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
