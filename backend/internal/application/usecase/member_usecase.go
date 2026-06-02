// backend/internal/application/usecase/member_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	memdom "narratives/internal/domain/member"
)

// -----------------------------------------------------------------------------
// DTOs
// -----------------------------------------------------------------------------

type MemberRecord struct {
	DocID  string
	Member memdom.Member
}

type MemberRecordPageResult struct {
	Items      []MemberRecord
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// -----------------------------------------------------------------------------
// Usecase 本体
// -----------------------------------------------------------------------------

type MemberUsecase struct {
	repo              memdom.Repository
	now               func() time.Time
	invitationCommand InvitationCommandPort
}

func NewMemberUsecase(repo memdom.Repository) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

func NewMemberUsecaseWithInvitationCommand(
	repo memdom.Repository,
	invitationCommand InvitationCommandPort,
) *MemberUsecase {
	return &MemberUsecase{
		repo:              repo,
		now:               time.Now,
		invitationCommand: invitationCommand,
	}
}

// -----------------------------------------------------------------------------
// Queries
// -----------------------------------------------------------------------------

// GetByID は member docId から MemberRecord を取得します。
// repository port の Get 系は GetByID のみに統一しています。
func (u *MemberUsecase) GetByID(ctx context.Context, id string) (MemberRecord, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	rec, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

// ListByCompanyID は companyID scope の member 一覧を取得します。
// repository port の List 系は ListByCompanyID のみに統一しています。
func (u *MemberUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
	f memdom.Filter,
	p memdom.Page,
) (MemberRecordPageResult, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		companyID = strings.TrimSpace(CompanyIDFromContext(ctx))
	}
	if companyID == "" {
		return MemberRecordPageResult{}, errors.New("member: companyID is empty")
	}

	res, err := u.repo.ListByCompanyID(ctx, companyID, f, p)
	if err != nil {
		return MemberRecordPageResult{}, err
	}

	items := make([]MemberRecord, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, MemberRecord{
			DocID:  item.DocID,
			Member: item.Member,
		})
	}

	return MemberRecordPageResult{
		Items:      items,
		TotalCount: res.TotalCount,
		TotalPages: res.TotalPages,
		Page:       res.Page,
		PerPage:    res.PerPage,
	}, nil
}

// -----------------------------------------------------------------------------
// Commands
// -----------------------------------------------------------------------------

type CreateMemberInput struct {
	// UID is Firebase Auth UID.
	//
	// 初回会社登録者など、Firebase Auth user がすでに確定している member 作成では必須。
	// 招待前 member 作成では空を許可し、招待承諾時に BindFirebaseUID で後付けする。
	UID string

	FirstName      string
	LastName       string
	FirstNameKana  string
	LastNameKana   string
	Email          string
	Permissions    []string
	AssignedBrands []string

	CompanyID string
	Status    string
	CreatedAt *time.Time
}

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (MemberRecord, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	cid := strings.TrimSpace(CompanyIDFromContext(ctx))
	companyID := strings.TrimSpace(in.CompanyID)
	if cid != "" {
		companyID = cid
	}

	m := memdom.Member{
		UID:            strings.TrimSpace(in.UID),
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		FirstNameKana:  in.FirstNameKana,
		LastNameKana:   in.LastNameKana,
		Email:          in.Email,
		Permissions:    dedupStrings(in.Permissions),
		AssignedBrands: dedupStrings(in.AssignedBrands),
		CompanyID:      companyID,
		Status:         in.Status,
		CreatedAt:      *createdAt,
		UpdatedAt:      nil,
	}

	rec, err := u.repo.Create(ctx, m)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

type UpdateMemberInput struct {
	ID string

	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string

	CompanyID *string
	Status    *string
}

func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (MemberRecord, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	patch := memdom.MemberPatch{
		FirstName:      in.FirstName,
		LastName:       in.LastName,
		FirstNameKana:  in.FirstNameKana,
		LastNameKana:   in.LastNameKana,
		Email:          in.Email,
		Permissions:    in.Permissions,
		AssignedBrands: in.AssignedBrands,
		Status:         in.Status,
	}

	if cid := strings.TrimSpace(CompanyIDFromContext(ctx)); cid != "" {
		patch.CompanyID = &cid
	} else if in.CompanyID != nil {
		companyID := strings.TrimSpace(*in.CompanyID)
		patch.CompanyID = &companyID
	}

	now := u.now().UTC()
	patch.UpdatedAt = &now

	rec, err := u.repo.Update(ctx, id, patch)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

type BindFirebaseUIDInput struct {
	DocID string
	UID   string
}

func (u *MemberUsecase) BindFirebaseUID(ctx context.Context, in BindFirebaseUIDInput) (MemberRecord, error) {
	docID := strings.TrimSpace(in.DocID)
	uid := strings.TrimSpace(in.UID)

	if docID == "" {
		return MemberRecord{}, fmt.Errorf("member docID is empty")
	}
	if uid == "" {
		return MemberRecord{}, fmt.Errorf("firebase uid is empty")
	}

	rec, err := u.repo.GetByID(ctx, docID)
	if err != nil {
		return MemberRecord{}, err
	}

	currentUID := strings.TrimSpace(rec.Member.UID)
	if currentUID != "" && currentUID != uid {
		return MemberRecord{}, memdom.ErrConflict
	}

	patch := memdom.MemberPatch{
		UID: &uid,
	}

	now := u.now().UTC()
	patch.UpdatedAt = &now

	updated, err := u.repo.Update(ctx, rec.DocID, patch)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  updated.DocID,
		Member: updated.Member,
	}, nil
}

func (u *MemberUsecase) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return memdom.ErrNotFound
	}

	return u.repo.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// Invitation (招待メール送信)
// -----------------------------------------------------------------------------

func (u *MemberUsecase) SendInvitation(ctx context.Context, memberID string) error {
	if u.invitationCommand == nil {
		return errors.New("invitation command is not configured")
	}

	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return fmt.Errorf("memberID is empty")
	}

	_, err := u.invitationCommand.CreateInvitationAndSend(ctx, memberID)
	if err != nil {
		return fmt.Errorf("send invitation failed: %w", err)
	}

	return nil
}
