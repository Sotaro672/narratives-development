// backend/internal/adapters/out/firestore/wallet_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	dbcommon "narratives/internal/adapters/out/firestore/common"
	wdom "narratives/internal/domain/wallet"
)

// =====================================================
// Firestore Wallet Repository
// Implements WalletUsecase.WalletRepo (minimal port)
// + legacy/extended wallet operations over Firestore.
// =====================================================

type WalletRepositoryFS struct {
	Client *firestore.Client
}

func NewWalletRepositoryFS(client *firestore.Client) *WalletRepositoryFS {
	return &WalletRepositoryFS{Client: client}
}

func (r *WalletRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("wallets")
}

// =====================================================
// Minimal WalletRepo methods (used by WalletUsecase)
// =====================================================

// GetByID treats id as walletAddress (document ID).
func (r *WalletRepositoryFS) GetByID(ctx context.Context, id string) (wdom.Wallet, error) {
	if r.Client == nil {
		return wdom.Wallet{}, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(id)
	if addr == "" {
		return wdom.Wallet{}, wdom.ErrNotFound
	}

	snap, err := r.col().Doc(addr).Get(ctx)
	if grpcstatus.Code(err) == codes.NotFound {
		return wdom.Wallet{}, wdom.ErrNotFound
	}
	if err != nil {
		return wdom.Wallet{}, err
	}

	return docToWallet(snap)
}

// Exists reports whether a wallet doc exists for the given id.
func (r *WalletRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(id)
	if addr == "" {
		return false, nil
	}

	_, err := r.col().Doc(addr).Get(ctx)
	if grpcstatus.Code(err) == codes.NotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Create implements WalletRepo.Create using Wallet as input.
func (r *WalletRepositoryFS) Create(ctx context.Context, v wdom.Wallet) (wdom.Wallet, error) {
	if r.Client == nil {
		return wdom.Wallet{}, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(v.WalletAddress)
	if addr == "" {
		return wdom.Wallet{}, errors.New("wallet: walletAddress is required")
	}

	now := time.Now().UTC()

	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	if v.LastUpdatedAt.IsZero() {
		v.LastUpdatedAt = v.CreatedAt
	}
	if strings.TrimSpace(string(v.Status)) == "" {
		v.Status = wdom.WalletStatus("active")
	}
	v.Tokens = dedupStrings(v.Tokens)

	ref := r.col().Doc(addr)
	data := map[string]any{
		"tokens":        v.Tokens,
		"status":        string(v.Status),
		"createdAt":     v.CreatedAt.UTC(),
		"updatedAt":     v.UpdatedAt.UTC(),
		"lastUpdatedAt": v.LastUpdatedAt.UTC(),
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if grpcstatus.Code(err) == codes.AlreadyExists {
			return wdom.Wallet{}, wdom.ErrConflict
		}
		return wdom.Wallet{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return wdom.Wallet{}, err
	}
	return docToWallet(snap)
}

// Save implements WalletRepo.Save as an upsert.
func (r *WalletRepositoryFS) Save(ctx context.Context, v wdom.Wallet) (wdom.Wallet, error) {
	if r.Client == nil {
		return wdom.Wallet{}, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(v.WalletAddress)
	if addr == "" {
		return wdom.Wallet{}, errors.New("wallet: walletAddress is required")
	}

	now := time.Now().UTC()

	if v.CreatedAt.IsZero() {
		v.CreatedAt = now
	}
	if v.UpdatedAt.IsZero() {
		v.UpdatedAt = now
	}
	if v.LastUpdatedAt.IsZero() {
		v.LastUpdatedAt = v.CreatedAt
	}
	if strings.TrimSpace(string(v.Status)) == "" {
		v.Status = wdom.WalletStatus("active")
	}
	v.Tokens = dedupStrings(v.Tokens)

	ref := r.col().Doc(addr)
	data := map[string]any{
		"tokens":        v.Tokens,
		"status":        string(v.Status),
		"createdAt":     v.CreatedAt.UTC(),
		"updatedAt":     v.UpdatedAt.UTC(),
		"lastUpdatedAt": v.LastUpdatedAt.UTC(),
	}

	if _, err := ref.Set(ctx, data); err != nil {
		return wdom.Wallet{}, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return wdom.Wallet{}, err
	}
	return docToWallet(snap)
}

// Delete implements WalletRepo.Delete.
func (r *WalletRepositoryFS) Delete(ctx context.Context, id string) error {
	return r.DeleteWallet(ctx, id)
}

// =====================================================
// Existing/extended RepositoryPort-style methods
// =====================================================

// GetAllWallets returns all wallets ordered similarly to the PG implementation.
func (r *WalletRepositoryFS) GetAllWallets(ctx context.Context) ([]*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.col().
		OrderBy("updatedAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Asc).
		Documents(ctx)
	defer it.Stop()

	var out []*wdom.Wallet
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return nil, err
		}
		ww := w
		out = append(out, &ww)
	}
	return out, nil
}

func (r *WalletRepositoryFS) GetWalletByAddress(ctx context.Context, walletAddress string) (*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return nil, wdom.ErrNotFound
	}

	snap, err := r.col().Doc(addr).Get(ctx)
	if grpcstatus.Code(err) == codes.NotFound {
		return nil, wdom.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	w, err := docToWallet(snap)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// CreateWallet: legacy-style creation from CreateWalletInput.
func (r *WalletRepositoryFS) CreateWallet(ctx context.Context, in wdom.CreateWalletInput) (*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(in.WalletAddress)
	if addr == "" {
		return nil, errors.New("wallet: walletAddress is required")
	}

	now := time.Now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	updatedAt := now
	if in.UpdatedAt != nil && !in.UpdatedAt.IsZero() {
		updatedAt = in.UpdatedAt.UTC()
	}

	lastUpdatedAt := createdAt
	if in.LastUpdatedAt != nil && !in.LastUpdatedAt.IsZero() {
		lastUpdatedAt = in.LastUpdatedAt.UTC()
	}

	statusStr := "active"
	if in.Status != nil {
		if s := strings.TrimSpace(string(*in.Status)); s != "" {
			statusStr = s
		}
	}

	ref := r.col().Doc(addr)
	data := map[string]any{
		"tokens":        dedupStrings(in.Tokens),
		"status":        statusStr,
		"createdAt":     createdAt,
		"updatedAt":     updatedAt,
		"lastUpdatedAt": lastUpdatedAt,
	}

	if _, err := ref.Create(ctx, data); err != nil {
		if grpcstatus.Code(err) == codes.AlreadyExists {
			return nil, wdom.ErrConflict
		}
		return nil, err
	}

	snap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}
	w, err := docToWallet(snap)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// UpdateWallet: 差分更新API向け。
func (r *WalletRepositoryFS) UpdateWallet(ctx context.Context, walletAddress string, in wdom.UpdateWalletInput) (*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return nil, errors.New("wallet: walletAddress is required")
	}

	ref := r.col().Doc(addr)
	var result *wdom.Wallet

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if grpcstatus.Code(err) == codes.NotFound {
			return wdom.ErrNotFound
		}
		if err != nil {
			return err
		}

		cur, err := docToWallet(snap)
		if err != nil {
			return err
		}

		changedTokens := false
		now := time.Now().UTC()

		if in.Tokens != nil {
			cur.Tokens = dedupStrings(*in.Tokens)
			changedTokens = true
		}

		if len(in.AddTokens) > 0 {
			set := make(map[string]struct{}, len(cur.Tokens))
			for _, t := range cur.Tokens {
				set[t] = struct{}{}
			}
			for _, t := range dedupStrings(in.AddTokens) {
				if _, ok := set[t]; !ok {
					cur.Tokens = append(cur.Tokens, t)
					set[t] = struct{}{}
				}
			}
			changedTokens = true
		}

		if len(in.RemoveTokens) > 0 {
			rm := make(map[string]struct{}, len(in.RemoveTokens))
			for _, t := range dedupStrings(in.RemoveTokens) {
				rm[t] = struct{}{}
			}
			newTokens := make([]string, 0, len(cur.Tokens))
			for _, t := range cur.Tokens {
				if _, ok := rm[t]; !ok {
					newTokens = append(newTokens, t)
				}
			}
			if len(newTokens) != len(cur.Tokens) {
				cur.Tokens = newTokens
				changedTokens = true
			}
		}

		if in.Status != nil {
			cur.Status = *in.Status
		}

		if in.UpdatedAt != nil {
			cur.UpdatedAt = in.UpdatedAt.UTC()
		} else {
			cur.UpdatedAt = now
		}

		if in.LastUpdatedAt != nil {
			cur.LastUpdatedAt = in.LastUpdatedAt.UTC()
		} else if changedTokens {
			cur.LastUpdatedAt = now
		}

		data := map[string]any{
			"tokens":        dedupStrings(cur.Tokens),
			"status":        string(cur.Status),
			"createdAt":     cur.CreatedAt.UTC(),
			"updatedAt":     cur.UpdatedAt.UTC(),
			"lastUpdatedAt": cur.LastUpdatedAt.UTC(),
		}

		if err := tx.Set(ref, data); err != nil {
			return err
		}

		tmp := cur
		result = &tmp
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, wdom.ErrNotFound
	}
	return result, nil
}

// DeleteWallet: legacy-style delete.
func (r *WalletRepositoryFS) DeleteWallet(ctx context.Context, walletAddress string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return wdom.ErrNotFound
	}

	ref := r.col().Doc(addr)
	if _, err := ref.Get(ctx); grpcstatus.Code(err) == codes.NotFound {
		return wdom.ErrNotFound
	} else if err != nil {
		return err
	}

	if _, err := ref.Delete(ctx); err != nil {
		return err
	}
	return nil
}

// SearchWallets: 全件取得してメモリ上でフィルタ・ソート・ページング。
func (r *WalletRepositoryFS) SearchWallets(ctx context.Context, opts wdom.WalletSearchOptions) (wdom.WalletPaginationResult, error) {
	if r.Client == nil {
		return wdom.WalletPaginationResult{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var all []*wdom.Wallet
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return wdom.WalletPaginationResult{}, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return wdom.WalletPaginationResult{}, err
		}
		ww := w
		if matchWalletFilter(&ww, opts.Filter) {
			all = append(all, &ww)
		}
	}

	sortWallets(all, opts.Sort)

	pageNum := safePage(opts.Pagination)
	perPage := safePerPage(opts.Pagination)
	pageNum, perPage, offset := dbcommon.NormalizePage(pageNum, perPage, 50, 200)

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}

	items := all[offset:end]
	totalPages := dbcommon.ComputeTotalPages(total, perPage)

	return wdom.WalletPaginationResult{
		Wallets:         items,
		TotalPages:      totalPages,
		TotalCount:      total,
		CurrentPage:     pageNum,
		ItemsPerPage:    perPage,
		HasNextPage:     pageNum < totalPages,
		HasPreviousPage: pageNum > 1,
	}, nil
}

// トークン操作系など

func (r *WalletRepositoryFS) AddTokenToWallet(ctx context.Context, walletAddress, mintAddress string) (*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(walletAddress)
	token := strings.TrimSpace(mintAddress)
	if addr == "" || token == "" {
		return nil, errors.New("wallet: walletAddress and mintAddress are required")
	}

	ref := r.col().Doc(addr)
	var result *wdom.Wallet

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if grpcstatus.Code(err) == codes.NotFound {
			return wdom.ErrNotFound
		}
		if err != nil {
			return err
		}
		cur, err := docToWallet(snap)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		exists := false
		for _, t := range cur.Tokens {
			if t == token {
				exists = true
				break
			}
		}
		if !exists {
			cur.Tokens = append(cur.Tokens, token)
			cur.LastUpdatedAt = now
		}
		cur.UpdatedAt = now

		data := map[string]any{
			"tokens":        dedupStrings(cur.Tokens),
			"status":        string(cur.Status),
			"createdAt":     cur.CreatedAt.UTC(),
			"updatedAt":     cur.UpdatedAt.UTC(),
			"lastUpdatedAt": cur.LastUpdatedAt.UTC(),
		}
		if err := tx.Set(ref, data); err != nil {
			return err
		}
		tmp := cur
		result = &tmp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *WalletRepositoryFS) RemoveTokenFromWallet(ctx context.Context, walletAddress, mintAddress string) (*wdom.Wallet, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	addr := strings.TrimSpace(walletAddress)
	token := strings.TrimSpace(mintAddress)
	if addr == "" || token == "" {
		return nil, errors.New("wallet: walletAddress and mintAddress are required")
	}

	ref := r.col().Doc(addr)
	var result *wdom.Wallet

	err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(ref)
		if grpcstatus.Code(err) == codes.NotFound {
			return wdom.ErrNotFound
		}
		if err != nil {
			return err
		}
		cur, err := docToWallet(snap)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		changed := false
		newTokens := make([]string, 0, len(cur.Tokens))
		for _, t := range cur.Tokens {
			if t == token {
				changed = true
				continue
			}
			newTokens = append(newTokens, t)
		}
		if changed {
			cur.Tokens = newTokens
			cur.LastUpdatedAt = now
		}
		cur.UpdatedAt = now

		data := map[string]any{
			"tokens":        dedupStrings(cur.Tokens),
			"status":        string(cur.Status),
			"createdAt":     cur.CreatedAt.UTC(),
			"updatedAt":     cur.UpdatedAt.UTC(),
			"lastUpdatedAt": cur.LastUpdatedAt.UTC(),
		}
		if err := tx.Set(ref, data); err != nil {
			return err
		}
		tmp := cur
		result = &tmp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *WalletRepositoryFS) AddTokensToWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*wdom.Wallet, error) {
	var last *wdom.Wallet
	for _, m := range dedupStrings(mintAddresses) {
		w, err := r.AddTokenToWallet(ctx, walletAddress, m)
		if err != nil {
			return nil, err
		}
		last = w
	}
	if last == nil {
		return r.GetWalletByAddress(ctx, walletAddress)
	}
	return last, nil
}

func (r *WalletRepositoryFS) RemoveTokensFromWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*wdom.Wallet, error) {
	var last *wdom.Wallet
	for _, m := range dedupStrings(mintAddresses) {
		w, err := r.RemoveTokenFromWallet(ctx, walletAddress, m)
		if err != nil {
			return nil, err
		}
		last = w
	}
	if last == nil {
		return r.GetWalletByAddress(ctx, walletAddress)
	}
	return last, nil
}

// GetWalletsBatch: 指定されたアドレスのウォレットを一括取得。
func (r *WalletRepositoryFS) GetWalletsBatch(ctx context.Context, req wdom.BatchWalletRequest) (wdom.BatchWalletResponse, error) {
	if r.Client == nil {
		return wdom.BatchWalletResponse{}, errors.New("firestore client is nil")
	}

	addresses := dedupStrings(req.WalletAddresses)
	resp := wdom.BatchWalletResponse{
		Wallets:  make([]*wdom.Wallet, 0, len(addresses)),
		NotFound: []string{},
	}
	if len(addresses) == 0 {
		return resp, nil
	}

	found := map[string]struct{}{}
	for _, addr := range addresses {
		snap, err := r.col().Doc(addr).Get(ctx)
		if grpcstatus.Code(err) == codes.NotFound {
			continue
		}
		if err != nil {
			return resp, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return resp, err
		}
		ww := w
		resp.Wallets = append(resp.Wallets, &ww)
		found[w.WalletAddress] = struct{}{}
	}

	if req.IncludeDefaults {
		for _, addr := range addresses {
			if _, ok := found[addr]; !ok {
				resp.NotFound = append(resp.NotFound, addr)
			}
		}
	}

	return resp, nil
}

// UpdateWalletsBatch: 1件ずつTx更新。
func (r *WalletRepositoryFS) UpdateWalletsBatch(ctx context.Context, updates []wdom.BatchWalletUpdate) (wdom.BatchWalletUpdateResponse, error) {
	if r.Client == nil {
		return wdom.BatchWalletUpdateResponse{}, errors.New("firestore client is nil")
	}

	res := wdom.BatchWalletUpdateResponse{
		Succeeded: []*wdom.Wallet{},
		Failed:    nil, // append時に正しい型の匿名structを使う
	}

	for _, u := range updates {
		addr := strings.TrimSpace(u.WalletAddress)
		if addr == "" {
			res.Failed = append(res.Failed, struct {
				WalletAddress string `json:"walletAddress"`
				Error         string `json:"error"`
			}{
				WalletAddress: u.WalletAddress,
				Error:         "empty walletAddress",
			})
			continue
		}

		ref := r.col().Doc(addr)

		var updated *wdom.Wallet
		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			snap, err := tx.Get(ref)
			if grpcstatus.Code(err) == codes.NotFound {
				return wdom.ErrNotFound
			}
			if err != nil {
				return err
			}
			w, err := docToWallet(snap)
			if err != nil {
				return err
			}

			changed := false

			if v, ok := u.Data["tokens"]; ok {
				if arr, ok2 := v.([]string); ok2 {
					w.Tokens = dedupStrings(arr)
					w.LastUpdatedAt = time.Now().UTC()
					changed = true
				}
			}
			if v, ok := u.Data["status"]; ok {
				if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
					w.Status = wdom.WalletStatus(strings.TrimSpace(s))
					changed = true
				}
			}

			if !changed {
				tmp := w
				updated = &tmp
				return nil
			}

			w.UpdatedAt = time.Now().UTC()

			data := map[string]any{
				"tokens":        dedupStrings(w.Tokens),
				"status":        string(w.Status),
				"createdAt":     w.CreatedAt.UTC(),
				"updatedAt":     w.UpdatedAt.UTC(),
				"lastUpdatedAt": w.LastUpdatedAt.UTC(),
			}
			if err := tx.Set(ref, data); err != nil {
				return err
			}
			tmp := w
			updated = &tmp
			return nil
		})

		if err != nil {
			res.Failed = append(res.Failed, struct {
				WalletAddress string `json:"walletAddress"`
				Error         string `json:"error"`
			}{
				WalletAddress: u.WalletAddress,
				Error:         err.Error(),
			})
			continue
		}
		if updated != nil {
			res.Succeeded = append(res.Succeeded, updated)
		}
	}

	return res, nil
}

// GetWalletStats: 全ウォレットを読み込み、統計値を計算。
func (r *WalletRepositoryFS) GetWalletStats(ctx context.Context) (wdom.WalletStats, error) {
	if r.Client == nil {
		return wdom.WalletStats{}, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var counts []int
	stats := wdom.WalletStats{}

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return wdom.WalletStats{}, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return wdom.WalletStats{}, err
		}
		stats.TotalWallets++
		c := len(w.Tokens)
		counts = append(counts, c)
		if c > 0 {
			stats.WalletsWithTokens++
			stats.TotalTokens += c
		} else {
			stats.WalletsWithoutTokens++
		}
	}

	if stats.TotalWallets > 0 {
		stats.AverageTokensPerWallet = float64(stats.TotalTokens) / float64(stats.TotalWallets)
	}

	if len(counts) > 0 {
		sort.Ints(counts)
		stats.TopHolderTokenCount = counts[len(counts)-1]
		mid := len(counts) / 2
		if len(counts)%2 == 1 {
			stats.MedianTokensPerWallet = float64(counts[mid])
		} else {
			stats.MedianTokensPerWallet = float64(counts[mid-1]+counts[mid]) / 2.0
		}
	}

	// UniqueTokenTypes
	it2 := r.col().Documents(ctx)
	defer it2.Stop()
	tokenSet := map[string]struct{}{}
	for {
		snap, err := it2.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return wdom.WalletStats{}, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return wdom.WalletStats{}, err
		}
		for _, t := range w.Tokens {
			tokenSet[t] = struct{}{}
		}
	}
	stats.UniqueTokenTypes = len(tokenSet)

	return stats, nil
}

// GetTokenDistribution: トークン数に応じたウォレット分布を計算。
func (r *WalletRepositoryFS) GetTokenDistribution(ctx context.Context) ([]wdom.TokenDistribution, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	counts := map[wdom.TokenTier]int{}
	total := 0

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return nil, err
		}
		tt := tierFromCount(len(w.Tokens))
		counts[tt]++
		total++
	}

	tiers := []wdom.TokenTier{
		wdom.TierWhale,
		wdom.TierLarge,
		wdom.TierMedium,
		wdom.TierSmall,
		wdom.TierEmpty,
	}

	out := make([]wdom.TokenDistribution, 0, len(tiers))
	for _, t := range tiers {
		cnt := counts[t]
		p := 0.0
		if total > 0 {
			p = float64(cnt) * 100.0 / float64(total)
		}
		out = append(out, wdom.TokenDistribution{
			Tier:       t,
			Count:      cnt,
			Percentage: p,
		})
	}
	return out, nil
}

// GetTokenHoldingStats: 特定トークンを保有するウォレット情報を計算。
func (r *WalletRepositoryFS) GetTokenHoldingStats(ctx context.Context, tokenID string) (wdom.TokenHoldingStats, error) {
	if r.Client == nil {
		return wdom.TokenHoldingStats{}, errors.New("firestore client is nil")
	}

	tokenID = strings.TrimSpace(tokenID)
	res := wdom.TokenHoldingStats{TokenID: tokenID}
	if tokenID == "" {
		return res, nil
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	type holder struct {
		addr string
		cnt  int
	}
	var holders []holder

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return wdom.TokenHoldingStats{}, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return wdom.TokenHoldingStats{}, err
		}
		cnt := len(w.Tokens)
		has := false
		for _, t := range w.Tokens {
			if t == tokenID {
				has = true
				break
			}
		}
		if has {
			res.HolderCount++
			res.TotalHoldings++
			holders = append(holders, holder{addr: w.WalletAddress, cnt: cnt})
		}
	}

	sort.SliceStable(holders, func(i, j int) bool {
		if holders[i].cnt == holders[j].cnt {
			return holders[i].addr < holders[j].addr
		}
		return holders[i].cnt > holders[j].cnt
	})

	limit := 10
	if len(holders) < limit {
		limit = len(holders)
	}
	for i := 0; i < limit; i++ {
		h := holders[i]
		res.TopHolders = append(res.TopHolders, struct {
			WalletAddress string `json:"walletAddress"`
			TokenCount    int    `json:"tokenCount"`
			Rank          int    `json:"rank"`
		}{
			WalletAddress: h.addr,
			TokenCount:    h.cnt,
			Rank:          i + 1,
		})
	}

	return res, nil
}

// GetWalletRanking: トークン数ベースのランキング。
func (r *WalletRepositoryFS) GetWalletRanking(ctx context.Context, req wdom.WalletRankingRequest) (wdom.WalletRankingResponse, error) {
	if r.Client == nil {
		return wdom.WalletRankingResponse{}, errors.New("firestore client is nil")
	}

	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	var all []struct {
		w   *wdom.Wallet
		cnt int
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var tokenFilter string
	if req.TokenID != nil {
		tokenFilter = strings.TrimSpace(*req.TokenID)
	}

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return wdom.WalletRankingResponse{}, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return wdom.WalletRankingResponse{}, err
		}
		if tokenFilter != "" {
			contains := false
			for _, t := range w.Tokens {
				if t == tokenFilter {
					contains = true
					break
				}
			}
			if !contains {
				continue
			}
		}
		cnt := len(w.Tokens)
		ww := w
		all = append(all, struct {
			w   *wdom.Wallet
			cnt int
		}{w: &ww, cnt: cnt})
	}

	sort.SliceStable(all, func(i, j int) bool {
		if all[i].cnt == all[j].cnt {
			return all[i].w.WalletAddress < all[j].w.WalletAddress
		}
		return all[i].cnt > all[j].cnt
	})

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	resp := wdom.WalletRankingResponse{
		Rankings: []wdom.TopWalletInfo{},
		Total:    total,
	}

	for i := offset; i < end; i++ {
		rank := i + 1
		info := all[i]
		resp.Rankings = append(resp.Rankings, wdom.TopWalletInfo{
			Wallet:     info.w,
			Rank:       rank,
			TokenCount: info.cnt,
			TierInfo:   wdom.TokenTierDefinition{},
		})
	}

	return resp, nil
}

// GetTokenHolders: 特定トークンの上位ホルダー一覧。
func (r *WalletRepositoryFS) GetTokenHolders(ctx context.Context, tokenID string, limit int) ([]wdom.TokenHolder, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return []wdom.TokenHolder{}, nil
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	type holder struct {
		addr string
		cnt  int
	}
	var hs []holder

	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		w, err := docToWallet(snap)
		if err != nil {
			return nil, err
		}
		has := false
		for _, t := range w.Tokens {
			if t == tokenID {
				has = true
				break
			}
		}
		if !has {
			continue
		}
		hs = append(hs, holder{addr: w.WalletAddress, cnt: len(w.Tokens)})
	}

	sort.SliceStable(hs, func(i, j int) bool {
		if hs[i].cnt == hs[j].cnt {
			return hs[i].addr < hs[j].addr
		}
		return hs[i].cnt > hs[j].cnt
	})

	if len(hs) > limit {
		hs = hs[:limit]
	}

	out := make([]wdom.TokenHolder, 0, len(hs))
	for _, h := range hs {
		out = append(out, wdom.TokenHolder{
			WalletAddress: h.addr,
			TokenCount:    h.cnt,
			Percentage:    nil,
			Tier:          tierFromCount(h.cnt),
		})
	}
	return out, nil
}

// ResetWallets: 全削除 (テスト用途など)
func (r *WalletRepositoryFS) ResetWallets(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	var snaps []*firestore.DocumentSnapshot
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}
		snaps = append(snaps, snap)
	}

	const chunkSize = 400
	for i := 0; i < len(snaps); i += chunkSize {
		end := i + chunkSize
		if end > len(snaps) {
			end = len(snaps)
		}
		if err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, s := range snaps[i:end] {
				if err := tx.Delete(s.Ref); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// WithTx: Firestore 用の簡易トランザクションヘルパー。
func (r *WalletRepositoryFS) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	return fn(ctx)
}

// =====================================================
// Helpers (Firestore <-> Domain)
// =====================================================

func docToWallet(doc *firestore.DocumentSnapshot) (wdom.Wallet, error) {
	data := doc.Data()
	if data == nil {
		return wdom.Wallet{}, wdom.ErrNotFound
	}

	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok {
			return v.UTC()
		}
		return time.Time{}
	}

	getStatus := func(key string) wdom.WalletStatus {
		if v, ok := data[key].(string); ok {
			return wdom.WalletStatus(strings.TrimSpace(v))
		}
		return wdom.WalletStatus("active")
	}

	getTokens := func(key string) []string {
		raw, ok := data[key]
		if !ok {
			return []string{}
		}
		switch vv := raw.(type) {
		case []string:
			return dedupStrings(vv)
		case []interface{}:
			out := make([]string, 0, len(vv))
			for _, x := range vv {
				if s, ok := x.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						out = append(out, s)
					}
				}
			}
			return dedupStrings(out)
		default:
			return []string{}
		}
	}

	return wdom.Wallet{
		WalletAddress: strings.TrimSpace(doc.Ref.ID),
		Tokens:        getTokens("tokens"),
		Status:        getStatus("status"),
		CreatedAt:     getTime("createdAt"),
		UpdatedAt:     getTime("updatedAt"),
		LastUpdatedAt: getTime("lastUpdatedAt"),
	}, nil
}

func matchWalletFilter(w *wdom.Wallet, f *wdom.WalletFilter) bool {
	if f == nil {
		return true
	}

	if v := strings.TrimSpace(f.SearchQuery); v != "" {
		if !strings.Contains(strings.ToLower(w.WalletAddress), strings.ToLower(v)) {
			return false
		}
	}

	if f.HasTokensOnly && len(w.Tokens) == 0 {
		return false
	}

	if f.MinTokenCount != nil && len(w.Tokens) < *f.MinTokenCount {
		return false
	}
	if f.MaxTokenCount != nil && len(w.Tokens) > *f.MaxTokenCount {
		return false
	}

	if len(f.TokenIDs) > 0 {
		want := map[string]struct{}{}
		for _, t := range dedupStrings(f.TokenIDs) {
			want[t] = struct{}{}
		}
		ok := false
		for _, t := range w.Tokens {
			if _, exists := want[t]; exists {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if len(f.Statuses) > 0 {
		ok := false
		for _, s := range f.Statuses {
			if w.Status == s {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}

	if len(f.Tiers) > 0 {
		okTier := false
		c := len(w.Tokens)
		for _, t := range f.Tiers {
			switch strings.ToLower(string(t)) {
			case "whale":
				if c >= 100 {
					okTier = true
				}
			case "large":
				if c >= 50 && c < 100 {
					okTier = true
				}
			case "medium":
				if c >= 10 && c < 50 {
					okTier = true
				}
			case "small":
				if c >= 1 && c < 10 {
					okTier = true
				}
			case "empty":
				if c == 0 {
					okTier = true
				}
			}
			if okTier {
				break
			}
		}
		if !okTier {
			return false
		}
	}

	checkTime := func(t time.Time, from, to *time.Time) bool {
		if !t.IsZero() {
			if from != nil && t.Before(from.UTC()) {
				return false
			}
			if to != nil && !t.Before(to.UTC()) {
				return false
			}
		}
		return true
	}

	if !checkTime(w.LastUpdatedAt, f.LastUpdatedAfter, f.LastUpdatedBefore) {
		return false
	}
	if !checkTime(w.CreatedAt, f.CreatedAfter, f.CreatedBefore) {
		return false
	}
	if !checkTime(w.UpdatedAt, f.UpdatedAfter, f.UpdatedBefore) {
		return false
	}

	return true
}

func sortWallets(items []*wdom.Wallet, sortCfg *wdom.WalletSortConfig) {
	if sortCfg == nil {
		sort.SliceStable(items, func(i, j int) bool {
			a, b := items[i], items[j]
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return a.WalletAddress < b.WalletAddress
			}
			return a.UpdatedAt.After(b.UpdatedAt)
		})
		return
	}

	col := strings.ToLower(strings.TrimSpace(sortCfg.Column))
	dir := strings.ToUpper(strings.TrimSpace(sortCfg.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	asc := dir == "ASC"

	less := func(i, j int) bool {
		a, b := items[i], items[j]

		switch col {
		case "walletaddress", "wallet_address":
			if a.WalletAddress == b.WalletAddress {
				if asc {
					return a.UpdatedAt.Before(b.UpdatedAt)
				}
				return a.UpdatedAt.After(b.UpdatedAt)
			}
			if asc {
				return a.WalletAddress < b.WalletAddress
			}
			return a.WalletAddress > b.WalletAddress

		case "tokencount", "token_count":
			if len(a.Tokens) == len(b.Tokens) {
				if asc {
					return a.WalletAddress < b.WalletAddress
				}
				return a.WalletAddress > b.WalletAddress
			}
			if asc {
				return len(a.Tokens) < len(b.Tokens)
			}
			return len(a.Tokens) > len(b.Tokens)

		case "lastupdatedat", "last_updated_at":
			if a.LastUpdatedAt.Equal(b.LastUpdatedAt) {
				if asc {
					return a.WalletAddress < b.WalletAddress
				}
				return a.WalletAddress > b.WalletAddress
			}
			if asc {
				return a.LastUpdatedAt.Before(b.LastUpdatedAt)
			}
			return a.LastUpdatedAt.After(b.LastUpdatedAt)

		case "createdat", "created_at":
			if a.CreatedAt.Equal(b.CreatedAt) {
				if asc {
					return a.WalletAddress < b.WalletAddress
				}
				return a.WalletAddress > b.WalletAddress
			}
			if asc {
				return a.CreatedAt.Before(b.CreatedAt)
			}
			return a.CreatedAt.After(b.CreatedAt)

		case "updatedat", "updated_at":
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				if asc {
					return a.WalletAddress < b.WalletAddress
				}
				return a.WalletAddress > b.WalletAddress
			}
			if asc {
				return a.UpdatedAt.Before(b.UpdatedAt)
			}
			return a.UpdatedAt.After(b.UpdatedAt)

		case "status":
			if a.Status == b.Status {
				if asc {
					return a.WalletAddress < b.WalletAddress
				}
				return a.WalletAddress > b.WalletAddress
			}
			if asc {
				return string(a.Status) < string(b.Status)
			}
			return string(a.Status) > string(b.Status)

		default:
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return a.WalletAddress < b.WalletAddress
			}
			return a.UpdatedAt.After(b.UpdatedAt)
		}
	}

	sort.SliceStable(items, less)
}

// =====================================================
// Small helpers
// =====================================================

func safePage(p *wdom.WalletPaginationOptions) int {
	if p == nil || p.Page <= 0 {
		return 1
	}
	return p.Page
}

func safePerPage(p *wdom.WalletPaginationOptions) int {
	if p == nil || p.ItemsPerPage <= 0 {
		return 50
	}
	return p.ItemsPerPage
}

func dedupStrings(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func tierFromCount(c int) wdom.TokenTier {
	switch {
	case c >= 100:
		return wdom.TierWhale
	case c >= 50:
		return wdom.TierLarge
	case c >= 10:
		return wdom.TierMedium
	case c >= 1:
		return wdom.TierSmall
	default:
		return wdom.TierEmpty
	}
}
