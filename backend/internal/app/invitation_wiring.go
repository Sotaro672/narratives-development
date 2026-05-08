package app

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"

	fsadapter "narratives/internal/adapters/out/firestore"
	mailadapter "narratives/internal/adapters/out/mail"
	"narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
)

// ============================================================
// InvitationTokenRepository のアプリケーション層向けアダプタ
// ============================================================

type invitationTokenRepoAdapter struct {
	fsRepo *fsadapter.InvitationTokenRepositoryFS
}

func (a *invitationTokenRepoAdapter) ResolveInvitationInfoByToken(
	ctx context.Context,
	token string,
) (memdom.InvitationInfo, error) {
	return a.fsRepo.ResolveInvitationInfoByToken(ctx, token)
}

func (a *invitationTokenRepoAdapter) CreateInvitationToken(
	ctx context.Context,
	info memdom.InvitationInfo,
) (string, error) {
	return a.fsRepo.CreateInvitationToken(ctx, info)
}

func (a *invitationTokenRepoAdapter) ConsumeInvitationToken(
	ctx context.Context,
	token string,
) error {
	return a.fsRepo.ConsumeInvitationToken(ctx, token)
}

// ============================================================
// NewInvitationCommandServiceWithResend
// ============================================================

func NewInvitationCommandServiceWithResend(
	fsClient *firestore.Client,
	memberRepo memdom.Repository,
	companyResolver mailadapter.CompanyNameResolver,
	brandResolver mailadapter.BrandNameResolver,
) (usecase.InvitationCommandPort, error) {
	if fsClient == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	fsTokenRepo := fsadapter.NewInvitationTokenRepositoryFS(fsClient)

	tokenRepo := &invitationTokenRepoAdapter{
		fsRepo: fsTokenRepo,
	}

	mailer := mailadapter.NewInvitationMailerWithResend(
		companyResolver,
		brandResolver,
	)

	cmdSvc := usecase.NewInvitationCommandService(
		tokenRepo,
		memberRepo,
		mailer,
	)

	log.Printf("[app] InvitationCommandService with Resend initialized")

	return cmdSvc, nil
}
