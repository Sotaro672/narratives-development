//frontend\console\list\src\infrastructure\payload\createListPayload.ts
import type { CreateListInput } from "../dto/createListInput";
import { getCurrentUserUid } from "../http/authToken";
import { normalizePricesForBackend } from "./listPricePayload";

export function buildCreateListPayloadArray(input: CreateListInput): Record<string, any> {
  const uid = getCurrentUserUid();

  const inventoryId = String(input?.inventoryId ?? "");
  const id = String(input?.id ?? "") || inventoryId;

  if (!id) {
    throw new Error("missing_id");
  }

  const title = String(input?.title ?? "");
  if (!title) {
    throw new Error("missing_title");
  }

  const prices = normalizePricesForBackend(input?.priceRows);

  return {
    id,
    inventoryId,
    title,
    description: String(input?.description ?? ""),
    assigneeId: String(input?.assigneeId ?? "") || undefined,
    createdBy: String(input?.createdBy ?? "") || uid || "system",
    prices,
  };
}