// backend/internal/platform/di/console/container_services.go
package console

import (
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
)

type services struct {
	companySvc *companydom.Service
	memberSvc  *memdom.Service
}

func buildDomainServices(r *repos) *services {
	companySvc := companydom.NewService(r.companyRepo)
	memberSvc := memdom.NewService(r.memberRepo)

	return &services{
		companySvc: companySvc,
		memberSvc:  memberSvc,
	}
}
