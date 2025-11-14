// frontend/list/src/domain/entity/sale.ts
// Mirrors backend/internal/domain/sale/entity.go
// Source of truth for Sale / SalePrice domain model (frontend side)

//
// Types
//

/**
 * 個別モデル番号ごとの販売価格
 * - backend/internal/domain/sale/entity.go: SalePrice に対応
 */
export interface SalePrice {
  /** モデル番号（例: "model_001"） */
  modelNumber: string;
  /** 販売価格（JPY） */
  price: number;
}

/**
 * 販売情報 (Sale)
 * - backend/internal/domain/sale/entity.go: Sale に対応
 */
export interface Sale {
  /** Sale ID（例: "sale_001"） */
  id: string;
  /** 対象 List の ID */
  listId: string;
  /** 割引情報 Discount の ID（存在する場合） */
  discountId?: string | null;
  /** モデル番号ごとの販売価格一覧 */
  prices: SalePrice[];
}

//
// Policy (frontend-side constants; keep in sync with backend)
//

/** 最低価格（0 以上） */
export const SALE_MIN_PRICE = 0;
/** 最高価格（0 は無効、backend は 10_000_000） */
export const SALE_MAX_PRICE = 10_000_000;
/** 少なくとも 1 件の価格が必要 */
export const SALE_MIN_PRICES_REQUIRED = 1;
/** modelNumber の許可パターン（backend の ModelNumberRe に対応） */
export const SALE_MODEL_NUMBER_REGEX = /^[A-Za-z0-9._-]{1,64}$/;

//
// Validation helpers
//

/**
 * 単一の SalePrice がドメインルールに従っているか簡易チェック
 * （必須: modelNumber / price 範囲）
 */
export function isValidSalePrice(p: SalePrice): boolean {
  if (!p) return false;
  const model = (p.modelNumber ?? "").trim();
  if (!model) return false;
  if (SALE_MODEL_NUMBER_REGEX && !SALE_MODEL_NUMBER_REGEX.test(model)) {
    return false;
  }
  if (!priceAllowed(p.price)) return false;
  return true;
}

/**
 * Sale 全体の検証
 * - 必須フィールド非空
 * - discountId があれば非空文字列
 * - prices がルールに従う & modelNumber が一意
 */
export function isValidSale(sale: Sale): boolean {
  if (!sale) return false;

  if (!sale.id?.trim()) return false;
  if (!sale.listId?.trim()) return false;

  if (sale.discountId != null && sale.discountId !== undefined) {
    if (!`${sale.discountId}`.trim()) return false;
  }

  if (!Array.isArray(sale.prices)) return false;
  if (sale.prices.length < SALE_MIN_PRICES_REQUIRED) return false;

  const seen = new Set<string>();
  for (const p of sale.prices) {
    if (!isValidSalePrice(p)) return false;
    const key = p.modelNumber.trim();
    if (seen.has(key)) return false; // 重複禁止
    seen.add(key);
  }

  return true;
}

/**
 * backend の aggregatePrices 相当:
 * - modelNumber をトリム
 * - 無効エントリを除外
 * - 同一 modelNumber が複数ある場合は「最後の有効値」を採用
 */
export function aggregateSalePrices(prices: SalePrice[]): SalePrice[] {
  if (!Array.isArray(prices)) return [];

  const tmp = new Map<string, number>();
  const order: string[] = [];

  for (const p of prices) {
    const model = (p.modelNumber ?? "").trim();
    if (!model) continue;
    if (!priceAllowed(p.price)) continue;
    if (!SALE_MODEL_NUMBER_REGEX.test(model)) continue;

    if (!tmp.has(model)) {
      order.push(model);
    }
    // 「最後の有効な値」が勝つ
    tmp.set(model, p.price);
  }

  return order.map((m) => ({
    modelNumber: m,
    price: tmp.get(m)!,
  }));
}

/**
 * 金額がポリシー範囲内かどうか
 */
export function priceAllowed(v: number): boolean {
  if (!Number.isFinite(v)) return false;
  if (v < SALE_MIN_PRICE) return false;
  if (SALE_MAX_PRICE > 0 && v > SALE_MAX_PRICE) return false;
  return true;
}
