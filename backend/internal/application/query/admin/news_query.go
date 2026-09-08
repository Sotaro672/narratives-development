// backend/internal/application/query/admin/news_query.go

package query

import (
	"context"

	applicationport "narratives/internal/application/port"
	common "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

type newsLister interface {
	ListNews(
		ctx context.Context,
		filter newsdom.Filter,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[newsdom.News], error)
}

type NewsItem struct {
	News          newsdom.News
	CreatedByName string
}

type NewsQuery struct {
	newsLister     newsLister
	authUserReader applicationport.AuthUserReader
}

func NewNewsQuery(
	newsLister newsLister,
	authUserReader applicationport.AuthUserReader,
) *NewsQuery {
	return &NewsQuery{
		newsLister:     newsLister,
		authUserReader: authUserReader,
	}
}

func (q *NewsQuery) ListNews(
	ctx context.Context,
	filter newsdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[NewsItem], error) {
	if q == nil || q.newsLister == nil {
		return common.PageResult[NewsItem]{}, nil
	}

	result, err := q.newsLister.ListNews(ctx, filter, sort, page)
	if err != nil {
		return common.PageResult[NewsItem]{}, err
	}

	nameCache := make(map[string]string)
	items := make([]NewsItem, 0, len(result.Items))

	for _, entity := range result.Items {
		createdByName, ok := nameCache[entity.CreatedBy]
		if !ok {
			createdByName = q.ResolveCreatedByName(ctx, entity.CreatedBy)
			nameCache[entity.CreatedBy] = createdByName
		}

		items = append(items, NewsItem{
			News:          entity,
			CreatedByName: createdByName,
		})
	}

	return common.PageResult[NewsItem]{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}, nil
}

func (q *NewsQuery) ResolveCreatedByName(ctx context.Context, createdBy string) string {
	if q == nil || q.authUserReader == nil || createdBy == "" {
		return ""
	}

	name, err := q.authUserReader.GetDisplayNameByUID(ctx, createdBy)
	if err != nil {
		return ""
	}

	return name
}
