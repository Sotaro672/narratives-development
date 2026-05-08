//frontend\console\list\src\infrastructure\http\list\payloads.ts
import type { CreateListInput, UpdateListInput } from "./types";
import { getCurrentUserUid } from "./authToken";
import { normalizeListDocId } from "./ids";
import { s } from "./string";
import { toNumberOrNull } from "./number";

/**
 * ✅ create 用の prices を正規化する（modelId + price ONLY）
 *
 * - modelId が無い行があれば例外（送信しない）
 * - price が null / NaN なら例外（Go 側が非nullableの可能性が高い）
 */
export function normalizePricesForBackend(
  rows: CreateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r) => {
    const modelId = s((r as any)?.modelId);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      throw new Error("missing_modelId_in_priceRows");
    }

    if (priceMaybe === null) {
      throw new Error("missing_price_in_priceRows");
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ update 用: modelId を row.modelId から取得する
 */
export function normalizePricesForBackendUpdate(
  rows: UpdateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r, idx) => {
    const modelId = s((r as any)?.modelId);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      throw new Error(`missing_modelId_in_priceRows_at_${idx}`);
    }
    if (priceMaybe === null) {
      throw new Error(`missing_price_in_priceRows_at_${idx}`);
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ CreateList payload（最小）
 * - 「create時に送るのは modelId と price」の方針を厳守
 * - ✅ 方針A: inventoryId は pb__tb をそのまま送る
 */
export function buildCreateListPayloadArray(input: CreateListInput): Record<string, any> {
  const uid = getCurrentUserUid();

  const inventoryId = s(input?.inventoryId);
  const id = normalizeListDocId(input?.id) || inventoryId;

  if (!id) {
    throw new Error("missing_id");
  }

  const title = s(input?.title);
  if (!title) {
    throw new Error("missing_title");
  }

  const prices = normalizePricesForBackend(input?.priceRows);

  return {
    id,
    inventoryId,
    title,
    description: String(input?.description ?? ""),
    assigneeId: s(input?.assigneeId) || undefined,
    createdBy: s(input?.createdBy) || uid || "system",
    prices,
  };
}

/**
 * ✅ Update payload（最小）
 */
export function buildUpdateListPayloadArray(input: UpdateListInput): Record<string, any> {
  const uid = getCurrentUserUid();

  const title = s(input?.title);
  const description =
    input?.description === undefined ? undefined : String(input?.description ?? "");

  const prices = normalizePricesForBackendUpdate(input?.priceRows);

  let status: string | undefined = undefined;
  if (input?.decision === "list") status = "listing";
  if (input?.decision === "hold") status = "hold";

  const payload: Record<string, any> = {
    title: title || undefined,
    description,
    assigneeId: s(input?.assigneeId) || undefined,
    prices,
    status,
    decision: undefined,
    updatedBy: s(input?.updatedBy) || uid || undefined,
  };

  for (const k of Object.keys(payload)) {
    if (payload[k] === undefined) delete payload[k];
  }

  return payload;
}