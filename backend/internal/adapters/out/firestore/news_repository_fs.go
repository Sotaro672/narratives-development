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
	defaultNewsPage           = 1
	defaultNewsPerPage        = 20
)

type NewsRepositoryFS struct {
	client     *firestore.Client
	collection string
}

type NewsReadRepositoryFS struct {
	client     *firestore.Client
	collection string
}

func NewNewsRepositoryFS(client *firestore.Client) *NewsRepositoryFS {
	return &NewsRepositoryFS{client: client, collection: defaultNewsCollection}
}

func NewNewsReadRepositoryFS(client *firestore.Client) *NewsReadRepositoryFS {
	return &NewsReadRepositoryFS{client: client, collection: defaultNewsReadCollection}
}

func (r *NewsRepositoryFS) WithCollection(name string) *NewsRepositoryFS {
	if r != nil && name != "" {
		r.collection = name
	}
	return r
}

func (r *NewsReadRepositoryFS) WithCollection(name string) *NewsReadRepositoryFS {
	if r != nil && name != "" {
		r.collection = name
	}
	return r
}

var _ newsdom.Repository = (*NewsRepositoryFS)(nil)
var _ newsdom.ReadRepository = (*NewsReadRepositoryFS)(nil)

type newsImageFirestoreDoc struct {
	FileURL    string `firestore:"fileUrl"`
	ObjectPath string `firestore:"objectPath"`
	FileName   string `firestore:"fileName"`
	MimeType   string `firestore:"mimeType"`
	FileSize   int64  `firestore:"fileSize"`
	Alt        string `firestore:"alt,omitempty"`
}

type newsFirestoreDoc struct {
	ID          string                 `firestore:"id"`
	Title       string                 `firestore:"title"`
	Body        string                 `firestore:"body"`
	Image       *newsImageFirestoreDoc `firestore:"image,omitempty"`
	PublishedAt time.Time              `firestore:"publishedAt"`
	CreatedAt   time.Time              `firestore:"createdAt"`
	CreatedBy   string                 `firestore:"createdBy"`
}

// RecipientType / RecipientID は Firestore に保存しない。
// document ID は NewsID + RecipientType + RecipientID から決定論的に生成し、
// 読者自身がその News を既読かどうか判定するためだけに利用する。
type newsReadFirestoreDoc struct {
	ID     string    `firestore:"id"`
	NewsID string    `firestore:"newsId"`
	ReadAt time.Time `firestore:"readAt"`
}

func (r *NewsRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *NewsRepositoryFS) doc(newsID newsdom.NewsID) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(newsID))
}

func (r *NewsReadRepositoryFS) collectionRef() *firestore.CollectionRef {
	return r.client.Collection(r.collection)
}

func (r *NewsReadRepositoryFS) doc(readID newsdom.NewsReadID) *firestore.DocumentRef {
	return r.collectionRef().Doc(string(readID))
}

func (r *NewsRepositoryFS) Create(ctx context.Context, entity newsdom.News) (newsdom.News, error) {
	if r == nil || r.client == nil {
		return newsdom.News{}, fmt.Errorf("news repository is not configured")
	}
	if err := entity.Validate(); err != nil {
		return newsdom.News{}, err
	}

	normalized, err := newsdom.New(entity.ID, entity.Title, entity.Body, entity.Image, entity.PublishedAt, entity.CreatedAt, entity.CreatedBy)
	if err != nil {
		return newsdom.News{}, err
	}

	if _, err = r.doc(normalized.ID).Create(ctx, encodeNews(normalized)); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return newsdom.News{}, newsdom.ErrConflict
		}
		return newsdom.News{}, err
	}
	return normalized, nil
}

func (r *NewsRepositoryFS) GetByID(ctx context.Context, newsID newsdom.NewsID) (newsdom.News, error) {
	if r == nil || r.client == nil {
		return newsdom.News{}, fmt.Errorf("news repository is not configured")
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

func (r *NewsRepositoryFS) List(ctx context.Context, filter newsdom.Filter, sortSpec common.Sort, page common.Page) (common.PageResult[newsdom.News], error) {
	if r == nil || r.client == nil {
		return common.PageResult[newsdom.News]{}, fmt.Errorf("news repository is not configured")
	}
	if newsHasTimeRange(filter.Updated) {
		return common.PageResult[newsdom.News]{}, fmt.Errorf("news: updated filter is not supported")
	}

	sortColumn := sortSpec.Column
	if sortColumn == "" {
		sortColumn = "publishedAt"
	}
	if _, ok := newsdom.AllowedSortColumns[sortColumn]; !ok {
		return common.PageResult[newsdom.News]{}, fmt.Errorf("news: invalid sort column: %s", sortColumn)
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
		if newsMatchesFilter(entity, filter) {
			items = append(items, entity)
		}
	}

	sortNewsItems(items, sortColumn, sortSpec.Order)
	return paginateNews(items, page), nil
}

func (r *NewsReadRepositoryFS) CreateIfAbsent(ctx context.Context, read newsdom.NewsRead) (newsdom.CreateNewsReadResult, error) {
	if r == nil || r.client == nil {
		return newsdom.CreateNewsReadResult{}, fmt.Errorf("news read repository is not configured")
	}
	if err := read.Validate(); err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}

	normalized, err := newsdom.NewRead(read.NewsID, read.RecipientType, read.RecipientID, read.ReadAt)
	if err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}

	ref := r.doc(normalized.ID)
	if _, err = ref.Create(ctx, encodeNewsRead(normalized)); err == nil {
		return newsdom.CreateNewsReadResult{Read: normalized, Created: true}, nil
	}
	if status.Code(err) != codes.AlreadyExists {
		return newsdom.CreateNewsReadResult{}, err
	}

	snapshot, err := ref.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newsdom.CreateNewsReadResult{}, newsdom.ErrReadNotFound
		}
		return newsdom.CreateNewsReadResult{}, err
	}

	existing, matched, err := decodeNewsReadForRecipient(snapshot, normalized.RecipientType, normalized.RecipientID)
	if err != nil {
		return newsdom.CreateNewsReadResult{}, err
	}
	if !matched {
		return newsdom.CreateNewsReadResult{}, newsdom.ErrReadNotFound
	}
	return newsdom.CreateNewsReadResult{Read: existing, Created: false}, nil
}

func (r *NewsReadRepositoryFS) Get(ctx context.Context, newsID newsdom.NewsID, recipientType newsdom.RecipientType, recipientID string) (newsdom.NewsRead, error) {
	if r == nil || r.client == nil {
		return newsdom.NewsRead{}, fmt.Errorf("news read repository is not configured")
	}

	readID, err := newsdom.BuildNewsReadID(newsID, recipientType, recipientID)
	if err != nil {
		return newsdom.NewsRead{}, err
	}

	snapshot, err := r.doc(readID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return newsdom.NewsRead{}, newsdom.ErrReadNotFound
		}
		return newsdom.NewsRead{}, err
	}

	read, matched, err := decodeNewsReadForRecipient(snapshot, recipientType, recipientID)
	if err != nil {
		return newsdom.NewsRead{}, err
	}
	if !matched || read.NewsID != newsID {
		return newsdom.NewsRead{}, newsdom.ErrReadNotFound
	}
	return read, nil
}

func (r *NewsReadRepositoryFS) List(ctx context.Context, filter newsdom.ReadFilter, sortSpec common.Sort, page common.Page) (common.PageResult[newsdom.NewsRead], error) {
	if r == nil || r.client == nil {
		return common.PageResult[newsdom.NewsRead]{}, fmt.Errorf("news read repository is not configured")
	}
	if filter.RecipientType == nil {
		return common.PageResult[newsdom.NewsRead]{}, newsdom.ErrInvalidRecipientType
	}
	if err := filter.RecipientType.Validate(); err != nil {
		return common.PageResult[newsdom.NewsRead]{}, err
	}
	if filter.RecipientID == "" {
		return common.PageResult[newsdom.NewsRead]{}, newsdom.ErrInvalidRecipientID
	}

	newsIDSet := make(map[newsdom.NewsID]struct{}, len(filter.NewsIDs))
	for _, newsID := range filter.NewsIDs {
		if err := newsID.Validate(); err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}
		newsIDSet[newsID] = struct{}{}
	}

	sortColumn := sortSpec.Column
	if sortColumn == "" {
		sortColumn = "readAt"
	}
	if _, ok := newsdom.AllowedReadSortColumns[sortColumn]; !ok {
		return common.PageResult[newsdom.NewsRead]{}, fmt.Errorf("news read: invalid sort column: %s", sortColumn)
	}
	if err := validateNewsSortOrder(sortSpec.Order); err != nil {
		return common.PageResult[newsdom.NewsRead]{}, err
	}

	iter := r.collectionRef().Documents(ctx)
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

		read, matched, err := decodeNewsReadForRecipient(snapshot, *filter.RecipientType, filter.RecipientID)
		if err != nil {
			return common.PageResult[newsdom.NewsRead]{}, err
		}
		if !matched || !newsReadMatchesFilter(read, filter, newsIDSet) {
			continue
		}
		items = append(items, read)
	}

	sortNewsReadItems(items, sortSpec.Order)
	return paginateNewsReads(items, page), nil
}

func encodeNews(entity newsdom.News) newsFirestoreDoc {
	var image *newsImageFirestoreDoc
	if entity.Image != nil {
		image = &newsImageFirestoreDoc{
			FileURL: entity.Image.FileURL, ObjectPath: entity.Image.ObjectPath, FileName: entity.Image.FileName,
			MimeType: entity.Image.MimeType, FileSize: entity.Image.FileSize, Alt: entity.Image.Alt,
		}
	}

	return newsFirestoreDoc{
		ID: string(entity.ID), Title: entity.Title, Body: entity.Body, Image: image,
		PublishedAt: entity.PublishedAt.UTC().Truncate(time.Microsecond),
		CreatedAt:   entity.CreatedAt.UTC().Truncate(time.Microsecond), CreatedBy: entity.CreatedBy,
	}
}

func encodeNewsRead(read newsdom.NewsRead) newsReadFirestoreDoc {
	return newsReadFirestoreDoc{
		ID: string(read.ID), NewsID: string(read.NewsID),
		ReadAt: read.ReadAt.UTC().Truncate(time.Microsecond),
	}
}

func decodeNews(snapshot *firestore.DocumentSnapshot) (newsdom.News, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return newsdom.News{}, fmt.Errorf("news: invalid firestore document")
	}

	var document newsFirestoreDoc
	if err := snapshot.DataTo(&document); err != nil {
		return newsdom.News{}, err
	}
	if document.ID != "" && document.ID != snapshot.Ref.ID {
		return newsdom.News{}, newsdom.ErrInvalidID
	}

	var image *newsdom.NewsImage
	if document.Image != nil {
		image = &newsdom.NewsImage{
			FileURL: document.Image.FileURL, ObjectPath: document.Image.ObjectPath, FileName: document.Image.FileName,
			MimeType: document.Image.MimeType, FileSize: document.Image.FileSize, Alt: document.Image.Alt,
		}
	}

	return newsdom.New(
		newsdom.NewsID(snapshot.Ref.ID), document.Title, document.Body, image,
		document.PublishedAt, document.CreatedAt, document.CreatedBy,
	)
}

func decodeNewsReadForRecipient(snapshot *firestore.DocumentSnapshot, recipientType newsdom.RecipientType, recipientID string) (newsdom.NewsRead, bool, error) {
	if snapshot == nil || snapshot.Ref == nil {
		return newsdom.NewsRead{}, false, fmt.Errorf("news read: invalid firestore document")
	}

	var document newsReadFirestoreDoc
	if err := snapshot.DataTo(&document); err != nil {
		return newsdom.NewsRead{}, false, err
	}
	if document.ID != "" && document.ID != snapshot.Ref.ID {
		return newsdom.NewsRead{}, false, newsdom.ErrInvalidReadID
	}

	newsID := newsdom.NewsID(document.NewsID)
	if err := newsID.Validate(); err != nil {
		return newsdom.NewsRead{}, false, err
	}

	expectedID, err := newsdom.BuildNewsReadID(newsID, recipientType, recipientID)
	if err != nil {
		return newsdom.NewsRead{}, false, err
	}
	if string(expectedID) != snapshot.Ref.ID {
		return newsdom.NewsRead{}, false, nil
	}

	read, err := newsdom.NewRead(newsID, recipientType, recipientID, document.ReadAt)
	if err != nil {
		return newsdom.NewsRead{}, false, err
	}
	if string(read.ID) != snapshot.Ref.ID {
		return newsdom.NewsRead{}, false, newsdom.ErrInvalidReadID
	}
	return read, true, nil
}

func newsMatchesFilter(entity newsdom.News, filter newsdom.Filter) bool {
	searchQuery := strings.ToLower(filter.SearchQuery)
	if searchQuery != "" {
		title := strings.ToLower(entity.Title)
		body := strings.ToLower(entity.Body)
		if !strings.Contains(title, searchQuery) && !strings.Contains(body, searchQuery) {
			return false
		}
	}
	if !newsTimeMatchesRange(entity.CreatedAt, filter.Created) {
		return false
	}
	if !newsTimeMatchesRange(entity.PublishedAt, filter.PublishedAt) {
		return false
	}
	return true
}

func newsReadMatchesFilter(read newsdom.NewsRead, filter newsdom.ReadFilter, newsIDSet map[newsdom.NewsID]struct{}) bool {
	if len(newsIDSet) > 0 {
		if _, ok := newsIDSet[read.NewsID]; !ok {
			return false
		}
	}
	return newsTimeMatchesRange(read.ReadAt, filter.ReadAt)
}

func newsTimeMatchesRange(value time.Time, timeRange common.TimeRange) bool {
	if timeRange.From != nil && value.Before(timeRange.From.UTC()) {
		return false
	}
	if timeRange.To != nil && value.After(timeRange.To.UTC()) {
		return false
	}
	return true
}

func newsHasTimeRange(timeRange common.TimeRange) bool {
	return timeRange.From != nil || timeRange.To != nil
}

func validateNewsSortOrder(order common.SortOrder) error {
	switch order {
	case "", common.SortAsc, common.SortDesc:
		return nil
	default:
		return fmt.Errorf("news: invalid sort order: %s", order)
	}
}

func sortNewsItems(items []newsdom.News, column string, order common.SortOrder) {
	descending := order != common.SortAsc
	sort.SliceStable(items, func(i int, j int) bool {
		left, right := items[i], items[j]
		comparison := 0

		switch column {
		case "title":
			comparison = strings.Compare(left.Title, right.Title)
		case "createdAt":
			comparison = compareNewsTime(left.CreatedAt, right.CreatedAt)
		default:
			comparison = compareNewsTime(left.PublishedAt, right.PublishedAt)
		}

		if comparison == 0 {
			comparison = strings.Compare(string(left.ID), string(right.ID))
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func sortNewsReadItems(items []newsdom.NewsRead, order common.SortOrder) {
	descending := order != common.SortAsc
	sort.SliceStable(items, func(i int, j int) bool {
		comparison := compareNewsTime(items[i].ReadAt, items[j].ReadAt)
		if comparison == 0 {
			comparison = strings.Compare(string(items[i].ID), string(items[j].ID))
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareNewsTime(left time.Time, right time.Time) int {
	switch {
	case left.Before(right):
		return -1
	case left.After(right):
		return 1
	default:
		return 0
	}
}

func normalizeNewsPage(page common.Page) (int, int) {
	pageNumber, perPage := page.Number, page.PerPage
	if pageNumber <= 0 {
		pageNumber = defaultNewsPage
	}
	if perPage <= 0 {
		perPage = defaultNewsPerPage
	}
	return pageNumber, perPage
}

func paginateNews(items []newsdom.News, page common.Page) common.PageResult[newsdom.News] {
	pageNumber, perPage := normalizeNewsPage(page)
	totalCount := len(items)
	totalPages := newsTotalPages(totalCount, perPage)

	start := (pageNumber - 1) * perPage
	if start >= totalCount {
		return common.PageResult[newsdom.News]{
			Items: []newsdom.News{}, TotalCount: totalCount, TotalPages: totalPages,
			Page: pageNumber, PerPage: perPage,
		}
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}
	pageItems := append([]newsdom.News(nil), items[start:end]...)
	return common.PageResult[newsdom.News]{
		Items: pageItems, TotalCount: totalCount, TotalPages: totalPages,
		Page: pageNumber, PerPage: perPage,
	}
}

func paginateNewsReads(items []newsdom.NewsRead, page common.Page) common.PageResult[newsdom.NewsRead] {
	pageNumber, perPage := normalizeNewsPage(page)
	totalCount := len(items)
	totalPages := newsTotalPages(totalCount, perPage)

	start := (pageNumber - 1) * perPage
	if start >= totalCount {
		return common.PageResult[newsdom.NewsRead]{
			Items: []newsdom.NewsRead{}, TotalCount: totalCount, TotalPages: totalPages,
			Page: pageNumber, PerPage: perPage,
		}
	}

	end := start + perPage
	if end > totalCount {
		end = totalCount
	}
	pageItems := append([]newsdom.NewsRead(nil), items[start:end]...)
	return common.PageResult[newsdom.NewsRead]{
		Items: pageItems, TotalCount: totalCount, TotalPages: totalPages,
		Page: pageNumber, PerPage: perPage,
	}
}

func newsTotalPages(totalCount int, perPage int) int {
	if perPage <= 0 {
		return 1
	}
	totalPages := int(math.Ceil(float64(totalCount) / float64(perPage)))
	if totalPages <= 0 {
		return 1
	}
	return totalPages
}
