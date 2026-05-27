// backend/internal/platform/di/console/container_services.go
package console

import (
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
)

type services struct {
	companySvc *companydom.Service
	brandSvc   *branddom.Service
	memberSvc  *memdom.Service
}

func buildDomainServices(r *repos) *services {
	companySvc := companydom.NewService(r.companyRepo)
	brandSvc := branddom.NewService(r.brandRepo)
	memberSvc := memdom.NewService(r.memberRepo)

	return &services{
		companySvc: companySvc,
		brandSvc:   brandSvc,
		memberSvc:  memberSvc,
	}
}
