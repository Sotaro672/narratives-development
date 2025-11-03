package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dbcommon "narratives/internal/adapters/out/db/common"
	tod "narratives/internal/domain/tokenOperation"
)

// TokenOperationRepositoryPG implements usecase.TokenOperationRepo using PostgreSQL.
type TokenOperationRepositoryPG struct {
	DB *sql.DB
}

func NewTokenOperationRepositoryPG(db *sql.DB) *TokenOperationRepositoryPG {
	return &TokenOperationRepositoryPG{DB: db}
}

// =====================================================
// usecase.TokenOperationRepo 準拠メソッド
// =====================================================

// GetByID returns a minimal domain.TokenOperation row.
func (r *TokenOperationRepositoryPG) GetByID(ctx context.Context, id string) (tod.TokenOperation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  id,
  token_blueprint_id,
  assignee_id
FROM token_operations
WHERE id = $1
LIMIT 1
`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	op, err := scanTokenOperationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tod.TokenOperation{}, tod.ErrNotFound
		}
		return tod.TokenOperation{}, err
	}
	return op, nil
}

// Exists checks if a row with the given ID exists.
func (r *TokenOperationRepositoryPG) Exists(ctx context.Context, id string) (bool, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `SELECT 1 FROM token_operations WHERE id = $1 LIMIT 1`
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

// Create inserts a new token_operations row using the given domain entity v.
// domain.TokenOperation には Name / Status / UpdatedBy 等のフィールドが無いので、
// それらはDB側でデフォルト値を入れる (name=”, status='operational', updated_by=”) 方針にする。
func (r *TokenOperationRepositoryPG) Create(ctx context.Context, v tod.TokenOperation) (tod.TokenOperation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	useProvidedID := strings.TrimSpace(v.ID) != ""

	// パターン1: IDを呼び出し側が用意している場合
	const qWithID = `
INSERT INTO token_operations (
  id,
  token_blueprint_id,
  assignee_id,
  name,
  status,
  updated_at,
  updated_by
) VALUES (
  $1,
  $2,
  $3,
  '',               -- name
  'operational',    -- status
  NOW(),            -- updated_at
  ''                -- updated_by
)
RETURNING
  id,
  token_blueprint_id,
  assignee_id
`

	// パターン2: IDはDB側で生成する場合
	const qNoID = `
INSERT INTO token_operations (
  id,
  token_blueprint_id,
  assignee_id,
  name,
  status,
  updated_at,
  updated_by
) VALUES (
  gen_random_uuid()::text,
  $1,
  $2,
  '',               -- name
  'operational',    -- status
  NOW(),            -- updated_at
  ''                -- updated_by
)
RETURNING
  id,
  token_blueprint_id,
  assignee_id
`

	var row *sql.Row
	if useProvidedID {
		row = run.QueryRowContext(ctx, qWithID,
			strings.TrimSpace(v.ID),
			strings.TrimSpace(v.TokenBlueprintID),
			strings.TrimSpace(v.AssigneeID),
		)
	} else {
		row = run.QueryRowContext(ctx, qNoID,
			strings.TrimSpace(v.TokenBlueprintID),
			strings.TrimSpace(v.AssigneeID),
		)
	}

	op, err := scanTokenOperationRow(row)
	if err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return tod.TokenOperation{}, tod.ErrConflict
		}
		return tod.TokenOperation{}, err
	}
	return op, nil
}

// Save updates token_blueprint_id / assignee_id for the given row ID.
// 他の列(name/status/updated_byなど)はdomain.TokenOperation側に存在しないので、ここでは触らない。
// updated_atだけはNOW()で更新しておく。
func (r *TokenOperationRepositoryPG) Save(ctx context.Context, v tod.TokenOperation) (tod.TokenOperation, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
UPDATE token_operations
SET
  token_blueprint_id = $2,
  assignee_id        = $3,
  updated_at         = NOW()
WHERE id = $1
RETURNING
  id,
  token_blueprint_id,
  assignee_id
`

	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(v.ID),
		strings.TrimSpace(v.TokenBlueprintID),
		strings.TrimSpace(v.AssigneeID),
	)

	op, err := scanTokenOperationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return tod.TokenOperation{}, tod.ErrNotFound
		}
		if dbcommon.IsUniqueViolation(err) {
			return tod.TokenOperation{}, tod.ErrConflict
		}
		return tod.TokenOperation{}, err
	}
	return op, nil
}

// Delete removes a row by ID.
func (r *TokenOperationRepositoryPG) Delete(ctx context.Context, id string) error {
	run := dbcommon.GetRunner(ctx, r.DB)

	res, err := run.ExecContext(ctx, `DELETE FROM token_operations WHERE id = $1`, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return tod.ErrNotFound
	}
	return nil
}

// =====================================================
// 追加の読み取り系 (拡張ビューなど) 既存のロジック
// =====================================================

func (r *TokenOperationRepositoryPG) GetOperationalTokens(ctx context.Context) ([]*tod.OperationalToken, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  o.id,
  o.token_blueprint_id,
  o.assignee_id,
  COALESCE(tb.name, '')     AS token_name,
  COALESCE(tb.symbol, '')   AS symbol,
  COALESCE(tb.brand_id, '') AS brand_id,
  COALESCE(m.name, '')      AS assignee_name,
  COALESCE(b.name, '')      AS brand_name,
  COALESCE(o.name, '')      AS op_name,
  COALESCE(o.status, '')    AS status,
  COALESCE(o.updated_at, NOW()) AS updated_at,
  COALESCE(o.updated_by, '')    AS updated_by
FROM token_operations o
LEFT JOIN token_blueprints tb ON tb.id = o.token_blueprint_id
LEFT JOIN brands b            ON b.id  = tb.brand_id
LEFT JOIN members m           ON m.id  = o.assignee_id
ORDER BY o.updated_at DESC, o.id DESC`

	rows, err := run.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tod.OperationalToken
	for rows.Next() {
		op, err := scanOperationalToken(rows)
		if err != nil {
			return nil, err
		}
		o := op
		out = append(out, &o)
	}
	return out, rows.Err()
}

func (r *TokenOperationRepositoryPG) GetOperationalTokenByID(ctx context.Context, id string) (*tod.OperationalToken, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT
  o.id,
  o.token_blueprint_id,
  o.assignee_id,
  COALESCE(tb.name, '')     AS token_name,
  COALESCE(tb.symbol, '')   AS symbol,
  COALESCE(tb.brand_id, '') AS brand_id,
  COALESCE(m.name, '')      AS assignee_name,
  COALESCE(b.name, '')      AS brand_name,
  COALESCE(o.name, '')      AS op_name,
  COALESCE(o.status, '')    AS status,
  COALESCE(o.updated_at, NOW()) AS updated_at,
  COALESCE(o.updated_by, '')    AS updated_by
FROM token_operations o
LEFT JOIN token_blueprints tb ON tb.id = o.token_blueprint_id
LEFT JOIN brands b            ON b.id  = tb.brand_id
LEFT JOIN members m           ON m.id  = o.assignee_id
WHERE o.id = $1
LIMIT 1`

	row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
	op, err := scanOperationalToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
		}
		return nil, err
	}
	return &op, nil
}

// こちらは OperationalToken (拡張ビュー) 用の create。
// usecase.TokenOperationRepo.Create とは別の用途なので残す。
func (r *TokenOperationRepositoryPG) CreateOperationalToken(ctx context.Context, in tod.CreateOperationalTokenData) (*tod.OperationalToken, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO token_operations (
  id, token_blueprint_id, assignee_id, name, status, updated_at, updated_by
) VALUES (
  gen_random_uuid()::text, $1, $2, '', 'operational', NOW(), ''
)
RETURNING id`

	var id string
	if err := run.QueryRowContext(ctx, q,
		strings.TrimSpace(in.TokenBlueprintID),
		strings.TrimSpace(in.AssigneeID),
	).Scan(&id); err != nil {
		if dbcommon.IsUniqueViolation(err) {
			return nil, tod.ErrConflict
		}
		return nil, err
	}
	return r.GetOperationalTokenByID(ctx, id)
}

// こちらは OperationalToken (拡張ビュー) 更新。
// usecase.TokenOperationRepo.Save とは別の用途。
func (r *TokenOperationRepositoryPG) UpdateOperationalToken(ctx context.Context, id string, in tod.UpdateOperationalTokenData) (*tod.OperationalToken, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	sets := []string{}
	args := []any{}
	i := 1

	if in.AssigneeID != nil {
		sets = append(sets, fmt.Sprintf("assignee_id = $%d", i))
		args = append(args, strings.TrimSpace(*in.AssigneeID))
		i++
	}
	if in.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", i))
		args = append(args, strings.TrimSpace(*in.Name))
		i++
	}
	if in.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", i))
		args = append(args, strings.TrimSpace(*in.Status))
		i++
	}
	if v := strings.TrimSpace(in.UpdatedBy); v != "" {
		sets = append(sets, fmt.Sprintf("updated_by = $%d", i))
		args = append(args, v)
		i++
	}
	// 業務的な更新なので updated_at も更新する
	sets = append(sets, "updated_at = NOW()")

	if len(sets) == 0 {
		return r.GetOperationalTokenByID(ctx, id)
	}

	args = append(args, strings.TrimSpace(id))
	q := fmt.Sprintf(`
UPDATE token_operations
SET %s
WHERE id = $%d
RETURNING id`, strings.Join(sets, ", "), i)

	var retID string
	if err := run.QueryRowContext(ctx, q, args...).Scan(&retID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: operational token not found", tod.ErrNotFound)
		}
		if dbcommon.IsUniqueViolation(err) {
			return nil, tod.ErrConflict
		}
		return nil, err
	}
	return r.GetOperationalTokenByID(ctx, retID)
}

// Holders / History / Contents / ProductDetail はそのまま

func (r *TokenOperationRepositoryPG) GetHoldersByTokenID(ctx context.Context, params tod.HolderSearchParams) (holders []*tod.Holder, total int, err error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where := []string{"token_id = $1"}
	args := []any{strings.TrimSpace(params.TokenID)}
	if q := strings.TrimSpace(params.Query); q != "" {
		where = append(where, fmt.Sprintf("(wallet_address ILIKE $%d)", len(args)+1))
		args = append(args, "%"+q+"%")
	}
	whereSQL := "WHERE " + strings.Join(where, " AND ")

	// Count
	if err := run.QueryRowContext(ctx, "SELECT COUNT(*) FROM token_holders "+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Paging
	_, per, off := dbcommon.NormalizePage(params.Offset/params.Limit+1, params.Limit, 50, 200)

	args2 := append(append([]any{}, args...), per, off)
	q := fmt.Sprintf(`
SELECT id, token_id, wallet_address, balance, updated_at
FROM token_holders
%s
ORDER BY updated_at DESC, id DESC
LIMIT $%d OFFSET $%d`, whereSQL, len(args)+1, len(args)+2)

	rows, err := run.QueryContext(ctx, q, args2...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*tod.Holder
	for rows.Next() {
		h, err := scanHolder(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, h)
	}
	return out, total, rows.Err()
}

func (r *TokenOperationRepositoryPG) GetTokenUpdateHistory(ctx context.Context, params tod.TokenUpdateHistorySearchParams) ([]*tod.TokenUpdateHistory, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	where := "WHERE token_id = $1"
	args := []any{strings.TrimSpace(params.TokenID)}

	q := `
SELECT id, token_id, event, assignee_id, note, created_at
FROM token_update_history
` + where + `
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3`

	limit := params.Offset + params.Limit
	if params.Limit <= 0 {
		limit = 100
	}
	offset := params.Offset
	rows, err := run.QueryContext(ctx, q, append(args, limit, offset)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tod.TokenUpdateHistory
	for rows.Next() {
		h, err := scanUpdateHistory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *TokenOperationRepositoryPG) GetTokenContents(ctx context.Context, tokenID string) ([]*tod.TokenContent, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT id, token_id, type, url, description, published_by, created_at
FROM token_operation_contents
WHERE token_id = $1
ORDER BY created_at DESC, id DESC`
	rows, err := run.QueryContext(ctx, q, strings.TrimSpace(tokenID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*tod.TokenContent
	for rows.Next() {
		tc, err := scanOperationContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

func (r *TokenOperationRepositoryPG) AddTokenContent(ctx context.Context, tokenID string, typ, url, description, publishedBy string) (*tod.TokenContent, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
INSERT INTO token_operation_contents (
  id, token_id, type, url, description, published_by, created_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, $5, NOW()
)
RETURNING id, token_id, type, url, description, published_by, created_at`

	row := run.QueryRowContext(ctx, q,
		strings.TrimSpace(tokenID),
		strings.TrimSpace(typ),
		strings.TrimSpace(url),
		strings.TrimSpace(description),
		strings.TrimSpace(publishedBy),
	)
	tc, err := scanOperationContent(row)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

func (r *TokenOperationRepositoryPG) DeleteTokenContent(ctx context.Context, contentID string) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	res, err := run.ExecContext(ctx, `DELETE FROM token_operation_contents WHERE id = $1`, strings.TrimSpace(contentID))
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: token content not found", tod.ErrNotFound)
	}
	return nil
}

func (r *TokenOperationRepositoryPG) GetProductDetailByID(ctx context.Context, productID string) (*tod.ProductDetail, error) {
	run := dbcommon.GetRunner(ctx, r.DB)

	const q = `
SELECT id, name, description
FROM product_details
WHERE id = $1`
	row := run.QueryRowContext(ctx, q, strings.TrimSpace(productID))
	pd, err := scanProductDetail(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: product detail not found", tod.ErrNotFound)
		}
		return nil, err
	}
	return &pd, nil
}

func (r *TokenOperationRepositoryPG) ResetTokenOperations(ctx context.Context) error {
	run := dbcommon.GetRunner(ctx, r.DB)
	// Best-effort order; adjust if FK constraints exist
	if _, err := run.ExecContext(ctx, `DELETE FROM token_operation_contents`); err != nil {
		return err
	}
	if _, err := run.ExecContext(ctx, `DELETE FROM token_update_history`); err != nil {
		return err
	}
	_, err := run.ExecContext(ctx, `DELETE FROM token_operations`)
	return err
}

// WithTx は他のRepositoryと同じトランザクションヘルパー
func (r *TokenOperationRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

// =====================================================
// Scanners
// =====================================================

// scanTokenOperationRow: 素の token_operations 行を domain.TokenOperation に落とす
// domain.TokenOperation には ID / TokenBlueprintID / AssigneeID しか無い前提で組む
func scanTokenOperationRow(s dbcommon.RowScanner) (tod.TokenOperation, error) {
	var (
		id   string
		tbid string
		aid  string
	)
	if err := s.Scan(
		&id,
		&tbid,
		&aid,
	); err != nil {
		return tod.TokenOperation{}, err
	}
	return tod.TokenOperation{
		ID:               strings.TrimSpace(id),
		TokenBlueprintID: strings.TrimSpace(tbid),
		AssigneeID:       strings.TrimSpace(aid),
	}, nil
}

// scanOperationalToken: JOIN済みビューを OperationalToken に落とす（既存のまま）
func scanOperationalToken(s dbcommon.RowScanner) (tod.OperationalToken, error) {
	var (
		id, tbid, aid              string
		tokenName, symbol, brandID string
		assigneeName, brandName    string
		name, status, updatedBy    string
		updatedAt                  time.Time
	)
	if err := s.Scan(
		&id,
		&tbid,
		&aid,
		&tokenName,
		&symbol,
		&brandID,
		&assigneeName,
		&brandName,
		&name,
		&status,
		&updatedAt,
		&updatedBy,
	); err != nil {
		return tod.OperationalToken{}, err
	}
	return tod.OperationalToken{
		ID:               strings.TrimSpace(id),
		TokenBlueprintID: strings.TrimSpace(tbid),
		AssigneeID:       strings.TrimSpace(aid),

		TokenName:    strings.TrimSpace(tokenName),
		Symbol:       strings.TrimSpace(symbol),
		BrandID:      strings.TrimSpace(brandID),
		AssigneeName: strings.TrimSpace(assigneeName),
		BrandName:    strings.TrimSpace(brandName),

		Name:      strings.TrimSpace(name),
		Status:    strings.TrimSpace(status),
		UpdatedAt: updatedAt.UTC(),
		UpdatedBy: strings.TrimSpace(updatedBy),
	}, nil
}

func scanHolder(s dbcommon.RowScanner) (*tod.Holder, error) {
	var (
		id, tokenID, wallet, balance string
		updatedAt                    time.Time
	)
	if err := s.Scan(&id, &tokenID, &wallet, &balance, &updatedAt); err != nil {
		return nil, err
	}
	return &tod.Holder{
		ID:            strings.TrimSpace(id),
		TokenID:       strings.TrimSpace(tokenID),
		WalletAddress: strings.TrimSpace(wallet),
		Balance:       strings.TrimSpace(balance),
		UpdatedAt:     updatedAt.UTC(),
	}, nil
}

func scanUpdateHistory(s dbcommon.RowScanner) (*tod.TokenUpdateHistory, error) {
	var (
		id, tokenID, event, assigneeID, note string
		createdAt                            time.Time
	)
	if err := s.Scan(&id, &tokenID, &event, &assigneeID, &note, &createdAt); err != nil {
		return nil, err
	}
	return &tod.TokenUpdateHistory{
		ID:         strings.TrimSpace(id),
		TokenID:    strings.TrimSpace(tokenID),
		Event:      strings.TrimSpace(event),
		AssigneeID: strings.TrimSpace(assigneeID),
		Note:       strings.TrimSpace(note),
		CreatedAt:  createdAt.UTC(),
	}, nil
}

func scanOperationContent(s dbcommon.RowScanner) (*tod.TokenContent, error) {
	var (
		id, tokenID, typ, url, desc, publishedBy string
		createdAt                                time.Time
	)
	if err := s.Scan(&id, &tokenID, &typ, &url, &desc, &publishedBy, &createdAt); err != nil {
		return nil, err
	}
	return &tod.TokenContent{
		ID:          strings.TrimSpace(id),
		TokenID:     strings.TrimSpace(tokenID),
		Type:        strings.TrimSpace(typ),
		URL:         strings.TrimSpace(url),
		Description: strings.TrimSpace(desc),
		PublishedBy: strings.TrimSpace(publishedBy),
		CreatedAt:   createdAt.UTC(),
	}, nil
}

func scanProductDetail(s dbcommon.RowScanner) (tod.ProductDetail, error) {
	var id, name, desc string
	if err := s.Scan(&id, &name, &desc); err != nil {
		return tod.ProductDetail{}, err
	}
	return tod.ProductDetail{
		ID:          strings.TrimSpace(id),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(desc),
	}, nil
}
