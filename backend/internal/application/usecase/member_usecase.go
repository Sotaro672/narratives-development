// backend/internal/application/usecase/member_usecase.go
package usecase

import (
	"context"
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

// -----------------------------------------------------------------------------
// Usecase 本体
// -----------------------------------------------------------------------------

type MemberUsecase struct {
	repo memdom.Repository
	now  func() time.Time
}

func NewMemberUsecase(
	repo memdom.Repository,
	_ InvitationCommandPort,
) *MemberUsecase {
	return &MemberUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// -----------------------------------------------------------------------------
// Commands
// -----------------------------------------------------------------------------

type CreateMemberInput struct {
	// UID is Firebase Auth UID.
	//
	// 初回会社登録者など、Firebase Auth user がすでに確定している member 作成では必須。
	// 招待前 member 作成では空を許可し、招待承諾時に InvitationCompleteService 側で UID を設定する。
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
	// MemberID is Firestore member document ID.
	// Console の PATCH /members/{id} は Firebase UID ではなく member docId を使う。
	MemberID string

	FirstName      *string
	LastName       *string
	FirstNameKana  *string
	LastNameKana   *string
	Email          *string
	Permissions    *[]string
	AssignedBrands *[]string

	CompanyID string
	Status    *string
}

func (u *MemberUsecase) Update(ctx context.Context, in UpdateMemberInput) (MemberRecord, error) {
	memberID := in.MemberID
	if memberID == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	companyID := in.CompanyID
	if cid := CompanyIDFromContext(ctx); cid != "" {
		companyID = cid
	}
	if companyID == "" {
		return MemberRecord{}, memdom.ErrNotFound
	}

	current, err := u.repo.GetByID(ctx, memberID)
	if err != nil {
		return MemberRecord{}, err
	}

	if current.Member.CompanyID != companyID {
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

	now := u.now().UTC()
	patch.UpdatedAt = &now

	rec, err := u.repo.Update(ctx, memberID, patch)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

type GetCurrentMemberInput struct {
	FirebaseUID string
}

func (u *MemberUsecase) GetCurrentMember(ctx context.Context, in GetCurrentMemberInput) (MemberRecord, error) {
	firebaseUID := in.FirebaseUID
	if firebaseUID == "" {
		return MemberRecord{}, memdom.ErrInvalidUID
	}

	rec, err := u.repo.GetByUID(ctx, firebaseUID)
	if err != nil {
		return MemberRecord{}, err
	}

	return MemberRecord{
		DocID:  rec.DocID,
		Member: rec.Member,
	}, nil
}

func (u *MemberUsecase) Delete(ctx context.Context, memberID string) error {
	if memberID == "" {
		return memdom.ErrNotFound
	}

	return u.repo.Delete(ctx, memberID)
}
