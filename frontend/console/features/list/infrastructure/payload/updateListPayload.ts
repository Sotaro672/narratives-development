// frontend/console/list/src/infrastructure/payload/updateListPayload.ts
import type { UpdateListInput } from "../dto/updateListInput";
import { getCurrentUserUid } from "../http/authToken";
import { normalizePricesForBackendUpdate } from "./listPricePayload";

export function buildUpdateListPayloadArray(
  input: UpdateListInput,
): Record<string, any> {
  const uid = getCurrentUserUid();

  const title = String(input?.title ?? "");
  const description =
    input?.description === undefined
      ? undefined
      : String(input.description ?? "");

  const prices = normalizePricesForBackendUpdate(input?.priceRows);

  const payload: Record<string, any> = {
    title: title || undefined,
    description,
    assigneeId: String(input?.assigneeId ?? "") || undefined,
    prices,
    status: input.status,
    updatedBy: String(input?.updatedBy ?? "") || uid || undefined,
  };

  for (const key of Object.keys(payload)) {
    if (payload[key] === undefined) {
      delete payload[key];
    }
  }

  return payload;
}