// frontend/shell/src/shared/types/inventory.ts

/**
 * InventoryModel
 * backend/internal/domain/inventory/entity.go の InventoryModel に対応。
 *
 * - modelNumber ごとに quantity（在庫数）を保持
 */
export interface InventoryModel {
  modelNumber: string;
  quantity: number;
}

/**
 * InventoryStatus
 * backend/internal/domain/inventory/entity.go の InventoryStatus に対応。
 *
 * - "inspecting" : 検品中
 * - "inspected"  : 検品完了
 * - "listed"     : 出品中（List 連携想定）
 * - "discarded"  : 廃棄
 * - "deleted"    : 論理削除
 */
export type InventoryStatus =
  | "inspecting"
  | "inspected"
  | "listed"
  | "discarded"
  | "deleted";

/** InventoryStatus の妥当性チェック */
export function isValidInventoryStatus(
  s: string,
): s is InventoryStatus {
  return (
    s === "inspecting" ||
    s === "inspected" ||
    s === "listed" ||
    s === "discarded" ||
    s === "deleted"
  );
}

/**
 * Inventory
 * backend/internal/domain/inventory/entity.go の Inventory に対応。
 *
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）
 * - connectedToken は null 許容（未連携時）
 */
export interface Inventory {
  id: string;
  /** 連携トークンID。未連携時は null */
  connectedToken: string | null;
  /** モデル別在庫一覧 */
  models: InventoryModel[];
  /** 在庫ロケーション（必須） */
  location: string;
  /** ステータス（必須） */
  status: InventoryStatus;
  /** 作成者（必須） */
  createdBy: string;
  /** 作成日時（必須, ISO8601） */
  createdAt: string;
  /** 更新者（必須） */
  updatedBy: string;
  /** 更新日時（必須, ISO8601, createdAt 以上） */
  updatedAt: string;
}

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

/** InventoryModel 単体の妥当性チェック */
export function validateInventoryModel(
  m: InventoryModel,
): string[] {
  const errors: string[] = [];

  if (!m.modelNumber?.trim()) {
    errors.push("modelNumber is required");
  }
  if (
    m.quantity == null ||
    Number.isNaN(m.quantity) ||
    m.quantity < 0
  ) {
    errors.push("quantity must be >= 0");
  }

  return errors;
}

/** InventoryModel 配列全体の妥当性チェック（重複 modelNumber 禁止） */
export function validateInventoryModels(
  models: InventoryModel[],
): string[] {
  const errors: string[] = [];
  const seen = new Set<string>();

  for (const m of models || []) {
    const prefix = `model[${m.modelNumber || "?"}]: `;
    for (const err of validateInventoryModel(m)) {
      errors.push(prefix + err);
    }

    const key = m.modelNumber?.trim();
    if (key) {
      if (seen.has(key)) {
        errors.push(
          `duplicate modelNumber: ${m.modelNumber}`,
        );
      } else {
        seen.add(key);
      }
    }
  }

  return errors;
}

/**
 * Inventory の妥当性チェック（Go 側 validate() に概ね対応）
 * 問題があればエラーメッセージ配列を返す。
 */
export function validateInventory(
  inv: Inventory,
): string[] {
  const errors: string[] = [];

  if (!inv.id?.trim()) errors.push("id is required");

  if (!inv.location?.trim()) {
    errors.push("location is required");
  }

  if (!isValidInventoryStatus(inv.status)) {
    errors.push(
      "status must be 'inspecting' | 'inspected' | 'listed' | 'discarded' | 'deleted'",
    );
  }

  if (!inv.createdBy?.trim()) {
    errors.push("createdBy is required");
  }
  if (!isValidDateTimeString(inv.createdAt)) {
    errors.push("createdAt must be a valid datetime");
  }

  if (!inv.updatedBy?.trim()) {
    errors.push("updatedBy is required");
  }
  if (!isValidDateTimeString(inv.updatedAt)) {
    errors.push("updatedAt must be a valid datetime");
  }
  if (
    isValidDateTimeString(inv.createdAt) &&
    isValidDateTimeString(inv.updatedAt) &&
    !isDateTimeOrderValid(inv.createdAt, inv.updatedAt)
  ) {
    errors.push("updatedAt must be >= createdAt");
  }

  // connectedToken: null or non-empty
  if (
    inv.connectedToken !== null &&
    inv.connectedToken.trim() === ""
  ) {
    errors.push(
      "connectedToken must be non-empty when provided",
    );
  }

  // models
  errors.push(...validateInventoryModels(inv.models || []));

  return errors;
}

/* =========================================================
 * Utility / Normalization
 * =======================================================*/

/**
 * InventoryModel 配列を Go 実装 aggregateModels と同様に集約:
 * - modelNumber を trim
 * - 空 modelNumber / quantity < 0 は無視
 * - 同一 modelNumber は quantity を合算（合計が負なら 0 に補正）
 * - 元の順序をできるだけ維持
 */
export function aggregateInventoryModels(
  models: InventoryModel[],
): InventoryModel[] {
  const buf = new Map<string, number>();
  const order: string[] = [];

  for (const m of models || []) {
    const num = (m.modelNumber || "").trim();
    if (!num) continue;
    if (
      m.quantity == null ||
      Number.isNaN(m.quantity) ||
      m.quantity < 0
    ) {
      continue;
    }

    if (!buf.has(num)) {
      order.push(num);
      buf.set(num, 0);
    }
    const next = (buf.get(num) ?? 0) + m.quantity;
    buf.set(num, next < 0 ? 0 : next);
  }

  return order.map((num) => ({
    modelNumber: num,
    quantity: buf.get(num) ?? 0,
  }));
}

/**
 * Inventory の正規化ヘルパ
 * - 文字列を trim
 * - connectedToken: 空文字は null
 * - models は aggregateInventoryModels で集約
 */
export function normalizeInventory(
  input: Inventory,
): Inventory {
  const norm = (v: string | null | undefined): string | null => {
    const t = v?.trim() ?? "";
    return t || null;
  };

  return {
    ...input,
    id: input.id.trim(),
    connectedToken: norm(input.connectedToken),
    models: aggregateInventoryModels(input.models || []),
    location: input.location.trim(),
    status: input.status,
    createdBy: input.createdBy.trim(),
    createdAt: input.createdAt.trim(),
    updatedBy: input.updatedBy.trim(),
    updatedAt: input.updatedAt.trim(),
  };
}

/* =========================================================
 * Behavior helpers (TS 版ユーティリティ)
 * Go 側メソッドに対応するミューテータを関数で提供。
 * =======================================================*/

/** トークン連携 */
export function connectToken(
  inv: Inventory,
  token: string,
): Inventory {
  const t = token.trim();
  if (!t) throw new Error("invalid connectedToken");
  return { ...inv, connectedToken: t };
}

/** トークン連携解除 */
export function disconnectToken(inv: Inventory): Inventory {
  return { ...inv, connectedToken: null };
}

/** ロケーション更新 */
export function updateLocation(
  inv: Inventory,
  location: string,
): Inventory {
  const loc = location.trim();
  if (!loc) throw new Error("invalid location");
  return { ...inv, location: loc };
}

/** ステータス更新 */
export function updateStatus(
  inv: Inventory,
  status: InventoryStatus,
): Inventory {
  if (!isValidInventoryStatus(status)) {
    throw new Error("invalid status");
  }
  return { ...inv, status };
}

/** 単一モデルの数量をセット（存在しなければ追加） */
export function setModelQuantity(
  inv: Inventory,
  modelNumber: string,
  quantity: number,
): Inventory {
  const mn = modelNumber.trim();
  if (!mn) throw new Error("invalid modelNumber");
  if (quantity < 0) throw new Error("invalid quantity");

  const next = {
    ...inv,
    models: aggregateInventoryModels([
      ...inv.models,
      { modelNumber: mn, quantity },
    ]),
  };
  return next;
}
