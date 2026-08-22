// frontend/console/shell/src/features/list/infrastructure/payload/createListPayload.ts

import type { CreateListInput } from "../dto/createListInput";

export function buildCreateListPayloadArray(
  input: CreateListInput,
): Record<string, unknown> {
  const inventoryId = input.inventoryId ?? "";
  const id = input.id || inventoryId;

  if (!id) {
    throw new Error("missing_id");
  }

  if (!input.title) {
    throw new Error("missing_title");
  }

  return {
    id,
    inventoryId,
    title: input.title,
    description: input.description,
    status: input.status,
    assigneeId: input.assigneeId || undefined,
    prices: input.priceRows,
  };
}