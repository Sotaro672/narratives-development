// frontend/console/inventory/src/application/listCreate/listCreate.input.ts

import type {
  ResolvedListCreateParams,
  CreateListPriceRow,
} from "./listCreate.types";

// list create (POST /lists) の input 型（list側のDTO層）
import type { CreateListInput } from "../../../../list/src/infrastructure/dto";

export function normalizeCreateListPriceRows(
  rows: unknown[],
): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];

  return arr.map((r) => {
    const row = r as {
      modelId: string;
      price?: number | null;
    };

    return {
      modelId: row.modelId,
      price: row.price,
    };
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
  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    // inventoryId(pb__tb) をそのまま送る
    inventoryId: args.params.inventoryId,
    title: args.listingTitle,
    description: args.description,
    decision: args.decision,
    assigneeId: args.assigneeId,
    priceRows: priceRows as any,
  } as CreateListInput;
}

export function validateCreateListInput(input: CreateListInput): void {
  if (!(input as any).title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray((input as { priceRows?: unknown })?.priceRows)
    ? (input as { priceRows: unknown[] }).priceRows
    : [];

  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  const missingModelId = rows.find((r) => {
    const row = r as { modelId?: string };
    return !row.modelId;
  });

  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }
}