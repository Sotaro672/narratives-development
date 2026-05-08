// backend/internal/platform/di/console/container_resolvers.go
package console

import (
	sharedfs "narratives/internal/adapters/out/firestore/shared"
	sharedquery "narratives/internal/application/query/shared"
	resolver "narratives/internal/application/resolver"
)

type resolvers struct {
	ownerResolveQuery *sharedquery.OwnerResolveQuery
	nameResolver      *resolver.NameResolver
}

func buildResolvers(c *clients, r *repos, s *services) *resolvers {
	// avatar owner は wallets/{avatarId}.walletAddress を逆引きする
	avatarAddrReader := sharedfs.NewAvatarWalletAddressReaderFS(c.fsClient, "wallets")
	brandAddrReader := sharedfs.NewBrandWalletAddressReaderFS(c.fsClient, "brands")

	ownerResolveQuery := sharedquery.NewOwnerResolveQuery(
		avatarAddrReader,
		brandAddrReader,
		r.avatarRepo, // avatarId -> avatarName
		s.brandSvc,   // brandId -> brandName
	)

	nameResolver := resolver.NewNameResolver(
		r.brandRepo,
		r.companyRepo,
		r.productBlueprintRepo,
		r.memberRepo,
		r.modelRepo,
		r.tokenBlueprintRepo,
	)

	return &resolvers{
		ownerResolveQuery: ownerResolveQuery,
		nameResolver:      nameResolver,
	}
}
