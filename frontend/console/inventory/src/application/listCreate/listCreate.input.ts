// frontend/console/inventory/src/application/listCreate/listCreate.input.ts

import type { ResolvedListCreateParams, CreateListPriceRow } from "./listCreate.types";
import { normalizeInventoryId, s, toNumberOrNull } from "./listCreate.utils";

// ✅ list create (POST /lists) の input 型（list側のHTTP層）
import type { CreateListInput } from "../../../../list/src/infrastructure/http/listRepositoryHTTP";

export function normalizeCreateListPriceRows(rows: any[]): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];
  return arr.map((r) => {
    const modelId = s((r as any)?.modelId ?? (r as any)?.ModelID);
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

  const missingModelId = rows.find((r: any) => !s(r?.modelId ?? r?.ModelID));
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
