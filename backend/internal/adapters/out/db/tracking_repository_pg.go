package db

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

    dbcommon "narratives/internal/adapters/out/db/common"
    trdom "narratives/internal/domain/tracking"
)

type TrackingRepositoryPG struct {
    DB *sql.DB
}

func NewTrackingRepositoryPG(db *sql.DB) *TrackingRepositoryPG {
    return &TrackingRepositoryPG{DB: db}
}

// GetAllTrackings returns all trackings ordered by updated_at desc, id desc.
func (r *TrackingRepositoryPG) GetAllTrackings(ctx context.Context) ([]*trdom.Tracking, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
FROM trackings
ORDER BY updated_at DESC, id DESC`
    rows, err := run.QueryContext(ctx, q)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []*trdom.Tracking
    for rows.Next() {
        t, err := scanTracking(rows)
        if err != nil {
            return nil, err
        }
        tt := t
        out = append(out, &tt)
    }
    return out, rows.Err()
}

func (r *TrackingRepositoryPG) GetTrackingByID(ctx context.Context, id string) (*trdom.Tracking, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
FROM trackings
WHERE id = $1
LIMIT 1`
    row := run.QueryRowContext(ctx, q, strings.TrimSpace(id))
    t, err := scanTracking(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, sql.ErrNoRows
        }
        return nil, err
    }
    return &t, nil
}

func (r *TrackingRepositoryPG) GetTrackingsByOrderID(ctx context.Context, orderID string) ([]*trdom.Tracking, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
SELECT
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
FROM trackings
WHERE order_id = $1
ORDER BY updated_at DESC, id DESC`
    rows, err := run.QueryContext(ctx, q, strings.TrimSpace(orderID))
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []*trdom.Tracking
    for rows.Next() {
        t, err := scanTracking(rows)
        if err != nil {
            return nil, err
        }
        tt := t
        out = append(out, &tt)
    }
    return out, rows.Err()
}

func (r *TrackingRepositoryPG) CreateTracking(ctx context.Context, in trdom.CreateTrackingInput) (*trdom.Tracking, error) {
    run := dbcommon.GetRunner(ctx, r.DB)
    const q = `
INSERT INTO trackings (
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
) VALUES (
  gen_random_uuid()::text, $1, $2, $3, $4, NOW(), NOW()
)
RETURNING
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
`
    row := run.QueryRowContext(ctx, q,
        strings.TrimSpace(in.OrderID),
        strings.TrimSpace(in.TrackingNumber),
        strings.TrimSpace(in.Carrier),
        dbcommon.NullableTrim(in.SpecialInstructions),
    )
    t, err := scanTracking(row)
    if err != nil {
        return nil, err
    }
    return &t, nil
}

func (r *TrackingRepositoryPG) UpdateTracking(ctx context.Context, id string, in trdom.UpdateTrackingInput) (*trdom.Tracking, error) {
    run := dbcommon.GetRunner(ctx, r.DB)

    sets := []string{}
    args := []any{}
    i := 1

    if in.Carrier != nil {
        sets = append(sets, fmt.Sprintf("carrier = $%d", i))
        args = append(args, strings.TrimSpace(*in.Carrier))
        i++
    }
    if in.TrackingNumber != nil {
        sets = append(sets, fmt.Sprintf("tracking_number = $%d", i))
        args = append(args, strings.TrimSpace(*in.TrackingNumber))
        i++
    }
    if in.SpecialInstructions != nil {
        v := strings.TrimSpace(*in.SpecialInstructions)
        if v == "" {
            sets = append(sets, "special_instructions = NULL")
        } else {
            sets = append(sets, fmt.Sprintf("special_instructions = $%d", i))
            args = append(args, v)
            i++
        }
    }
    // Always bump updated_at
    sets = append(sets, "updated_at = NOW()")

    if len(sets) == 0 {
        return r.GetTrackingByID(ctx, id)
    }

    args = append(args, strings.TrimSpace(id))
    q := fmt.Sprintf(`
UPDATE trackings
SET %s
WHERE id = $%d
RETURNING
  id, order_id, tracking_number, carrier, special_instructions, created_at, updated_at
`, strings.Join(sets, ", "), i)

    row := run.QueryRowContext(ctx, q, args...)
    t, err := scanTracking(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, sql.ErrNoRows
        }
        return nil, err
    }
    return &t, nil
}

func (r *TrackingRepositoryPG) DeleteTracking(ctx context.Context, id string) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    res, err := run.ExecContext(ctx, `DELETE FROM trackings WHERE id = $1`, strings.TrimSpace(id))
    if err != nil {
        return err
    }
    if n, _ := res.RowsAffected(); n == 0 {
        return sql.ErrNoRows
    }
    return nil
}

func (r *TrackingRepositoryPG) ResetTrackings(ctx context.Context) error {
    run := dbcommon.GetRunner(ctx, r.DB)
    _, err := run.ExecContext(ctx, `DELETE FROM trackings`)
    return err
}

func (r *TrackingRepositoryPG) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
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

// Helpers

func scanTracking(s dbcommon.RowScanner) (trdom.Tracking, error) {
    var (
        id, orderID, trackingNumber, carrier string
        specialNS                            sql.NullString
        createdAt, updatedAt                 time.Time
    )
    if err := s.Scan(
        &id, &orderID, &trackingNumber, &carrier, &specialNS, &createdAt, &updatedAt,
    ); err != nil {
        return trdom.Tracking{}, err
    }
    var special *string
    if specialNS.Valid {
        v := strings.TrimSpace(specialNS.String)
        if v != "" {
            special = &v
        }
    }
    return trdom.Tracking{
        ID:                  strings.TrimSpace(id),
        OrderID:             strings.TrimSpace(orderID),
        TrackingNumber:      strings.TrimSpace(trackingNumber),
        Carrier:             strings.TrimSpace(carrier),
        SpecialInstructions: special,
        CreatedAt:           createdAt.UTC(),
        UpdatedAt:           updatedAt.UTC(),
    }, nil
}