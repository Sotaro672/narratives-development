package campaign

import (
    "context"

    campaignImage "narratives/internal/domain/campaignImage"
    campaignPerformance "narratives/internal/domain/campaignPerformance"
)

// Narrow repo contracts used by this service.
// Use your existing repository interfaces if they already match these shapes.
type campaignRepo interface {
    GetByID(ctx context.Context, id string) (Campaign, error)
}
type performanceRepo interface {
    GetByID(ctx context.Context, id string) (campaignPerformance.CampaignPerformance, error)
}
type imageRepo interface {
    GetByID(ctx context.Context, id string) (campaignImage.CampaignImage, error)
}

// ViewModel passed to the screen.
type CampaignDetail struct {
    Campaign    Campaign                                   `json:"campaign"`
    Performance *campaignPerformance.CampaignPerformance   `json:"performance,omitempty"`
    Image       *campaignImage.CampaignImage               `json:"image,omitempty"`
}

type Service struct {
    cRepo campaignRepo
    pRepo performanceRepo
    iRepo imageRepo
}

func NewService(c campaignRepo, p performanceRepo, i imageRepo) *Service {
    return &Service{
        cRepo: c,
        pRepo: p,
        iRepo: i,
    }
}

// GetDetail loads Campaign with optional Performance and Image by their IDs.
// If PerformanceID or ImageID is set but the record is not found, it is ignored.
func (s *Service) GetDetail(ctx context.Context, campaignID string) (CampaignDetail, error) {
    c, err := s.cRepo.GetByID(ctx, campaignID)
    if err != nil {
        return CampaignDetail{}, err
    }

    var perf *campaignPerformance.CampaignPerformance
    if c.PerformanceID != nil && *c.PerformanceID != "" && s.pRepo != nil {
        if p, err := s.pRepo.GetByID(ctx, *c.PerformanceID); err == nil {
            perf = &p
        }
    }

    var img *campaignImage.CampaignImage
    if c.ImageID != nil && *c.ImageID != "" && s.iRepo != nil {
        if im, err := s.iRepo.GetByID(ctx, *c.ImageID); err == nil {
            img = &im
        }
    }

    return CampaignDetail{
        Campaign:    c,
        Performance: perf,
        Image:       img,
    }, nil
}