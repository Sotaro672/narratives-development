// backend\internal\adapters\out\firestore\wallet_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	dbcommon "narratives/internal/adapters/out/db/common"
	wdom "narratives/internal/domain/wallet"
)

type WalletRepositoryPG struct {
	DB *sql.DB
}

func NewWalletRepositoryPG(db *sql.DB) *WalletRepositoryPG {
	return &WalletRepositoryPG{DB: db}
}

// =====================================================
// usecase.WalletRepo 準拠メソッド
// =====================================================

// GetByID implements WalletRepo.GetByID.
// id は wallet_address とみなす。
func (r *WalletRepositoryPG) GetByID(ctx context.Context, id string) (wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
FROM wallets
WHERE wallet_address = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wdom.Wallet{}, wdom.ErrNotFound
		}
		return wdom.Wallet{}, err
	}
	return w, nil
}

// Exists implements WalletRepo.Exists.
func (r *WalletRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `SELECT 1 FROM wallets WHERE wallet_address = $1 LIMIT 1`
	var dummy int
	err := run.QueryRowContext(ctx, q, strings.TrimSpace(id)).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Create implements WalletRepo.Create.
//
// v.WalletAddress が必須。timestamps は v に値がなければ now() で埋める。
func (r *WalletRepositoryPG) Create(ctx context.Context, v wdom.Wallet) (wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	addr := strings.TrimSpace(v.WalletAddress)
	if addr == "" {
		return wdom.Wallet{}, wdom.ErrInvalidWalletAddress
	}

	now := time.Now().UTC()

	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	} else {
		createdAt = createdAt.UTC()
	}

	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = now
	} else {
		updatedAt = updatedAt.UTC()
	}

	lastUpdatedAt := v.LastUpdatedAt
	if lastUpdatedAt.IsZero() {
		lastUpdatedAt = createdAt
	} else {
		lastUpdatedAt = lastUpdatedAt.UTC()
	}

	status := strings.TrimSpace(string(v.Status))
	if status == "" {
		status = "active"
	}

	const q = `
INSERT INTO wallets (
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
`
	row := run.QueryRowContext(ctx, q,
		addr,
		pq.Array(dedupStrings(v.Tokens)),
		status,
		createdAt, updatedAt, lastUpdatedAt,
	)

	w, err := scanWallet(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return wdom.Wallet{}, wdom.ErrConflict
		}
		return wdom.Wallet{}, err
	}

	return w, nil
}

// Save implements WalletRepo.Save.
//
// Wallet の全フィールドを上書きするイメージ。
// updated_at/last_updated_at には v の値を使う。
// レコードが存在しなければ ErrNotFound。
func (r *WalletRepositoryPG) Save(ctx context.Context, v wdom.Wallet) (wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	addr := strings.TrimSpace(v.WalletAddress)
	if addr == "" {
		return wdom.Wallet{}, wdom.ErrInvalidWalletAddress
	}

	status := strings.TrimSpace(string(v.Status))
	if status == "" {
		status = "active"
	}

	createdAt := v.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	} else {
		createdAt = createdAt.UTC()
	}

	updatedAt := v.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	} else {
		updatedAt = updatedAt.UTC()
	}

	lastUpdatedAt := v.LastUpdatedAt
	if lastUpdatedAt.IsZero() {
		lastUpdatedAt = updatedAt
	} else {
		lastUpdatedAt = lastUpdatedAt.UTC()
	}

	const q = `
UPDATE wallets
SET
  tokens          = $2,
  status          = $3,
  created_at      = $4,
  updated_at      = $5,
  last_updated_at = $6
WHERE wallet_address = $1
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at
`
	row := run.QueryRowContext(ctx, q,
		addr,
		pq.Array(dedupStrings(v.Tokens)),
		status,
		createdAt,
		updatedAt,
		lastUpdatedAt,
	)

	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wdom.Wallet{}, wdom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return wdom.Wallet{}, wdom.ErrConflict
		}
		return wdom.Wallet{}, err
	}

	return w, nil
}

// Delete implements WalletRepo.Delete.
func (r *WalletRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx,
		`DELETE FROM wallets WHERE wallet_address = $1`,
		strings.TrimSpace(id),
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return wdom.ErrNotFound
	}
	return nil
}

// =====================================================
// 既存の拡張系メソッド (Searchやランキングなど)
// =====================================================

func (r *WalletRepositoryPG) GetAllWallets(ctx context.Context) ([]*wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
FROM wallets
ORDER BY updated_at DESC, wallet_address ASC`
	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*wdom.Wallet
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		ww := w
		out = append(out, &ww)
	}
	return out, rows.Err()
}

func (r *WalletRepositoryPG) GetWalletByAddress(ctx context.Context, walletAddress string) (*wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
FROM wallets
WHERE wallet_address = $1
LIMIT 1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(walletAddress))
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wdom.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepositoryPG) SearchWallets(ctx context.Context, opts wdom.WalletSearchOptions) (wdom.WalletPaginationResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildWalletWhere(opts)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildWalletOrderBy(opts.Sort)
	if orderBy == "" {
		orderBy = "ORDER BY updated_at DESC, wallet_address ASC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(
		safePage(opts.Pagination),
		safePerPage(opts.Pagination),
		50, 200,
	)

	// Count
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM wallets "+whereSQL, args...).Scan(&total); err != nil {
		return wdom.WalletPaginationResult{}, err
	}

	// Data
	q := fmt.Sprintf(`
SELECT
  wallet_address, tokens, status, created_at, updated_at, last_updated_at
FROM wallets
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return wdom.WalletPaginationResult{}, err
	}
	defer rows.Close()

	items := make([]*wdom.Wallet, 0, perPage)
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return wdom.WalletPaginationResult{}, err
		}
		ww := w
		items = append(items, &ww)
	}
	if err := rows.Err(); err != nil {
		return wdom.WalletPaginationResult{}, err
	}

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

// UpdateWallet: 差分更新API向け。既存コードそのまま。
func (r *WalletRepositoryPG) UpdateWallet(ctx context.Context, walletAddress string, in wdom.UpdateWalletInput) (*wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// tokens 丸ごと置き換えパス
	if in.Tokens != nil {
		toks := dedupStrings(*in.Tokens)
		const q = `
UPDATE wallets
SET tokens = $1,
    updated_at = CASE WHEN $2::timestamptz IS NULL THEN NOW() ELSE $2 END,
    last_updated_at = CASE WHEN $3::timestamptz IS NULL THEN NOW() ELSE $3 END,
    status = COALESCE($4, status)
WHERE wallet_address = $5
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at
`
		var updAt, lastUpdAt any
		if in.UpdatedAt != nil {
			updAt = in.UpdatedAt.UTC()
		}
		if in.LastUpdatedAt != nil {
			lastUpdAt = in.LastUpdatedAt.UTC()
		}
		var status *string
		if in.Status != nil {
			s := strings.TrimSpace(string(*in.Status))
			status = &s
		}
		row := run.QueryRowContext(ctx, q,
			pq.Array(toks),
			updAt, lastUpdAt, status,
			strings.TrimSpace(walletAddress),
		)
		w, err := scanWallet(row)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, wdom.ErrNotFound
			}
			return nil, err
		}
		return &w, nil
	}

	// 差分 (AddTokens / RemoveTokens / Status / UpdatedAt系)
	sets := []string{
		"updated_at = NOW()",
	}
	args := []any{}
	i := 1

	// AddTokens
	if len(in.AddTokens) > 0 {
		for _, t := range dedupStrings(in.AddTokens) {
			sets = append(sets, fmt.Sprintf(`
tokens = CASE WHEN array_position(tokens, $%d) IS NULL THEN array_append(tokens, $%d) ELSE tokens END`, i, i))
			args = append(args, strings.TrimSpace(t))

			sets = append(sets, fmt.Sprintf(`
last_updated_at = CASE WHEN array_position(tokens, $%d) IS NULL THEN NOW() ELSE last_updated_at END`, i))
			i++
		}
	}

	// RemoveTokens
	if len(in.RemoveTokens) > 0 {
		for _, t := range dedupStrings(in.RemoveTokens) {
			sets = append(sets, fmt.Sprintf(`
tokens = CASE WHEN array_position(tokens, $%d) IS NULL THEN tokens ELSE array_remove(tokens, $%d) END`, i, i))
			args = append(args, strings.TrimSpace(t))

			sets = append(sets, fmt.Sprintf(`
last_updated_at = CASE WHEN array_position(tokens, $%d) IS NULL THEN last_updated_at ELSE NOW() END`, i))
			i++
		}
	}

	// Status
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(string(*in.Status)))
		i++
	}

	// Optional overrides
	if in.UpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("updated_at = $%d", i))
		args = append(args, in.UpdatedAt.UTC())
		i++
	}
	if in.LastUpdatedAt != nil {
		sets = append(sets, fmt.Sprintf("last_updated_at = $%d", i))
		args = append(args, in.LastUpdatedAt.UTC())
		i++
	}

	if len(sets) == 0 {
		// no-op → 現状返す
		return r.GetWalletByAddress(ctx, walletAddress)
	}

	args = append(args, strings.TrimSpace(walletAddress))
	q := fmt.Sprintf(`
UPDATE wallets
SET %s
WHERE wallet_address = $%d
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wdom.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepositoryPG) DeleteWallet(ctx context.Context, walletAddress string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx,
		`DELETE FROM wallets WHERE wallet_address = $1`,
		strings.TrimSpace(walletAddress),
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return wdom.ErrNotFound
	}
	return nil
}

func (r *WalletRepositoryPG) AddTokenToWallet(ctx context.Context, walletAddress, mintAddress string) (*wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
UPDATE wallets
SET
  tokens = CASE WHEN array_position(tokens, $1) IS NULL THEN array_append(tokens, $1) ELSE tokens END,
  updated_at = NOW(),
  last_updated_at = CASE WHEN array_position(tokens, $1) IS NULL THEN NOW() ELSE last_updated_at END
WHERE wallet_address = $2
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(mintAddress),
		strings.TrimSpace(walletAddress),
	)
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wdom.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepositoryPG) RemoveTokenFromWallet(ctx context.Context, walletAddress, mintAddress string) (*wdom.Wallet, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
UPDATE wallets
SET
  tokens = CASE WHEN array_position(tokens, $1) IS NULL THEN tokens ELSE array_remove(tokens, $1) END,
  updated_at = NOW(),
  last_updated_at = CASE WHEN array_position(tokens, $1) IS NULL THEN last_updated_at ELSE NOW() END
WHERE wallet_address = $2
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(mintAddress),
		strings.TrimSpace(walletAddress),
	)
	w, err := scanWallet(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, wdom.ErrNotFound
		}
		return nil, err
	}
	return &w, nil
}

func (r *WalletRepositoryPG) AddTokensToWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*wdom.Wallet, error) {
	var last *wdom.Wallet
	if err := r.WithTx(ctx, func(txCtx context.Context) error {
		for _, m := range dedupStrings(mintAddresses) {
			w, err := r.AddTokenToWallet(txCtx, walletAddress, m)
			if err != nil {
				return err
			}
			last = w
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if last == nil {
		return r.GetWalletByAddress(ctx, walletAddress)
	}
	return last, nil
}

func (r *WalletRepositoryPG) RemoveTokensFromWallet(ctx context.Context, walletAddress string, mintAddresses []string) (*wdom.Wallet, error) {
	var last *wdom.Wallet
	if err := r.WithTx(ctx, func(txCtx context.Context) error {
		for _, m := range dedupStrings(mintAddresses) {
			w, err := r.RemoveTokenFromWallet(txCtx, walletAddress, m)
			if err != nil {
				return err
			}
			last = w
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if last == nil {
		return r.GetWalletByAddress(ctx, walletAddress)
	}
	return last, nil
}

func (r *WalletRepositoryPG) GetWalletsBatch(ctx context.Context, req wdom.BatchWalletRequest) (wdom.BatchWalletResponse, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	addresses := dedupStrings(req.WalletAddresses)
	resp := wdom.BatchWalletResponse{
		Wallets:  make([]*wdom.Wallet, 0, len(addresses)),
		NotFound: []string{},
	}
	if len(addresses) == 0 {
		return resp, nil
	}

	q := `
SELECT wallet_address, tokens, status, created_at, updated_at, last_updated_at
FROM wallets
WHERE wallet_address = ANY($1)
`
	rows, err := run.QueryContext(ctx, q, pq.Array(addresses))
	if err != nil {
		return resp, err
	}
	defer rows.Close()

	found := map[string]struct{}{}
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return resp, err
		}
		ww := w
		resp.Wallets = append(resp.Wallets, &ww)
		found[w.WalletAddress] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return resp, err
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

func (r *WalletRepositoryPG) UpdateWalletsBatch(ctx context.Context, updates []wdom.BatchWalletUpdate) (wdom.BatchWalletUpdateResponse, error) {
	res := wdom.BatchWalletUpdateResponse{
		Succeeded: []*wdom.Wallet{},
		Failed: []struct {
			WalletAddress string `json:"walletAddress"`
			Error         string `json:"error"`
		}{},
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return res, err
	}
	txCtx := dbcommon.CtxWithTx(ctx, tx)

	for _, u := range updates {
		addr := strings.TrimSpace(u.WalletAddress)
		if addr == "" {
			res.Failed = append(res.Failed, struct {
				WalletAddress string `json:"walletAddress"`
				Error         string `json:"error"`
			}{WalletAddress: u.WalletAddress, Error: "empty walletAddress"})
			continue
		}

		sets := []string{}
		args := []any{}
		i := 1

		if v, ok := u.Data["tokens"]; ok {
			if arr, ok2 := v.([]string); ok2 {
				sets = append(sets, fmt.Sprintf("tokens = $%d", i))
				args = append(args, pq.Array(dedupStrings(arr)))
				i++
				sets = append(sets, "last_updated_at = NOW()")
			}
		}

		if v, ok := u.Data["status"]; ok {
			if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
				sets = append(sets, fmt.Sprintf("status = $%d", i))
				args = append(args, strings.TrimSpace(s))
				i++
			}
		}

		sets = append(sets, "updated_at = NOW()")

		if len(sets) == 0 {
			w, err := r.GetWalletByAddress(txCtx, addr)
			if err != nil {
				res.Failed = append(res.Failed, struct {
					WalletAddress string `json:"walletAddress"`
					Error         string `json:"error"`
				}{WalletAddress: u.WalletAddress, Error: err.Error()})
				continue
			}
			res.Succeeded = append(res.Succeeded, w)
			continue
		}

		args = append(args, addr)
		q := fmt.Sprintf(`
UPDATE wallets
SET %s
WHERE wallet_address = $%d
RETURNING wallet_address, tokens, status, created_at, updated_at, last_updated_at
`, strings.Join(sets, ", "), i)

		row := tx.QueryRowContext(txCtx, q, args...)
		w, err := scanWallet(row)
		if err != nil {
			res.Failed = append(res.Failed, struct {
				WalletAddress string `json:"walletAddress"`
				Error         string `json:"error"`
			}{WalletAddress: u.WalletAddress, Error: err.Error()})
			continue
		}
		ww := w
		res.Succeeded = append(res.Succeeded, &ww)
	}

	if err := tx.Commit(); err != nil {
		return res, err
	}
	return res, nil
}

func (r *WalletRepositoryPG) GetWalletStats(ctx context.Context) (wdom.WalletStats, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	var stats wdom.WalletStats
	const q = `
WITH t AS (
  SELECT cardinality(tokens) AS cnt FROM wallets
)
SELECT
  (SELECT COUNT(*) FROM wallets) as total_wallets,
  (SELECT COUNT(*) FROM wallets WHERE cardinality(tokens) > 0) as wallets_with_tokens,
  (SELECT COUNT(*) FROM wallets WHERE cardinality(tokens) = 0) as wallets_without_tokens,
  COALESCE((SELECT SUM(cnt) FROM t), 0) as total_tokens,
  COALESCE((SELECT AVG(cnt)::float8 FROM t), 0) as avg_tokens_per_wallet,
  COALESCE((SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY cnt) FROM t), 0) as median_tokens_per_wallet,
  COALESCE((SELECT MAX(cnt) FROM t), 0) as top_holder_token_count,
  COALESCE((SELECT COUNT(DISTINCT x) FROM wallets, LATERAL unnest(tokens) AS x), 0) as unique_token_types
`
	row := run.QueryRowContext(ctx, q)
	if err := row.Scan(
		&stats.TotalWallets,
		&stats.WalletsWithTokens,
		&stats.WalletsWithoutTokens,
		&stats.TotalTokens,
		&stats.AverageTokensPerWallet,
		&stats.MedianTokensPerWallet,
		&stats.TopHolderTokenCount,
		&stats.UniqueTokenTypes,
	); err != nil {
		return wdom.WalletStats{}, err
	}
	return stats, nil
}

func (r *WalletRepositoryPG) GetTokenDistribution(ctx context.Context) ([]wdom.TokenDistribution, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
WITH t AS (
  SELECT cardinality(tokens) AS cnt FROM wallets
), totals AS (
  SELECT COUNT(*) AS total FROM t
)
SELECT 'whale' AS tier, COUNT(*) AS cnt FROM t WHERE cnt >= 100
UNION ALL
SELECT 'large', COUNT(*) FROM t WHERE cnt >= 50 AND cnt < 100
UNION ALL
SELECT 'medium', COUNT(*) FROM t WHERE cnt >= 10 AND cnt < 50
UNION ALL
SELECT 'small', COUNT(*) FROM t WHERE cnt >= 1 AND cnt < 10
UNION ALL
SELECT 'empty', COUNT(*) FROM t WHERE cnt = 0
`
	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rec struct {
		tier string
		cnt  int
	}
	recs := []rec{}
	total := 0

	for rows.Next() {
		var tier string
		var cnt int
		if err := rows.Scan(&tier, &cnt); err != nil {
			return nil, err
		}
		recs = append(recs, rec{tier: tier, cnt: cnt})
		total += cnt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]wdom.TokenDistribution, 0, len(recs))
	for _, rrec := range recs {
		p := 0.0
		if total > 0 {
			p = float64(rrec.cnt) * 100.0 / float64(total)
		}
		out = append(out, wdom.TokenDistribution{
			Tier:       wdom.TokenTier(rrec.tier),
			Count:      rrec.cnt,
			Percentage: p,
		})
	}
	return out, nil
}

func (r *WalletRepositoryPG) GetTokenHoldingStats(ctx context.Context, tokenID string) (wdom.TokenHoldingStats, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	var res wdom.TokenHoldingStats
	res.TokenID = strings.TrimSpace(tokenID)

	const holdersQ = `
SELECT wallet_address, cardinality(tokens) AS token_count
FROM wallets
WHERE tokens @> ARRAY[$1]::text[]
ORDER BY token_count DESC, wallet_address ASC
LIMIT 10
`
	rows, err := run.QueryContext(ctx, holdersQ, res.TokenID)
	if err != nil {
		return wdom.TokenHoldingStats{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var addr string
		var cnt int
		if err := rows.Scan(&addr, &cnt); err != nil {
			return wdom.TokenHoldingStats{}, err
		}
		res.TopHolders = append(res.TopHolders, struct {
			WalletAddress string `json:"walletAddress"`
			TokenCount    int    `json:"tokenCount"`
			Rank          int    `json:"rank"`
		}{
			WalletAddress: addr,
			TokenCount:    cnt,
			Rank:          len(res.TopHolders) + 1,
		})
	}
	if err := rows.Err(); err != nil {
		return wdom.TokenHoldingStats{}, err
	}

	const aggQ = `
SELECT
  COUNT(*) AS holder_count,
  COALESCE(SUM(1),0) AS total_holdings
FROM wallets
WHERE tokens @> ARRAY[$1]::text[]
`
	if err := run.QueryRowContext(ctx, aggQ, res.TokenID).Scan(
		&res.HolderCount,
		&res.TotalHoldings,
	); err != nil {
		return wdom.TokenHoldingStats{}, err
	}

	return res, nil
}

func (r *WalletRepositoryPG) GetWalletRanking(ctx context.Context, req wdom.WalletRankingRequest) (wdom.WalletRankingResponse, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	where := ""
	args := []any{}
	if req.TokenID != nil && strings.TrimSpace(*req.TokenID) != "" {
		where = "WHERE tokens @> ARRAY[$1]::text[]"
		args = append(args, strings.TrimSpace(*req.TokenID))
	}

	q := fmt.Sprintf(`
WITH ranked AS (
  SELECT
    wallet_address, tokens, status, created_at, updated_at, last_updated_at,
    cardinality(tokens) AS token_count,
    ROW_NUMBER() OVER (
      ORDER BY cardinality(tokens) DESC, wallet_address ASC
    ) AS rnk
  FROM wallets
  %s
)
SELECT wallet_address, tokens, status, created_at, updated_at, last_updated_at, token_count, rnk
FROM ranked
ORDER BY rnk
LIMIT $%d OFFSET $%d
`, where, len(args)+1, len(args)+2)

	args = append(args, limit, offset)

	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return wdom.WalletRankingResponse{}, err
	}
	defer rows.Close()

	resp := wdom.WalletRankingResponse{}
	for rows.Next() {
		var w wdom.Wallet
		var tkCnt, rank int
		var tokens []string
		if err := rows.Scan(
			&w.WalletAddress,
			pq.Array(&tokens),
			&w.Status,
			&w.CreatedAt,
			&w.UpdatedAt,
			&w.LastUpdatedAt,
			&tkCnt,
			&rank,
		); err != nil {
			return wdom.WalletRankingResponse{}, err
		}
		w.Tokens = tokens

		resp.Rankings = append(resp.Rankings, wdom.TopWalletInfo{
			Wallet:     &w,
			Rank:       rank,
			TokenCount: tkCnt,
			TierInfo:   wdom.TokenTierDefinition{}, // ここでTierInfoを埋めたいなら別ロジックで
		})
	}
	if err := rows.Err(); err != nil {
		return wdom.WalletRankingResponse{}, err
	}

	// total
	totalQ := "SELECT COUNT(*) FROM wallets"
	var total int
	if where != "" {
		totalQ += " " + where
		if err := run.QueryRowContext(ctx, totalQ, args[:len(args)-2]...).Scan(&total); err != nil {
			return wdom.WalletRankingResponse{}, err
		}
	} else {
		if err := run.QueryRowContext(ctx, totalQ).Scan(&total); err != nil {
			return wdom.WalletRankingResponse{}, err
		}
	}
	resp.Total = total

	return resp, nil
}

func (r *WalletRepositoryPG) GetTokenHolders(ctx context.Context, tokenID string, limit int) ([]wdom.TokenHolder, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	const q = `
SELECT wallet_address, cardinality(tokens) AS token_count
FROM wallets
WHERE tokens @> ARRAY[$1]::text[]
ORDER BY token_count DESC, wallet_address ASC
LIMIT $2`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(tokenID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []wdom.TokenHolder
	for rows.Next() {
		var addr string
		var cnt int
		if err := rows.Scan(&addr, &cnt); err != nil {
			return nil, err
		}
		out = append(out, wdom.TokenHolder{
			WalletAddress: addr,
			TokenCount:    cnt,
			Percentage:    nil, // 後段ロジックで計算してもOK
			Tier:          tierFromCount(cnt),
		})
	}
	return out, rows.Err()
}

func (r *WalletRepositoryPG) ResetWallets(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	// 依存ログテーブルがあるなら先に消す
	if _, err := run.ExecContext(ctx, `DELETE FROM wallet_update_logs`); err != nil {
		// あってもなくても気にしない
	}
	_, err := run.ExecContext(ctx, `DELETE FROM wallets`)
	return err
}

func (r *WalletRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	txCtx := dbcommon.CtxWithTx(ctx, tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// =====================================================
// Helpers
// =====================================================

func scanWallet(s dbcommon.RowScanner) (wdom.Wallet, error) {
	var (
		addr                                string
		tokens                              []string
		status                              string
		createdAt, updatedAt, lastUpdatedAt time.Time
	)
	if err := s.Scan(
		&addr,
		pq.Array(&tokens),
		&status,
		&createdAt,
		&updatedAt,
		&lastUpdatedAt,
	); err != nil {
		return wdom.Wallet{}, err
	}

	return wdom.Wallet{
		WalletAddress: strings.TrimSpace(addr),
		Tokens:        tokens,
		Status:        wdom.WalletStatus(strings.TrimSpace(status)),
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
		LastUpdatedAt: lastUpdatedAt.UTC(),
	}, nil
}

func buildWalletWhere(opts wdom.WalletSearchOptions) ([]string, []any) {
	where := []string{}
	args := []any{}

	f := opts.Filter
	if f == nil {
		return where, args
	}

	// 部分一致検索
	if v := strings.TrimSpace(f.SearchQuery); v != "" {
		where = append(where, fmt.Sprintf("wallet_address ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}

	// hasTokensOnly
	if f.HasTokensOnly {
		where = append(where, "cardinality(tokens) > 0")
	}

	// token count range
	if f.MinTokenCount != nil {
		where = append(where, fmt.Sprintf("cardinality(tokens) >= $%d", len(args)+1))
		args = append(args, *f.MinTokenCount)
	}
	if f.MaxTokenCount != nil {
		where = append(where, fmt.Sprintf("cardinality(tokens) <= $%d", len(args)+1))
		args = append(args, *f.MaxTokenCount)
	}

	// tokenIDs overlap
	if len(f.TokenIDs) > 0 {
		where = append(where, fmt.Sprintf("tokens && $%d", len(args)+1))
		args = append(args, pq.Array(dedupStrings(f.TokenIDs)))
	}

	// statuses
	if len(f.Statuses) > 0 {
		base := len(args)
		ph := make([]string, 0, len(f.Statuses))
		for i, s := range f.Statuses {
			args = append(args, strings.TrimSpace(string(s)))
			ph = append(ph, fmt.Sprintf("$%d", base+i+1))
		}
		where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
	}

	// tiers
	if len(f.Tiers) > 0 {
		ors := []string{}
		for _, t := range f.Tiers {
			switch strings.ToLower(string(t)) {
			case "whale":
				ors = append(ors, "cardinality(tokens) >= 100")
			case "large":
				ors = append(ors, "(cardinality(tokens) >= 50 AND cardinality(tokens) < 100)")
			case "medium":
				ors = append(ors, "(cardinality(tokens) >= 10 AND cardinality(tokens) < 50)")
			case "small":
				ors = append(ors, "(cardinality(tokens) >= 1 AND cardinality(tokens) < 10)")
			case "empty":
				ors = append(ors, "cardinality(tokens) = 0")
			}
		}
		if len(ors) > 0 {
			where = append(where, "("+strings.Join(ors, " OR ")+")")
		}
	}

	// time ranges
	addTime := func(col string, v *time.Time, op string) {
		if v != nil {
			where = append(where, fmt.Sprintf("%s %s $%d", col, op, len(args)+1))
			args = append(args, v.UTC())
		}
	}
	addTime("last_updated_at", f.LastUpdatedAfter, ">=")
	addTime("last_updated_at", f.LastUpdatedBefore, "<")
	addTime("created_at", f.CreatedAfter, ">=")
	addTime("created_at", f.CreatedBefore, "<")
	addTime("updated_at", f.UpdatedAfter, ">=")
	addTime("updated_at", f.UpdatedBefore, "<")

	return where, args
}

func buildWalletOrderBy(sort *wdom.WalletSortConfig) string {
	if sort == nil {
		return ""
	}
	col := strings.ToLower(strings.TrimSpace(sort.Column))
	switch col {
	case "walletaddress", "wallet_address":
		col = "wallet_address"
	case "tokencount", "token_count":
		col = "cardinality(tokens)"
	case "lastupdatedat", "last_updated_at":
		col = "last_updated_at"
	case "createdat", "created_at":
		col = "created_at"
	case "updatedat", "updated_at":
		col = "updated_at"
	case "status":
		col = "status"
	default:
		return ""
	}

	dir := strings.ToUpper(strings.TrimSpace(sort.Order))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}

	// セカンダリキーでソート安定化
	sec := "wallet_address"
	if col == "wallet_address" {
		sec = "updated_at"
	}

	return fmt.Sprintf("ORDER BY %s %s, %s %s", col, dir, sec, dir)
}

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
