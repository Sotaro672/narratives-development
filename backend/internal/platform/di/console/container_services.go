// backend/internal/platform/di/console/container_services.go
package console

import (
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memdom "narratives/internal/domain/member"
	pbdom "narratives/internal/domain/productBlueprint"
)

type services struct {
	companySvc *companydom.Service
	brandSvc   *branddom.Service
	memberSvc  *memdom.Service
	pbSvc      *pbdom.Service
}

func buildDomainServices(r *repos) *services {
	companySvc := companydom.NewService(r.companyRepo)
	brandSvc := branddom.NewService(r.brandRepo)
	memberSvc := memdom.NewService(r.memberRepo)

	// ✅ adapter を介さず repo を直で渡す
	// r.productBlueprintRepo が pbdom.Service の期待する interface を満たす前提
	pbSvc := pbdom.NewService(r.productBlueprintRepo)

	return &services{
		companySvc: companySvc,
		brandSvc:   brandSvc,
		memberSvc:  memberSvc,
		pbSvc:      pbSvc,
	}
}
