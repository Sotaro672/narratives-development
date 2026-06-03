// backend/internal/platform/di/console/container_services.go
package console

import (
	memdom "narratives/internal/domain/member"
)

type services struct {
	memberSvc *memdom.Service
}

func buildDomainServices(r *repos) *services {
	memberSvc := memdom.NewService(r.memberRepo)

	return &services{
		memberSvc: memberSvc,
	}
}
