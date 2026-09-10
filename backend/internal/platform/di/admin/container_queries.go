// backend/internal/platform/di/admin/container_queries.go
package admin

import (
	"context"
	"errors"

	firebaseadp "narratives/internal/adapters/out/firebase"
	adminquery "narratives/internal/application/query/admin"
	solanainfra "narratives/internal/infra/solana"
	shared "narratives/internal/platform/di/shared"
)

type queries struct {
	newsQuery                           *adminquery.NewsQuery
	reportNameQuery                     *adminquery.ReportNameQuery
	contractDetailQuery                 *adminquery.ContractDetailQuery
	contractListQuery                   *adminquery.ContractListQuery
	contractTokenBlueprintQuery         *adminquery.ContractTokenBlueprintQuery
	contractProductBlueprintQuery       *adminquery.ContractProductBlueprintQuery
	contractTokenBlueprintReviewQuery   *adminquery.ContractTokenBlueprintReviewQuery
	contractProductBlueprintReviewQuery *adminquery.ContractProductBlueprintReviewQuery
	gasBalanceQuery                     *adminquery.GasBalanceQuery
	mintListQuery                       *adminquery.MintListQuery
}

func buildQueries(
	ctx context.Context,
	infra *shared.Infra,
	r *repos,
	res *resolvers,
	u *usecases,
) (*queries, error) {
	if infra == nil {
		return nil, errors.New("di.admin: shared infra is nil")
	}
	if infra.FirebaseAuth == nil {
		return nil, errors.New("di.admin: firebase auth is nil")
	}
	if r == nil {
		return nil, errors.New("di.admin: repositories are nil")
	}
	if res == nil || res.nameResolver == nil {
		return nil, errors.New("di.admin: resolvers are nil")
	}
	if u == nil {
		return nil, errors.New("di.admin: usecases are nil")
	}
	if u.newsUsecase == nil {
		return nil, errors.New("di.admin: news usecase is nil")
	}
	if u.productBlueprintReviewUsecase == nil {
		return nil, errors.New("di.admin: product blueprint review usecase is nil")
	}
	if u.tokenBlueprintReviewUsecase == nil {
		return nil, errors.New("di.admin: token blueprint review usecase is nil")
	}

	authUserReader := firebaseadp.NewAuthUserReader(infra.FirebaseAuth)
	if authUserReader == nil {
		return nil, errors.New("di.admin: auth user reader is nil")
	}

	newsQuery := adminquery.NewNewsQuery(
		u.newsUsecase,
		authUserReader,
	)
	if newsQuery == nil {
		return nil, errors.New("di.admin: news query is nil")
	}

	reportNameQuery := adminquery.NewReportNameQuery(
		r.avatarRepo,
		r.brandRepo,
		r.companyRepo,
		r.listRepo,
		r.memberRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
	)

	contractDetailQuery := adminquery.NewContractDetailQuery(
		r.companyRepo,
		r.brandRepo,
		r.memberRepo,
		r.productBlueprintRepo,
		r.productBlueprintReviewRepo,
		r.tokenBlueprintRepo,
		r.inventoryRepo,
		r.listRepo,
		r.reportRepo,
	)

	contractListQuery := adminquery.NewContractListQuery(
		r.companyRepo,
		r.brandRepo,
		r.memberRepo,
		r.productBlueprintRepo,
		r.tokenBlueprintRepo,
		r.inventoryRepo,
		r.listRepo,
		r.listImageRepo,
		r.modelRepo,
		r.orderConsoleLister,
		r.reportRepo,
	)

	contractProductBlueprintQuery := adminquery.NewContractProductBlueprintQuery(
		r.companyRepo,
		r.brandRepo,
		r.memberRepo,
		r.productBlueprintRepo,
		r.modelRepo,
		r.productBlueprintReviewRepo,
		r.reportRepo,
	)
	if contractProductBlueprintQuery == nil {
		return nil, errors.New("di.admin: contract product blueprint query is nil")
	}

	contractProductBlueprintReviewQuery := adminquery.NewContractProductBlueprintReviewQuery(
		r.companyRepo,
		r.productBlueprintRepo,
		u.productBlueprintReviewUsecase,
	)
	if contractProductBlueprintReviewQuery == nil {
		return nil, errors.New("di.admin: contract product blueprint review query is nil")
	}

	contractTokenBlueprintQuery := adminquery.NewContractTokenBlueprintQuery(
		r.companyRepo,
		r.brandRepo,
		r.memberRepo,
		r.tokenBlueprintRepo,
		u.tokenBlueprintReviewUsecase,
		r.reportRepo,
	)

	contractTokenBlueprintReviewQuery := adminquery.NewContractTokenBlueprintReviewQuery(
		r.companyRepo,
		r.tokenBlueprintRepo,
		u.tokenBlueprintReviewUsecase,
		r.reportRepo,
	)
	if contractTokenBlueprintReviewQuery == nil {
		return nil, errors.New("di.admin: contract token blueprint review query is nil")
	}

	mintListQuery := adminquery.NewMintListQuery(
		r.mintRepo,
		res.nameResolver,
		r.memberRepo,
	)
	if mintListQuery == nil {
		return nil, errors.New("di.admin: mint list query is nil")
	}

	solanaClient, err := solanainfra.NewMintClient(ctx)
	if err != nil {
		return nil, err
	}

	gasBalanceQuery := adminquery.NewGasBalanceQuery(
		func(ctx context.Context) (*adminquery.GasBalanceResult, error) {
			result, err := solanaClient.GetReserveBalance(ctx)
			if err != nil {
				return nil, err
			}
			if result == nil {
				return nil, errors.New("di.admin: reserve balance result is nil")
			}

			return &adminquery.GasBalanceResult{
				Cluster:         result.Cluster,
				Address:         result.Address,
				BalanceLamports: result.BalanceLamports,
				BalanceSOL:      result.BalanceSOL,
			}, nil
		},
	)
	if gasBalanceQuery == nil {
		return nil, errors.New("di.admin: gas balance query is nil")
	}

	return &queries{
		newsQuery:                           newsQuery,
		reportNameQuery:                     reportNameQuery,
		contractDetailQuery:                 contractDetailQuery,
		contractListQuery:                   contractListQuery,
		contractTokenBlueprintQuery:         contractTokenBlueprintQuery,
		contractProductBlueprintQuery:       contractProductBlueprintQuery,
		contractTokenBlueprintReviewQuery:   contractTokenBlueprintReviewQuery,
		contractProductBlueprintReviewQuery: contractProductBlueprintReviewQuery,
		gasBalanceQuery:                     gasBalanceQuery,
		mintListQuery:                       mintListQuery,
	}, nil
}

func (q *queries) applyToContainer(c *Container) {
	if q == nil || c == nil {
		return
	}

	c.newsQuery = q.newsQuery
	c.reportNameQuery = q.reportNameQuery
	c.contractDetailQuery = q.contractDetailQuery
	c.contractListQuery = q.contractListQuery
	c.contractTokenBlueprintQuery = q.contractTokenBlueprintQuery
	c.contractProductBlueprintQuery = q.contractProductBlueprintQuery
	c.contractTokenBlueprintReviewQuery = q.contractTokenBlueprintReviewQuery
	c.contractProductBlueprintReviewQuery = q.contractProductBlueprintReviewQuery
	c.gasBalanceQuery = q.gasBalanceQuery
	c.mintListQuery = q.mintListQuery
}
