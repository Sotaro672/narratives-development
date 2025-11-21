package app

import (
	"fmt"
	"log"

	"cloud.google.com/go/firestore"

	"narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"

	fsadapter "narratives/internal/adapters/out/firestore"
	mailadapter "narratives/internal/adapters/out/mail"
)

// MemberRepo をどう作っているかはプロジェクトによりますが、
// ここでは memdom.Repository を引数でもらう形にしておくと安全です。
func NewInvitationCommandServiceWithSendGrid(
	fsClient *firestore.Client,
	memberRepo memdom.Repository,
) (usecase.InvitationCommandPort, error) {
	if fsClient == nil {
		return nil, fmt.Errorf("firestore client is nil")
	}

	// 1) 招待トークンリポジトリ（すでに実装済みの FS 版）
	tokenRepo := fsadapter.NewInvitationTokenRepositoryFS(fsClient)

	// 2) SendGrid バックエンドの InvitationMailer
	mailer := mailadapter.NewInvitationMailerWithSendGrid()

	// 3) InvitationCommandService を SendGrid 付きで生成
	cmdSvc := usecase.NewInvitationCommandService(
		tokenRepo,
		memberRepo,
		mailer,
	)

	log.Printf("[app] InvitationCommandService with SendGrid initialized")

	return cmdSvc, nil
}
