// backend/internal/platform/di/admin/container_resolvers.go
package admin

import (
	appresolver "narratives/internal/application/resolver"
)

type resolvers struct {
	nameResolver *appresolver.NameResolver
}

func buildResolvers(r *repos) *resolvers {
	if r == nil {
		return nil
	}

	nameResolver := appresolver.NewNameResolver(
		r.brandRepo,
		r.companyRepo,
		r.productBlueprintRepo,
		r.memberRepo,
		nil,
		r.modelRepo,
		r.tokenBlueprintRepo,
	)

	return &resolvers{
		nameResolver: nameResolver,
	}
}
