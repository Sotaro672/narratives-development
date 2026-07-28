// frontend/console/shell/src/features/list/infrastructure/payload/updateListPayload.ts

import type { UpdateListInput } from "../dto/updateListInput";
import { getCurrentUserUid } from "../http/authToken";
import { normalizePricesForBackend } from "./listPricePayload";

export function buildUpdateListPayloadArray(
  input: UpdateListInput,
): Record<string, unknown> {
  const uid = getCurrentUserUid();

  const payload: Record<string, unknown> = {
    title: input.title || undefined,
    description: input.description,
    assigneeId: input.assigneeId || undefined,
    prices: normalizePricesForBackend(input.priceRows),
    status: input.status,
    updatedBy: input.updatedBy || uid || undefined,
  };

  for (const key of Object.keys(payload)) {
    if (payload[key] === undefined) {
      delete payload[key];
    }
  }

  return payload;
}