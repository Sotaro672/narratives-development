// backend/internal/platform/di/console/container_resolvers.go
package console

import (
	sharedfs "narratives/internal/adapters/out/firestore/shared"
	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"
)

type resolvers struct {
	ownerResolveQ *sharedquery.OwnerResolveQuery
	nameResolver  *resolver.NameResolver
}

func buildResolvers(c *clients, r *repos, s *services) *resolvers {
	avatarAddrReader := sharedfs.NewAvatarWalletAddressReaderFS(c.fsClient, "avatars")
	brandAddrReader := sharedfs.NewBrandWalletAddressReaderFS(c.fsClient, "brands")
	avatarNameReader := avatarNameReaderAdapter{repo: r.avatarRepo}

	ownerResolveQ := sharedquery.NewOwnerResolveQuery(
		avatarAddrReader,
		brandAddrReader,
		avatarNameReader,
		s.brandSvc,
	)

	tokenBlueprintNameRepo := &tokenBlueprintNameRepoAdapter{repo: r.tokenBlueprintRepo}
	nameResolver := resolver.NewNameResolver(
		r.brandRepo,
		r.companyRepo,
		r.productBlueprintRepo,
		r.memberRepo,
		r.modelRepo,
		tokenBlueprintNameRepo,
	)

	return &resolvers{
		ownerResolveQ: ownerResolveQ,
		nameResolver:  nameResolver,
	}
}
