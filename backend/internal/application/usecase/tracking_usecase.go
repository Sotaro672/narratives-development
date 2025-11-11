// backend\internal\application\usecase\tracking_usecase.go
package usecase

import (
	"context"
	"strings"

	trackingdom "narratives/internal/domain/tracking"
)

// TrackingRepo defines the minimal persistence port needed by TrackingUsecase.
type TrackingRepo interface {
	GetByID(ctx context.Context, id string) (trackingdom.Tracking, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, v trackingdom.Tracking) (trackingdom.Tracking, error)
	Save(ctx context.Context, v trackingdom.Tracking) (trackingdom.Tracking, error)
	Delete(ctx context.Context, id string) error
}

// TrackingUsecase orchestrates tracking operations.
type TrackingUsecase struct {
	repo TrackingRepo
}

func NewTrackingUsecase(repo TrackingRepo) *TrackingUsecase {
	return &TrackingUsecase{repo: repo}
}

// Queries

func (u *TrackingUsecase) GetByID(ctx context.Context, id string) (trackingdom.Tracking, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *TrackingUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *TrackingUsecase) Create(ctx context.Context, v trackingdom.Tracking) (trackingdom.Tracking, error) {
	return u.repo.Create(ctx, v)
}

func (u *TrackingUsecase) Save(ctx context.Context, v trackingdom.Tracking) (trackingdom.Tracking, error) {
	return u.repo.Save(ctx, v)
}

func (u *TrackingUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
