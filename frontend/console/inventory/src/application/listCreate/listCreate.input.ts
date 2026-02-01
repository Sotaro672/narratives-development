// frontend/console/inventory/src/application/listCreate/listCreate.input.ts

import type { ResolvedListCreateParams, CreateListPriceRow } from "./listCreate.types";
import { normalizeInventoryId, s, toNumberOrNull } from "./listCreate.utils";

// ✅ list create (POST /lists) の input 型（list側のHTTP層）
import type { CreateListInput } from "../../../../list/src/infrastructure/http/listRepositoryHTTP";

/**
 * ✅ modelId の正規化
 * - 受け取り側で “名揺れ” を吸収する（UI層のVM/DTO混在を許容）
 * - ここで確実に modelId を埋める（空なら ""）
 */
function normalizeModelId(v: any): string {
  // 候補を優先順位順に拾う
  const cand =
    (v as any)?.modelId ??
    (v as any)?.ModelID ??
    (v as any)?.modelID ??
    (v as any)?.ModelId ??
    (v as any)?.id ?? // PriceCard VM が modelId を id に持っているケースを吸収
    (v as any)?.modelRefId ??
    (v as any)?.modelRefID ??
    (v as any)?.modelKey ??
    "";

  return s(cand);
}

export function normalizeCreateListPriceRows(rows: any[]): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];
  return arr.map((r) => {
    const modelId = normalizeModelId(r);
    const price = toNumberOrNull((r as any)?.price);
    return { modelId, price };
  });
}

export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    // ✅ 最重要: inventoryId(pb__tb) をそのまま送る
    inventoryId: normalizeInventoryId(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    priceRows: priceRows as any,
  } as CreateListInput;
}

export function validateCreateListInput(input: CreateListInput): void {
  const title = s((input as any)?.title);
  if (!title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray((input as any)?.priceRows) ? (input as any).priceRows : [];
  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  // ✅ modelId 名揺れを吸収した上でチェック（空文字はNG）
  const missingModelId = rows.find((r: any) => !s(r?.modelId ?? r?.ModelID ?? r?.id ?? r?.modelID ?? r?.ModelId));
  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }

  const hasPositivePrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n > 0;
  });
  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

  const hasZeroPrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n === 0;
  });
  if (hasZeroPrice) {
    throw new Error("価格に 0 円が含まれています。0 円は指定できません。");
  }
}
