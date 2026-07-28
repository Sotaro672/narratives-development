// frontend/console/shell/src/features/list/infrastructure/payload/updateListPayload.ts

import type { UpdateListInput } from "../dto/updateListInput";
import { normalizePricesForBackend } from "./listPricePayload";

export function buildUpdateListPayloadArray(
  input: UpdateListInput,
): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    title: input.title || undefined,
    description: input.description,
    assigneeId: input.assigneeId || undefined,
    prices:
      input.priceRows === undefined
        ? undefined
        : normalizePricesForBackend(input.priceRows),
    status: input.status,
  };

  for (const key of Object.keys(payload)) {
    if (payload[key] === undefined) {
      delete payload[key];
    }
  }

  return payload;
}