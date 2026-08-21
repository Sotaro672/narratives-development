// backend/internal/platform/di/console/container_services.go
package console

import (
	memdom "narratives/internal/domain/member"
	transportationdom "narratives/internal/domain/transportation"
)

type services struct {
	memberSvc         *memdom.Service
	transportationSvc *transportationdom.Service
}

func buildDomainServices(r *repos) *services {
	memberSvc := memdom.NewService(
		r.memberRepo,
	)

	transportationSvc := transportationdom.NewService(
		r.transportationRepo,
	)

	return &services{
		memberSvc:         memberSvc,
		transportationSvc: transportationSvc,
	}
}
