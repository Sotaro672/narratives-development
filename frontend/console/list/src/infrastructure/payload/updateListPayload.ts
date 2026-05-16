//frontend\console\list\src\infrastructure\payload\updateListPayload.ts
import type { UpdateListInput } from "../dto/updateListInput";
import { getCurrentUserUid } from "../http/authToken";
import { normalizePricesForBackendUpdate } from "./listPricePayload";

export function buildUpdateListPayloadArray(input: UpdateListInput): Record<string, any> {
  const uid = getCurrentUserUid();

  const title = String(input?.title ?? "");
  const description =
    input?.description === undefined ? undefined : String(input?.description ?? "");

  const prices = normalizePricesForBackendUpdate(input?.priceRows);

  let status: string | undefined = undefined;
  if (input?.decision === "list") status = "listing";
  if (input?.decision === "hold") status = "hold";

  const payload: Record<string, any> = {
    title: title || undefined,
    description,
    assigneeId: String(input?.assigneeId ?? "") || undefined,
    prices,
    status,
    decision: undefined,
    updatedBy: String(input?.updatedBy ?? "") || uid || undefined,
  };

  for (const k of Object.keys(payload)) {
    if (payload[k] === undefined) delete payload[k];
  }

  return payload;
}