// backend/internal/adapters/out/firestore/contanct_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	common "narratives/internal/domain/common"
	contact "narratives/internal/domain/contact"
)

type ContactRepositoryFS struct {
	client *firestore.Client
}

func NewContactRepositoryFS(client *firestore.Client) *ContactRepositoryFS {
	return &ContactRepositoryFS{client: client}
}

func (r *ContactRepositoryFS) col() *firestore.CollectionRef {
	return r.client.Collection(contact.CollectionName)
}

func (r *ContactRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (contact.Contact, error) {
	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		return contact.Contact{}, err
	}

	var c contact.Contact
	if err := snap.DataTo(&c); err != nil {
		return contact.Contact{}, err
	}

	c.ID = snap.Ref.ID
	return c, nil
}

func (r *ContactRepositoryFS) Create(
	ctx context.Context,
	entity contact.Contact,
) (contact.Contact, error) {
	now := time.Now().UTC()
	attachmentImageIDs := append([]string{}, entity.AttachmentImageIDs...)

	doc := map[string]any{
		"name":               entity.Name,
		"email":              entity.Email,
		"company":            entity.Company,
		"message":            entity.Message,
		"attachmentImageIds": attachmentImageIDs,
		"isRead":             false,
		"source":             entity.Source,
		"createdAt":          firestore.ServerTimestamp,
		"updatedAt":          (*time.Time)(nil),
	}

	ref, _, err := r.col().Add(ctx, doc)
	if err != nil {
		return contact.Contact{}, err
	}

	created, err := r.GetByID(ctx, ref.ID)
	if err != nil {
		entity.ID = ref.ID
		entity.AttachmentImageIDs = attachmentImageIDs
		entity.IsRead = false
		entity.CreatedAt = now
		return entity, nil
	}

	return created, nil
}

func (r *ContactRepositoryFS) Update(
	ctx context.Context,
	id string,
	patch contact.Patch,
) (contact.Contact, error) {
	updates := make([]firestore.Update, 0, 10)

	if patch.Name != nil {
		updates = append(updates, firestore.Update{
			Path:  "name",
			Value: *patch.Name,
		})
	}
	if patch.Email != nil {
		updates = append(updates, firestore.Update{
			Path:  "email",
			Value: *patch.Email,
		})
	}
	if patch.Company != nil {
		updates = append(updates, firestore.Update{
			Path:  "company",
			Value: *patch.Company,
		})
	}
	if patch.Message != nil {
		updates = append(updates, firestore.Update{
			Path:  "message",
			Value: *patch.Message,
		})
	}
	if patch.IsRead != nil {
		updates = append(updates, firestore.Update{
			Path:  "isRead",
			Value: *patch.IsRead,
		})
	}
	if patch.Source != nil {
		updates = append(updates, firestore.Update{
			Path:  "source",
			Value: *patch.Source,
		})
	}

	if len(updates) == 0 {
		return r.GetByID(ctx, id)
	}

	updates = append(updates, firestore.Update{
		Path:  "updatedAt",
		Value: firestore.ServerTimestamp,
	})

	_, err := r.col().Doc(id).Update(ctx, updates)
	if err != nil {
		return contact.Contact{}, err
	}

	return r.GetByID(ctx, id)
}

func (r *ContactRepositoryFS) Delete(
	ctx context.Context,
	id string,
) error {
	_, err := r.col().Doc(id).Delete(ctx)
	return err
}

func (r *ContactRepositoryFS) List(
	ctx context.Context,
	filter contact.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[contact.Contact], error) {
	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 20
	}

	pageNumber := page.Number
	if pageNumber <= 0 {
		pageNumber = 1
	}

	offset := (pageNumber - 1) * perPage

	q, err := r.buildQuery(filter, sort)
	if err != nil {
		return common.PageResult[contact.Contact]{}, err
	}

	totalCount, err := r.count(ctx, q)
	if err != nil {
		return common.PageResult[contact.Contact]{}, err
	}

	it := q.Offset(offset).Limit(perPage).Documents(ctx)
	defer it.Stop()

	items := make([]contact.Contact, 0, perPage)

	for {
		snap, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[contact.Contact]{}, err
		}

		var c contact.Contact
		if err := snap.DataTo(&c); err != nil {
			return common.PageResult[contact.Contact]{}, err
		}

		c.ID = snap.Ref.ID
		items = append(items, c)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = int(
			math.Ceil(float64(totalCount) / float64(perPage)),
		)
	}

	return common.PageResult[contact.Contact]{
		Items:      items,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}, nil
}

func (r *ContactRepositoryFS) buildQuery(
	filter contact.Filter,
	sort common.Sort,
) (firestore.Query, error) {
	q := r.col().Query

	if filter.IsRead != nil {
		q = q.Where("isRead", "==", *filter.IsRead)
	}

	if filter.Created.From != nil {
		q = q.Where("createdAt", ">=", *filter.Created.From)
	}
	if filter.Created.To != nil {
		q = q.Where("createdAt", "<=", *filter.Created.To)
	}

	if filter.Updated.From != nil {
		q = q.Where("updatedAt", ">=", *filter.Updated.From)
	}
	if filter.Updated.To != nil {
		q = q.Where("updatedAt", "<=", *filter.Updated.To)
	}

	if filter.SearchQuery != "" {
		return firestore.Query{}, fmt.Errorf(
			"SearchQuery is not supported in firestore contact list yet",
		)
	}

	col := sort.Column
	if col == "" {
		col = "createdAt"
	}

	order := sort.Order
	if order == "" {
		order = common.SortDesc
	}

	switch col {
	case "createdAt", "updatedAt", "isRead", "email", "name":
	default:
		return firestore.Query{}, fmt.Errorf(
			"invalid sort column: %s",
			col,
		)
	}

	dir := firestore.Desc
	if order == common.SortAsc {
		dir = firestore.Asc
	}

	q = q.OrderBy(col, dir)
	return q, nil
}

func (r *ContactRepositoryFS) count(
	ctx context.Context,
	q firestore.Query,
) (int, error) {
	it := q.Documents(ctx)
	defer it.Stop()

	n := 0

	for {
		_, err := it.Next()
		if err == iterator.Done {
			return n, nil
		}
		if err != nil {
			return 0, err
		}

		n++
	}
}
