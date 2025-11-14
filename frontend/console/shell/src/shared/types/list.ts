// frontend/shell/src/shared/types/list.ts

/**
 * ListStatus
 * backend/internal/domain/list/entity.go の ListStatus に対応。
 *
 * - "listing"   : 掲載中
 * - "suspended" : 一時停止
 * - "deleted"   : 削除済み
 */
export type ListStatus = "listing" | "suspended" | "deleted";

/** ListStatus の妥当性チェック */
export function isValidListStatus(s: string): s is ListStatus {
  return s === "listing" || s === "suspended" || s === "deleted";
}

/**
 * ListPrice
 * backend/internal/domain/list/entity.go の ListPrice に対応。
 *
 * - modelNumber ごとに price (JPY) を定義
 */
export interface ListPrice {
  modelNumber: string;
  price: number; // JPY
}

/**
 * List
 * backend/internal/domain/list/entity.go の List に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を利用
 * - updatedBy/updatedAt/deletedAt/deletedBy は任意
 * - imageId は ListImage.id を想定（必須）
 */
export interface List {
  id: string;
  inventoryId: string;
  status: ListStatus;
  assigneeId: string;
  imageId: string;
  description: string;
  prices: ListPrice[];

  createdBy: string;
  createdAt: string;

  updatedBy?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/* =========================================================
 * Policy / Constants (Go 側と整合)
 * =======================================================*/

export const MAX_DESCRIPTION_LENGTH = 2000;
export const MIN_PRICE = 0;
export const MAX_PRICE = 10_000_000;

/* =========================================================
 * Validation helpers
 * =======================================================*/

/** 簡易な日時文字列チェック（ISO8601/Date.parse ベース） */
export function isValidDateTimeString(
  value: string | null | undefined,
): boolean {
  if (!value) return false;
  const v = value.trim();
  if (!v) return false;
  const t = Date.parse(v);
  return !Number.isNaN(t);
}

/** a <= b の順序であれば true */
export function isDateTimeOrderValid(
  a: string | null | undefined,
  b: string | null | undefined,
): boolean {
  if (!a || !b) return false;
  const ta = Date.parse(a);
  const tb = Date.parse(b);
  if (Number.isNaN(ta) || Number.isNaN(tb)) return false;
  return ta <= tb;
}

/** ListPrice 単体の妥当性チェック */
export function validateListPrice(p: ListPrice): string[] {
  const errors: string[] = [];

  if (!p.modelNumber?.trim()) {
    errors.push("modelNumber is required");
  }

  if (p.price == null || Number.isNaN(p.price)) {
    errors.push("price is required");
  } else if (p.price < MIN_PRICE || p.price > MAX_PRICE) {
    errors.push(
      `price must be between ${MIN_PRICE} and ${MAX_PRICE}`,
    );
  }

  return errors;
}

/** ListPrice 配列全体の妥当性チェック（重複 modelNumber 禁止含む） */
export function validateListPrices(prices: ListPrice[]): string[] {
  const errors: string[] = [];
  const seen = new Set<string>();

  for (const p of prices) {
    const prefix = `price[${p.modelNumber || "?"}]: `;
    for (const err of validateListPrice(p)) {
      errors.push(prefix + err);
    }

    const key = p.modelNumber?.trim();
    if (key) {
      if (seen.has(key)) {
        errors.push(`duplicate modelNumber: ${p.modelNumber}`);
      } else {
        seen.add(key);
      }
    }
  }

  return errors;
}

/**
 * List の妥当性チェック（Go 側 validate() と概ね対応）
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateList(list: List): string[] {
  const errors: string[] = [];

  if (!list.id?.trim()) errors.push("id is required");
  if (!list.inventoryId?.trim())
    errors.push("inventoryId is required");
  if (!list.assigneeId?.trim())
    errors.push("assigneeId is required");
  if (!list.imageId?.trim())
    errors.push("imageId is required");

  if (!list.description?.trim()) {
    errors.push("description is required");
  } else if (list.description.length > MAX_DESCRIPTION_LENGTH) {
    errors.push(
      `description length must be <= ${MAX_DESCRIPTION_LENGTH}`,
    );
  }

  if (!list.createdBy?.trim())
    errors.push("createdBy is required");
  if (!isValidDateTimeString(list.createdAt)) {
    errors.push("createdAt must be a valid datetime");
  }

  if (!isValidListStatus(list.status)) {
    errors.push(
      "status must be 'listing' | 'suspended' | 'deleted'",
    );
  }

  // prices
  errors.push(...validateListPrices(list.prices || []));

  // updatedAt / updatedBy
  const hasUpdatedAt = !!list.updatedAt?.trim();
  const hasUpdatedBy = !!list.updatedBy?.trim();
  if (hasUpdatedAt && !isValidDateTimeString(list.updatedAt)) {
    errors.push("updatedAt must be a valid datetime when set");
  }
  if (!hasUpdatedAt && hasUpdatedBy) {
    errors.push("updatedBy must not be set without updatedAt");
  }
  if (
    hasUpdatedAt &&
    !isDateTimeOrderValid(list.createdAt, list.updatedAt!)
  ) {
    errors.push("updatedAt must be >= createdAt");
  }

  // deletedAt / deletedBy （両方 null か両方セット）
  const hasDeletedAt = !!list.deletedAt?.trim();
  const hasDeletedBy = !!list.deletedBy?.trim();
  if (hasDeletedAt && !isValidDateTimeString(list.deletedAt)) {
    errors.push("deletedAt must be a valid datetime when set");
  }
  if (hasDeletedAt !== hasDeletedBy) {
    errors.push(
      "deletedAt and deletedBy must be both set or both null",
    );
  }
  if (
    hasDeletedAt &&
    !isDateTimeOrderValid(list.createdAt, list.deletedAt!)
  ) {
    errors.push("deletedAt must be >= createdAt");
  }

  return errors;
}

/* =========================================================
 * Utility
 * =======================================================*/

/**
 * ListPrice 配列を Go 実装と同様のルールで集約:
 * - modelNumber を trim
 * - 空 modelNumber は無視
 * - price が範囲外の場合は無視
 * - 同一 modelNumber が複数ある場合は「最後の値」が採用される
 */
export function aggregateListPrices(
  prices: ListPrice[],
): ListPrice[] {
  const tmp = new Map<string, number>();
  const order: string[] = [];

  for (const p of prices || []) {
    const mn = (p.modelNumber || "").trim();
    if (!mn) continue;

    if (!tmp.has(mn)) {
      order.push(mn);
    }

    if (
      typeof p.price === "number" &&
      p.price >= MIN_PRICE &&
      p.price <= MAX_PRICE
    ) {
      tmp.set(mn, p.price);
    }
  }

  return order.map((mn) => ({
    modelNumber: mn,
    price: tmp.get(mn)!,
  }));
}

/**
 * List の正規化ヘルパ
 * - trim
 * - prices を aggregateListPrices で集約
 * - optional 日付/文字列は空文字なら null 扱い
 */
export function normalizeList(input: List): List {
  const norm = (v: string | null | undefined): string | null => {
    const t = v?.trim() ?? "";
    return t || null;
  };

  return {
    ...input,
    id: input.id.trim(),
    inventoryId: input.inventoryId.trim(),
    assigneeId: input.assigneeId.trim(),
    imageId: input.imageId.trim(),
    description: input.description.trim(),
    status: input.status,
    prices: aggregateListPrices(input.prices || []),
    createdBy: input.createdBy.trim(),
    createdAt: input.createdAt.trim(),
    updatedBy: norm(input.updatedBy),
    updatedAt: norm(input.updatedAt),
    deletedAt: norm(input.deletedAt),
    deletedBy: norm(input.deletedBy),
  };
}
