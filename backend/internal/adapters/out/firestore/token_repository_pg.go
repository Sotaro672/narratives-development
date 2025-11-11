// backend\internal\adapters\out\firestore\token_repository_pg.go
package firestore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	dbcommon "narratives/internal/adapters/out/db/common"
	tokendom "narratives/internal/domain/token"
)

type TokenRepositoryPG struct {
	DB *sql.DB
}

func NewTokenRepositoryPG(db *sql.DB) *TokenRepositoryPG {
	return &TokenRepositoryPG{DB: db}
}

// ======================================================================
// TokenRepo facade for usecase.TokenRepo
// (Make TokenRepositoryPG satisfy the interface expected by TokenUsecase.)
// ======================================================================

// GetByID(ctx, id) (tokendom.Token, error)
// Here "id" from the usecase == mint_address in DB.
func (r *TokenRepositoryPG) GetByID(ctx context.Context, id string) (tokendom.Token, error) {
	return r.GetByMintAddress(ctx, id)
}

// Exists(ctx, id) (bool, error)
// Check existence by mint_address.
func (r *TokenRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `SELECT 1 FROM tokens WHERE mint_address = $1`
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

// Create(ctx, v tokendom.Token) (tokendom.Token, error)
// Insert using the passed token fields directly.
func (r *TokenRepositoryPG) Create(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	// We assume MintAddress, MintRequestID, Owner are all meaningful.
	// In your current schema, mint_address is NOT generated automatically,
	// so we use v.MintAddress as-is.
	const q = `
INSERT INTO tokens (mint_address, mint_request_id, owner)
VALUES ($1, $2, $3)
RETURNING mint_address, mint_request_id, owner
`
	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(v.MintAddress),
		strings.TrimSpace(v.MintRequestID),
		strings.TrimSpace(v.Owner),
	)
	t, err := scanToken(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return tokendom.Token{}, tokendom.ErrConflict
		}
		return tokendom.Token{}, err
	}
	return t, nil
}

// Save(ctx, v tokendom.Token) (tokendom.Token, error)
// Upsert-like: if token with MintAddress exists -> Update, else -> Create.
func (r *TokenRepositoryPG) Save(ctx context.Context, v tokendom.Token) (tokendom.Token, error) {
	mintAddress := strings.TrimSpace(v.MintAddress)
	if mintAddress == "" {
		// We cannot "upsert" without a key. In this domain, MintAddress is the key.
		// We'll treat this as a hard error because silently generating addresses
		// would break blockchain assumptions.
		return tokendom.Token{}, errors.New("mint address required for Save")
	}

	exists, err := r.Exists(ctx, mintAddress)
	if err != nil {
		return tokendom.Token{}, err
	}

	if !exists {
		// New row
		return r.Create(ctx, v)
	}

	// Row exists -> build UpdateTokenInput then call Update
	patch := tokendom.UpdateTokenInput{
		MintRequestID: func(s string) *string {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			x := strings.TrimSpace(s)
			return &x
		}(v.MintRequestID),
		Owner: func(s string) *string {
			if strings.TrimSpace(s) == "" {
				return nil
			}
			x := strings.TrimSpace(s)
			return &x
		}(v.Owner),
	}

	return r.Update(ctx, mintAddress, patch)
}

// Delete(ctx, id) error
// (Signature already matches usecase.TokenRepo.Delete; we just treat id as mint_address.)

// ======================================================================
// Lower-level / richer query methods
// (List, Count, Transfer, GetStats, etc. are still available.)
// ======================================================================

func (r *TokenRepositoryPG) GetByMintAddress(ctx context.Context, mintAddress string) (tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT mint_address, mint_request_id, owner
FROM tokens
WHERE mint_address = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(mintAddress))
	t, err := scanToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tokendom.Token{}, tokendom.ErrNotFound
		}
		return tokendom.Token{}, err
	}
	return t, nil
}

func (r *TokenRepositoryPG) List(ctx context.Context, filter tokendom.Filter, sort tokendom.Sort, page tokendom.Page) (tokendom.PageResult, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where, args := buildTokenWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}

	orderBy := buildTokenOrderBy(sort)
	if orderBy == "" {
		orderBy = "ORDER BY minted_at DESC, mint_address DESC"
	}

	pageNum, perPage, offset := dbcommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM tokens "+whereSQL, args...).Scan(&total); err != nil {
		return tokendom.PageResult{}, err
	}

	q := fmt.Sprintf(`
SELECT mint_address, mint_request_id, owner
FROM tokens
%s
%s
LIMIT $%d OFFSET $%d
`, whereSQL, orderBy, len(args)+1, len(args)+2)

	args = append(args, perPage, offset)
	rows, err := run.QueryContext(ctx, q, args...)
	if err != nil {
		return tokendom.PageResult{}, err
	}
	defer rows.Close()

	items := make([]tokendom.Token, 0, perPage)
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return tokendom.PageResult{}, err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return tokendom.PageResult{}, err
	}

	return tokendom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: dbcommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *TokenRepositoryPG) Count(ctx context.Context, filter tokendom.Filter) (int, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	where, args := buildTokenWhere(filter)
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = "WHERE " + strings.Join(where, " AND ")
	}
	var total int
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM tokens "+whereSQL, args...).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

func (r *TokenRepositoryPG) GetByOwner(ctx context.Context, owner string) ([]tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT mint_address, mint_request_id, owner
FROM tokens
WHERE owner = $1
ORDER BY minted_at DESC, mint_address DESC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(owner))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tokendom.Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *TokenRepositoryPG) GetByMintRequest(ctx context.Context, mintRequestID string) ([]tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
SELECT mint_address, mint_request_id, owner
FROM tokens
WHERE mint_request_id = $1
ORDER BY minted_at DESC, mint_address DESC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(mintRequestID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []tokendom.Token
	for rows.Next() {
		t, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Update is our lower-level UPDATE ... RETURNING.
// It's used by Save().
func (r *TokenRepositoryPG) Update(ctx context.Context, mintAddress string, in tokendom.UpdateTokenInput) (tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	if in.MintRequestID != nil {
		sets = append(sets, fmt.Sprintf("mint_request_id = $%d", i))
		args = append(args, strings.TrimSpace(*in.MintRequestID))
		i++
	}
	if in.Owner != nil {
		sets = append(sets, fmt.Sprintf("owner = $%d", i))
		args = append(args, strings.TrimSpace(*in.Owner))
		i++
		// bump last_transferred_at whenever owner changes
		sets = append(sets, "last_transferred_at = NOW()")
	}

	if len(sets) == 0 {
		return r.GetByMintAddress(ctx, mintAddress)
	}

	args = append(args, strings.TrimSpace(mintAddress))
	q := fmt.Sprintf(`
UPDATE tokens
SET %s
WHERE mint_address = $%d
RETURNING mint_address, mint_request_id, owner
`, strings.Join(sets, ", "), i)

	row := run.QueryRowContext(ctx, q, args...)
	t, err := scanToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tokendom.Token{}, tokendom.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return tokendom.Token{}, tokendom.ErrConflict
		}
		return tokendom.Token{}, err
	}
	return t, nil
}

func (r *TokenRepositoryPG) Delete(ctx context.Context, mintAddress string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM tokens WHERE mint_address = $1`, strings.TrimSpace(mintAddress))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return tokendom.ErrNotFound
	}
	return nil
}

func (r *TokenRepositoryPG) Transfer(ctx context.Context, mintAddress, newOwner string) (tokendom.Token, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	const q = `
UPDATE tokens
SET owner = $1, last_transferred_at = NOW()
WHERE mint_address = $2
RETURNING mint_address, mint_request_id, owner
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(newOwner), strings.TrimSpace(mintAddress))
	t, err := scanToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tokendom.Token{}, tokendom.ErrNotFound
		}
		return tokendom.Token{}, err
	}
	return t, nil
}

func (r *TokenRepositoryPG) Burn(ctx context.Context, mintAddress string) error {
	return r.Delete(ctx, mintAddress)
}

func (r *TokenRepositoryPG) GetStats(ctx context.Context) (tokendom.TokenStats, error) {
	run := dbcommon.GetRunner(ctx, r.DB)
	var stats tokendom.TokenStats

	// totals
	if err := run.QueryRowContext(ctx, `SELECT COUNT(*) FROM tokens`).Scan(&stats.TotalTokens); err != nil {
		return tokendom.TokenStats{}, err
	}
	if err := run.QueryRowContext(ctx, `SELECT COUNT(DISTINCT owner) FROM tokens`).Scan(&stats.UniqueOwners); err != nil {
		return tokendom.TokenStats{}, err
	}
	if err := run.QueryRowContext(ctx, `SELECT COUNT(DISTINCT mint_request_id) FROM tokens`).Scan(&stats.UniqueMintRequests); err != nil {
		return tokendom.TokenStats{}, err
	}

	// by owner
	stats.ByOwner = map[string]int{}
	rows, err := run.QueryContext(ctx, `SELECT owner, COUNT(*) FROM tokens GROUP BY owner`)
	if err != nil {
		return tokendom.TokenStats{}, err
	}
	for rows.Next() {
		var owner string
		var cnt int
		if err := rows.Scan(&owner, &cnt); err != nil {
			rows.Close()
			return tokendom.TokenStats{}, err
		}
		stats.ByOwner[strings.TrimSpace(owner)] = cnt
	}
	_ = rows.Close()

	// by mint_request
	stats.ByMintRequest = map[string]int{}
	rows, err = run.QueryContext(ctx, `SELECT mint_request_id, COUNT(*) FROM tokens GROUP BY mint_request_id`)
	if err != nil {
		return tokendom.TokenStats{}, err
	}
	for rows.Next() {
		var mr string
		var cnt int
		if err := rows.Scan(&mr, &cnt); err != nil {
			rows.Close()
			return tokendom.TokenStats{}, err
		}
		stats.ByMintRequest[strings.TrimSpace(mr)] = cnt
	}
	_ = rows.Close()

	// top owners
	rows, err = run.QueryContext(ctx, `
SELECT owner, COUNT(*) AS c
FROM tokens
GROUP BY owner
ORDER BY c DESC, owner ASC
LIMIT 10`)
	if err != nil {
		return tokendom.TokenStats{}, err
	}
	for rows.Next() {
		var owner string
		var cnt int
		if err := rows.Scan(&owner, &cnt); err != nil {
			rows.Close()
			return tokendom.TokenStats{}, err
		}
		stats.TopOwners = append(stats.TopOwners, struct {
			Owner string
			Count int
		}{
			Owner: strings.TrimSpace(owner),
			Count: cnt,
		})
	}
	_ = rows.Close()

	// top mint requests
	rows, err = run.QueryContext(ctx, `
SELECT mint_request_id, COUNT(*) AS c
FROM tokens
GROUP BY mint_request_id
ORDER BY c DESC, mint_request_id ASC
LIMIT 10`)
	if err != nil {
		return tokendom.TokenStats{}, err
	}
	for rows.Next() {
		var mr string
		var cnt int
		if err := rows.Scan(&mr, &cnt); err != nil {
			rows.Close()
			return tokendom.TokenStats{}, err
		}
		stats.TopMintRequests = append(stats.TopMintRequests, struct {
			MintRequestID string
			Count         int
		}{
			MintRequestID: strings.TrimSpace(mr),
			Count:         cnt,
		})
	}
	_ = rows.Close()

	return stats, nil
}

func (r *TokenRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *TokenRepositoryPG) Reset(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	_, err := run.ExecContext(ctx, `DELETE FROM tokens`)
	return err
}

// ======================================================================
// Helpers
// ======================================================================

func scanToken(s dbcommon.RowScanner) (tokendom.Token, error) {
	var (
		mintAddress   string
		mintRequestID string
		owner         string
	)
	if err := s.Scan(&mintAddress, &mintRequestID, &owner); err != nil {
		return tokendom.Token{}, err
	}
	return tokendom.Token{
		MintAddress:   strings.TrimSpace(mintAddress),
		MintRequestID: strings.TrimSpace(mintRequestID),
		Owner:         strings.TrimSpace(owner),
	}, nil
}

func buildTokenWhere(f tokendom.Filter) ([]string, []any) {
	where := []string{}
	args := []any{}

	addEqList := func(col string, vals []string) {
		clean := make([]string, 0, len(vals))
		for _, v := range vals {
			if v = strings.TrimSpace(v); v != "" {
				clean = append(clean, v)
			}
		}
		if len(clean) == 0 {
			return
		}
		base := len(args)
		ph := make([]string, len(clean))
		for i, v := range clean {
			args = append(args, v)
			ph[i] = fmt.Sprintf("$%d", base+i+1)
		}
		where = append(where, fmt.Sprintf("%s IN (%s)", col, strings.Join(ph, ",")))
	}

	addEqList("mint_address", f.MintAddresses)
	addEqList("mint_request_id", f.MintRequestIDs)
	addEqList("owner", f.Owners)

	if v := strings.TrimSpace(f.MintAddressLike); v != "" {
		where = append(where, fmt.Sprintf("mint_address ILIKE $%d", len(args)+1))
		args = append(args, "%"+v+"%")
	}

	// Time ranges
	if f.MintedFrom != nil {
		where = append(where, fmt.Sprintf("minted_at >= $%d", len(args)+1))
		args = append(args, f.MintedFrom.UTC())
	}
	if f.MintedTo != nil {
		where = append(where, fmt.Sprintf("minted_at < $%d", len(args)+1))
		args = append(args, f.MintedTo.UTC())
	}
	if f.LastTransferredFrom != nil {
		where = append(where, fmt.Sprintf("last_transferred_at >= $%d", len(args)+1))
		args = append(args, f.LastTransferredFrom.UTC())
	}
	if f.LastTransferredTo != nil {
		where = append(where, fmt.Sprintf("last_transferred_at < $%d", len(args)+1))
		args = append(args, f.LastTransferredTo.UTC())
	}

	return where, args
}

func buildTokenOrderBy(s tokendom.Sort) string {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	switch col {
	case "mintaddress", "mint_address":
		col = "mint_address"
	case "mintrequestid", "mint_request_id":
		col = "mint_request_id"
	case "owner":
		col = "owner"
	case "mintedat", "minted_at":
		col = "minted_at"
	case "lasttransferredat", "last_transferred_at":
		col = "last_transferred_at"
	default:
		return ""
	}
	dir := strings.ToUpper(strings.TrimSpace(string(s.Order)))
	if dir != "ASC" && dir != "DESC" {
		dir = "DESC"
	}
	return fmt.Sprintf("ORDER BY %s %s, mint_address %s", col, dir, dir)
}
