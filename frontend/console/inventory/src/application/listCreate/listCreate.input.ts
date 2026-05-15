// frontend/console/inventory/src/application/listCreate/listCreate.input.ts

import type {
  ResolvedListCreateParams,
  CreateListPriceRow,
} from "./listCreate.types";
import { s, toNumberOrNull } from "./listCreate.utils";

// list create (POST /lists) の input 型（list側のHTTP層）
import type { CreateListInput } from "../../../../list/src/infrastructure/http/list";

export function normalizeCreateListPriceRows(
  rows: unknown[],
): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];

  return arr.map((r) => {
    const row = r as {
      modelId?: unknown;
      price?: unknown;
    };

    const modelId = s(row.modelId);
    const price = toNumberOrNull(row.price);

    return { modelId, price };
  });
}

export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: unknown[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    // inventoryId(pb__tb) をそのまま送る
    inventoryId: s(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    priceRows: priceRows as any,
  } as CreateListInput;
}

export function validateCreateListInput(input: CreateListInput): void {
  const title = s((input as { title?: unknown })?.title);
  if (!title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray((input as { priceRows?: unknown })?.priceRows)
    ? (input as { priceRows: unknown[] }).priceRows
    : [];

  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  const missingModelId = rows.find((r) => {
    const row = r as { modelId?: unknown };
    return !s(row.modelId);
  });

  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }

  const hasPositivePrice = rows.some((r) => {
    const row = r as { price?: unknown };
    const n = toNumberOrNull(row.price);
    return n !== null && n > 0;
  });

  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

  const hasZeroPrice = rows.some((r) => {
    const row = r as { price?: unknown };
    const n = toNumberOrNull(row.price);
    return n !== null && n === 0;
  });

  if (hasZeroPrice) {
    throw new Error("価格に 0 円が含まれています。0 円は指定できません。");
  }
}