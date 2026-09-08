// backend/internal/adapters/out/firestore/news_repository_fs.go
package firestore

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	newsdom "narratives/internal/domain/news"
)

const (
	defaultNewsCollection     = "news"
	defaultNewsReadCollection = "newsReads"

	defaultNewsPage    = 1
	defaultNewsPerPage = 20
)

// ============================================================
// Repository
// ============================================================

type NewsRepositoryFS struct {
	client     *firestore.Client
	collection string
}

type NewsReadRepositoryFS struct {
	client     *firestore.Client
	collection string
}

func NewNewsRepositoryFS(
	client *firestore.Client,
) *NewsRepositoryFS {
	return &NewsRepositoryFS{
		client:     client,
		collection: defaultNewsCollection,
	}
}

func NewNewsReadRepositoryFS(
	client *firestore.Client,
) *NewsReadRepositoryFS {
	return &NewsReadRepositoryFS{
		client:     client,
		collection: defaultNewsReadCollection,
	}
}

func (r *NewsRepositoryFS) WithCollection(
	name string,
) *NewsRepositoryFS {
	if r != nil && strings.TrimSpace(name) != "" {
		r.collection = strings.TrimSpace(name)
	}

	return r
}

func (r *NewsReadRepositoryFS) WithCollection(
	name string,
) *NewsReadRepositoryFS {
	if r != nil && strings.TrimSpace(name) != "" {
		r.collection = strings.TrimSpace(name)
	}

	return r
}

// Compile-time checks.
var _ newsdom.Repository = (*NewsRepositoryFS)(nil)
var _ newsdom.ReadRepository = (*NewsReadRepositoryFS)(nil)

// ============================================================
// Firestore documents
// ============================================================

type newsFirestoreDoc struct {
	ID          string    `firestore:"id"`
	Title       string    `firestore:"title"`
	Body        string    `firestore:"body"`
	PublishedAt time.Time `firestore:"publishedAt"`
	CreatedAt   time.Time `firestore:"createdAt"`
	CreatedBy   string    `firestore:"createdBy"`
}

type newsReadFirestoreDoc struct {
	ID            string    `firestore:"id"`
	NewsID        string    `firestore:"newsId"`
	RecipientType string    `firestore:"recipientType"`
	RecipientID   string    `firestore:"recipientId"`
	ReadAt        time.Time `firestore:"readAt"`
}

// ============================================================
// References
// ============================================================

func (r *NewsRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *NewsRepositoryFS) doc(
	newsID newsdom.NewsID,
) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(newsID))
}

func (r *NewsReadRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *NewsReadRepositoryFS) doc(
	readID newsdom.NewsReadID,
) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(readID))
}

// ============================================================
// NewsRepositoryFS - Create
// ============================================================

func (r *NewsRepositoryFS) Create(
	ctx context.Context,
	entity newsdom.News,
) (newsdom.News, error) {
	if r == nil || r.client == nil {
		return newsdom.News{},
			fmt.Errorf("news repository is not configured")
	}

	if err := entity.Validate(); err != nil {
		return newsdom.News{}, err
	}

	normalized, err := newsdom.New(
		entity.ID,
		entity.Title,
		entity.Body,
		entity.PublishedAt,
		entity.CreatedAt,
		entity.CreatedBy,
	)
	if err != nil {
		return newsdom.News{}, err
	}

	_, err = r.doc(normalized.ID).Create(
		ctx,
		encodeNews(normalized),
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return newsdom.News{}, newsdom.ErrConflict
		}

		return newsdom.News{}, err
	}

	return normalized, nil
}

// ============================================================
// NewsRepositoryFS - GetByID
// ============================================================

func (r *NewsRepositoryFS) GetByID(
	ctx context.Context,
	newsID newsdom.NewsID,
) (newsdom.News, error) {
	if r == nil || r.client == nil {
		return newsdom.News{},
			fmt.Errorf("news repository is not configured")
	}

	if err := newsID.Validate(); err != nil {
		return newsdom.News{}, err
	}

	snapshot, err := r.doc(newsID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newsdom.News{}, newsdom.ErrNotFound
		}

		return newsdom.News{}, err
	}

	return decodeNews(snapshot)
}

// ============================================================
// NewsRepositoryFS - List
// ============================================================

func (r *NewsRepositoryFS) List(
	ctx context.Context,
	filter newsdom.Filter,
	sortSpec common.Sort,
	page common.Page,
) (common.PageResult[newsdom.News], error) {
	if r == nil || r.client == nil {
		return common.PageResult[newsdom.News]{},
			fmt.Errorf("news repository is not configured")
	}

	if newsHasTimeRange(filter.Updated) {
		return common.PageResult[newsdom.News]{},
			fmt.Errorf("news: updated filter is not supported")
	}

	sortColumn := strings.TrimSpace(sortSpec.Column)
	if sortColumn == "" {
		sortColumn = "publishedAt"
	}

	if _, ok := newsdom.AllowedSortColumns[sortColumn]; !ok {
		return common.PageResult[newsdom.News]{},
			fmt.Errorf(
				"news: invalid sort column: %s",
				sortColumn,
			)
	}

	if err := validateNewsSortOrder(sortSpec.Order); err != nil {
		return common.PageResult[newsdom.News]{}, err
	}

	iter := r.collectionRef().Documents(ctx)
	defer iter.Stop()

	items := make([]newsdom.News, 0)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[newsdom.News]{}, err
		}

		entity, err := decodeNews(snapshot)
		if err != nil {
			return common.PageResult[newsdom.News]{}, err
		}

		if !newsMatchesFilter(entity, filter) {
			continue
		}

		items = append(items, entity)
	}

	sortNewsItems(
		items,
		sortColumn,
		sortSpec.Order,
	)

	return paginateNews(
		items,
		page,
	), nil
}

// ============================================================
// NewsReadRepositoryFS - CreateIfAbsent
// ============================================================

func (r *NewsReadRepositoryFS) CreateIfAbsent(
	ctx context.Context,
	read newsdom.NewsRead,
) (newsdom.CreateNewsReadResult, error) {
	if r == nil || r.client == nil {
		return newsdom.CreateNewsReadResult{},
			fmt.Errorf("news read repository is not configured")
	}

	if err := read.Validate(); err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}

	normalized, err := newsdom.NewRead(
		read.NewsID,
		read.RecipientType,
		read.RecipientID,
		read.ReadAt,
	)
	if err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}

	ref := r.doc(normalized.ID)

	_, err = ref.Create(
		ctx,
		encodeNewsRead(normalized),
	)
	if err == nil {
		return newsdom.CreateNewsReadResult{
			Read:    normalized,
			Created: true,
		}, nil
	}

	if status.Code(err) != codes.AlreadyExists {
		return newsdom.CreateNewsReadResult{}, err
	}

	snapshot, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newsdom.CreateNewsReadResult{},
				newsdom.ErrReadNotFound
		}

		return newsdom.CreateNewsReadResult{}, err
	}

	existing, err := decodeNewsRead(snapshot)
	if err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}

	return newsdom.CreateNewsReadResult{
		Read:    existing,
		Created: false,
	}, nil
}

// ============================================================
// NewsReadRepositoryFS - Get
// ============================================================

func (r *NewsReadRepositoryFS) Get(
	ctx context.Context,
	newsID newsdom.NewsID,
	recipientType newsdom.RecipientType,
	recipientID string,
) (newsdom.NewsRead, error) {
	if r == nil || r.client == nil {
		return newsdom.NewsRead{},
			fmt.Errorf("news read repository is not configured")
	}

	recipientID = strings.TrimSpace(recipientID)

	readID, err := newsdom.BuildNewsReadID(
		newsID,
		recipientType,
		recipientID,
	)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	snapshot, err := r.doc(readID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newsdom.NewsRead{},
				newsdom.ErrReadNotFound
		}

		return newsdom.NewsRead{}, err
	}

	read, err := decodeNewsRead(snapshot)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	if read.NewsID != newsID ||
		read.RecipientType != recipientType ||
		read.RecipientID != recipientID {
		return newsdom.NewsRead{},
			newsdom.ErrReadNotFound
	}

	return read, nil
}

// ============================================================
// NewsReadRepositoryFS - List
// ============================================================

func (r *NewsReadRepositoryFS) List(
	ctx context.Context,
	filter newsdom.ReadFilter,
	sortSpec common.Sort,
	page common.Page,
) (common.PageResult[newsdom.NewsRead], error) {
	if r == nil || r.client == nil {
		return common.PageResult[newsdom.NewsRead]{},
			fmt.Errorf("news read repository is not configured")
	}

	if filter.RecipientType != nil {
		if err := filter.RecipientType.Validate(); err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}
	}

	recipientID := strings.TrimSpace(filter.RecipientID)

	newsIDSet := make(
		map[newsdom.NewsID]struct{},
		len(filter.NewsIDs),
	)

	for _, newsID := range filter.NewsIDs {
		if err := newsID.Validate(); err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}

		newsIDSet[newsID] = struct{}{}
	}

	sortColumn := strings.TrimSpace(sortSpec.Column)
	if sortColumn == "" {
		sortColumn = "readAt"
	}

	if _, ok := newsdom.AllowedReadSortColumns[sortColumn]; !ok {
		return common.PageResult[newsdom.NewsRead]{},
			fmt.Errorf(
				"news read: invalid sort column: %s",
				sortColumn,
			)
	}

	if err := validateNewsSortOrder(sortSpec.Order); err != nil {
		return common.PageResult[newsdom.NewsRead]{}, err
	}

	query := r.collectionRef().Query

	// 通常の Console / Mall 利用では RecipientID が必ず指定される。
	// RecipientType + RecipientID の複合Whereを避け、まず RecipientID の
	// 単一indexで絞り込み、RecipientType は domain entity 復元後に検証する。
	if recipientID != "" {
		query = query.Where(
			"recipientId",
			"==",
			recipientID,
		)
	} else if filter.RecipientType != nil {
		query = query.Where(
			"recipientType",
			"==",
			string(*filter.RecipientType),
		)
	}

	iter := query.Documents(ctx)
	defer iter.Stop()

	items := make([]newsdom.NewsRead, 0)

	for {
		snapshot, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}

		read, err := decodeNewsRead(snapshot)
		if err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}

		if !newsReadMatchesFilter(
			read,
			filter,
			recipientID,
			newsIDSet,
		) {
			continue
		}

		items = append(items, read)
	}

	sortNewsReadItems(
		items,
		sortSpec.Order,
	)

	return paginateNewsReads(
		items,
		page,
	), nil
}

// ============================================================
// Encode
// ============================================================

func encodeNews(
	entity newsdom.News,
) newsFirestoreDoc {
	return newsFirestoreDoc{
		ID:          string(entity.ID),
		Title:       entity.Title,
		Body:        entity.Body,
		PublishedAt: entity.PublishedAt.UTC().Truncate(time.Microsecond),
		CreatedAt:   entity.CreatedAt.UTC().Truncate(time.Microsecond),
		CreatedBy:   entity.CreatedBy,
	}
}

func encodeNewsRead(
	read newsdom.NewsRead,
) newsReadFirestoreDoc {
	return newsReadFirestoreDoc{
		ID:            string(read.ID),
		NewsID:        string(read.NewsID),
		RecipientType: string(read.RecipientType),
		RecipientID:   read.RecipientID,
		ReadAt:        read.ReadAt.UTC().Truncate(time.Microsecond),
	}
}

// ============================================================
// Decode
// ============================================================

func decodeNews(
	snapshot *firestore.DocumentSnapshot,
) (newsdom.News, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return newsdom.News{},
			fmt.Errorf("news: invalid firestore document")
	}

	var document newsFirestoreDoc
	if err := snapshot.DataTo(&document); err != nil {
		return newsdom.News{}, err
	}

	if document.ID != "" &&
		document.ID != snapshot.Ref.ID {
		return newsdom.News{}, newsdom.ErrInvalidID
	}

	return newsdom.New(
		newsdom.NewsID(snapshot.Ref.ID),
		document.Title,
		document.Body,
		document.PublishedAt,
		document.CreatedAt,
		document.CreatedBy,
	)
}

func decodeNewsRead(
	snapshot *firestore.DocumentSnapshot,
) (newsdom.NewsRead, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return newsdom.NewsRead{},
			fmt.Errorf("news read: invalid firestore document")
	}

	var document newsReadFirestoreDoc
	if err := snapshot.DataTo(&document); err != nil {
		return newsdom.NewsRead{}, err
	}

	if document.ID != "" &&
		document.ID != snapshot.Ref.ID {
		return newsdom.NewsRead{},
			newsdom.ErrInvalidReadID
	}

	read, err := newsdom.NewRead(
		newsdom.NewsID(document.NewsID),
		newsdom.RecipientType(document.RecipientType),
		document.RecipientID,
		document.ReadAt,
	)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	if string(read.ID) != snapshot.Ref.ID {
		return newsdom.NewsRead{},
			newsdom.ErrInvalidReadID
	}

	return read, nil
}

// ============================================================
// Filter
// ============================================================

func newsMatchesFilter(
	entity newsdom.News,
	filter newsdom.Filter,
) bool {
	searchQuery := strings.ToLower(
		strings.TrimSpace(filter.SearchQuery),
	)

	if searchQuery != "" {
		title := strings.ToLower(entity.Title)
		body := strings.ToLower(entity.Body)

		if !strings.Contains(title, searchQuery) &&
			!strings.Contains(body, searchQuery) {
			return false
		}
	}

	if !newsTimeMatchesRange(
		entity.CreatedAt,
		filter.Created,
	) {
		return false
	}

	if !newsTimeMatchesRange(
		entity.PublishedAt,
		filter.PublishedAt,
	) {
		return false
	}

	return true
}

func newsReadMatchesFilter(
	read newsdom.NewsRead,
	filter newsdom.ReadFilter,
	recipientID string,
	newsIDSet map[newsdom.NewsID]struct{},
) bool {
	if filter.RecipientType != nil &&
		read.RecipientType != *filter.RecipientType {
		return false
	}

	if recipientID != "" &&
		read.RecipientID != recipientID {
		return false
	}

	if len(newsIDSet) > 0 {
		if _, ok := newsIDSet[read.NewsID]; !ok {
			return false
		}
	}

	if !newsTimeMatchesRange(
		read.ReadAt,
		filter.ReadAt,
	) {
		return false
	}

	return true
}

func newsTimeMatchesRange(
	value time.Time,
	timeRange common.TimeRange,
) bool {
	if timeRange.From != nil &&
		value.Before(timeRange.From.UTC()) {
		return false
	}

	if timeRange.To != nil &&
		value.After(timeRange.To.UTC()) {
		return false
	}

	return true
}

func newsHasTimeRange(
	timeRange common.TimeRange,
) bool {
	return timeRange.From != nil ||
		timeRange.To != nil
}

// ============================================================
// Sort
// ============================================================

func validateNewsSortOrder(
	order common.SortOrder,
) error {
	switch order {
	case "", common.SortAsc, common.SortDesc:
		return nil
	default:
		return fmt.Errorf(
			"news: invalid sort order: %s",
			order,
		)
	}
}

func sortNewsItems(
	items []newsdom.News,
	column string,
	order common.SortOrder,
) {
	descending := order != common.SortAsc

	sort.SliceStable(
		items,
		func(i int, j int) bool {
			left := items[i]
			right := items[j]

			comparison := 0

			switch column {
			case "title":
				comparison = strings.Compare(
					left.Title,
					right.Title,
				)

			case "createdAt":
				comparison = compareNewsTime(
					left.CreatedAt,
					right.CreatedAt,
				)

			default:
				comparison = compareNewsTime(
					left.PublishedAt,
					right.PublishedAt,
				)
			}

			if comparison == 0 {
				comparison = strings.Compare(
					string(left.ID),
					string(right.ID),
				)
			}

			if descending {
				return comparison > 0
			}

			return comparison < 0
		},
	)
}

func sortNewsReadItems(
	items []newsdom.NewsRead,
	order common.SortOrder,
) {
	descending := order != common.SortAsc

	sort.SliceStable(
		items,
		func(i int, j int) bool {
			left := items[i]
			right := items[j]

			comparison := compareNewsTime(
				left.ReadAt,
				right.ReadAt,
			)

			if comparison == 0 {
				comparison = strings.Compare(
					string(left.ID),
					string(right.ID),
				)
			}

			if descending {
				return comparison > 0
			}

			return comparison < 0
		},
	)
}

func compareNewsTime(
	left time.Time,
	right time.Time,
) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

// ============================================================
// Pagination
// ============================================================

func normalizeNewsPage(
	page common.Page,
) (int, int) {
	pageNumber := page.Number
	perPage := page.PerPage

	if pageNumber <= 0 {
		pageNumber = defaultNewsPage
	}

	if perPage <= 0 {
		perPage = defaultNewsPerPage
	}

	return pageNumber, perPage
}

func paginateNews(
	items []newsdom.News,
	page common.Page,
) common.PageResult[newsdom.News] {
	pageNumber, perPage := normalizeNewsPage(page)

	totalCount := len(items)
	totalPages := newsTotalPages(
		totalCount,
		perPage,
	)

	start := (pageNumber - 1) * perPage
	if start >= totalCount {
		return common.PageResult[newsdom.News]{
			Items:      []newsdom.News{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNumber,
			PerPage:    perPage,
		}
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	pageItems := append(
		[]newsdom.News(nil),
		items[start:end]...,
	)

	return common.PageResult[newsdom.News]{
		Items:      pageItems,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}
}

func paginateNewsReads(
	items []newsdom.NewsRead,
	page common.Page,
) common.PageResult[newsdom.NewsRead] {
	pageNumber, perPage := normalizeNewsPage(page)

	totalCount := len(items)
	totalPages := newsTotalPages(
		totalCount,
		perPage,
	)

	start := (pageNumber - 1) * perPage
	if start >= totalCount {
		return common.PageResult[newsdom.NewsRead]{
			Items:      []newsdom.NewsRead{},
			TotalCount: totalCount,
			TotalPages: totalPages,
			Page:       pageNumber,
			PerPage:    perPage,
		}
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}

	pageItems := append(
		[]newsdom.NewsRead(nil),
		items[start:end]...,
	)

	return common.PageResult[newsdom.NewsRead]{
		Items:      pageItems,
		TotalCount: totalCount,
		TotalPages: totalPages,
		Page:       pageNumber,
		PerPage:    perPage,
	}
}

func newsTotalPages(
	totalCount int,
	perPage int,
) int {
	if perPage <= 0 {
		return 1
	}

	totalPages := int(
		math.Ceil(
			float64(totalCount) /
				float64(perPage),
		),
	)

	if totalPages <= 0 {
		return 1
	}

	return totalPages
}
