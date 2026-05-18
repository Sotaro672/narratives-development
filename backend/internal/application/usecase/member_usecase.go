// backend/internal/application/usecase/member_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	common "narratives/internal/domain/common"
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

func (u *MemberUsecase) GetByID(ctx context.Context, id string) (memdom.Member, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *MemberUsecase) GetByDocID(ctx context.Context, docID string) (MemberRecord, error) {
	rec, err := u.repo.GetByDocID(ctx, docID)
	if err != nil {
		return MemberRecord{}, err
	}
	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

func (u *MemberUsecase) GetByFirebaseUID(ctx context.Context, firebaseUID string) (memdom.Member, error) {
	return u.repo.GetByFirebaseUID(ctx, firebaseUID)
}

func (u *MemberUsecase) GetRecordByFirebaseUID(ctx context.Context, firebaseUID string) (MemberRecord, error) {
	rec, err := u.repo.GetRecordByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return MemberRecord{}, err
	}
	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

func (u *MemberUsecase) GetByEmail(ctx context.Context, email string) (memdom.Member, error) {
	return u.repo.GetByEmail(ctx, email)
}

func (u *MemberUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, id)
}

func (u *MemberUsecase) List(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (common.PageResult[memdom.Member], error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}

	return u.repo.List(ctx, f, s, p)
}

func (u *MemberUsecase) ListWithDocID(
	ctx context.Context,
	f memdom.Filter,
	s common.Sort,
	p common.Page,
) (MemberRecordPageResult, error) {
	if cid := CompanyIDFromContext(ctx); cid != "" {
		f.CompanyID = cid
	}

	res, err := u.repo.ListWithDocID(ctx, f, s, p)
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

func (u *MemberUsecase) Create(ctx context.Context, in CreateMemberInput) (memdom.Member, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	cid := CompanyIDFromContext(ctx)
	companyID := in.CompanyID
	if cid != "" {
		companyID = cid
	}

	m := memdom.Member{
		UID:            in.UID,
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

	return u.repo.Create(ctx, m)
}

func (u *MemberUsecase) CreateWithDocID(ctx context.Context, in CreateMemberInput) (MemberRecord, error) {
	createdAt := in.CreatedAt
	if createdAt == nil || createdAt.IsZero() {
		t := u.now().UTC()
		createdAt = &t
	}

	cid := CompanyIDFromContext(ctx)
	companyID := in.CompanyID
	if cid != "" {
		companyID = cid
	}

	m := memdom.Member{
		UID:            in.UID,
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

	rec, err := u.repo.CreateWithDocID(ctx, m)
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

func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (memdom.Member, error) {
	id := in.ID
	if id == "" {
		return memdom.Member{}, memdom.ErrNotFound
	}

	currentRec, err := u.repo.GetByDocID(ctx, id)
	if err != nil {
		return memdom.Member{}, err
	}

	current := currentRec.Member

	if in.FirstName != nil {
		current.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		current.LastName = *in.LastName
	}
	if in.FirstNameKana != nil {
		current.FirstNameKana = *in.FirstNameKana
	}
	if in.LastNameKana != nil {
		current.LastNameKana = *in.LastNameKana
	}
	if in.Email != nil {
		current.Email = *in.Email
	}
	if in.Permissions != nil {
		current.Permissions = dedupStrings(*in.Permissions)
	}
	if in.AssignedBrands != nil {
		current.AssignedBrands = dedupStrings(*in.AssignedBrands)
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		current.CompanyID = cid
	} else if in.CompanyID != nil {
		current.CompanyID = *in.CompanyID
	}

	if in.Status != nil {
		current.Status = *in.Status
	}

	now := u.now().UTC()
	current.UpdatedAt = &now

	return u.repo.SaveByDocID(ctx, currentRec.DocID, current, nil)
}

type BindFirebaseUIDInput struct {
	DocID string
	UID   string
}

func (u *MemberUsecase) BindFirebaseUID(ctx context.Context, in BindFirebaseUIDInput) (MemberRecord, error) {
	docID := in.DocID
	uid := in.UID

	if docID == "" {
		return MemberRecord{}, fmt.Errorf("member docID is empty")
	}
	if uid == "" {
		return MemberRecord{}, fmt.Errorf("firebase uid is empty")
	}

	rec, err := u.repo.GetByDocID(ctx, docID)
	if err != nil {
		return MemberRecord{}, err
	}

	currentUID := rec.Member.UID
	if currentUID != "" && currentUID != uid {
		return MemberRecord{}, memdom.ErrConflict
	}

	if err := rec.Member.BindUID(uid, u.now().UTC()); err != nil {
		return MemberRecord{}, err
	}

	saved, err := u.repo.SaveByDocID(ctx, rec.DocID, rec.Member, nil)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: saved,
	}, nil
}

func (u *MemberUsecase) Save(ctx context.Context, m memdom.Member) (memdom.Member, error) {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = u.now().UTC()
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		m.CompanyID = cid
	}

	return u.repo.Save(ctx, m, nil)
}

func (u *MemberUsecase) SaveByDocID(ctx context.Context, docID string, m memdom.Member) (memdom.Member, error) {
	if docID == "" {
		return memdom.Member{}, fmt.Errorf("member docID is empty")
	}

	if m.CreatedAt.IsZero() {
		m.CreatedAt = u.now().UTC()
	}

	if cid := CompanyIDFromContext(ctx); cid != "" {
		m.CompanyID = cid
	}

	return u.repo.SaveByDocID(ctx, docID, m, nil)
}

func (u *MemberUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// Invitation (招待メール送信)
// -----------------------------------------------------------------------------

func (u *MemberUsecase) SendInvitation(ctx context.Context, memberID string) error {
	if u.invitationCommand == nil {
		return errors.New("invitation command is not configured")
	}

	if memberID == "" {
		return fmt.Errorf("memberID is empty")
	}

	_, err := u.invitationCommand.CreateInvitationAndSend(ctx, memberID)
	if err != nil {
		return fmt.Errorf("send invitation failed: %w", err)
	}

	return nil
}
