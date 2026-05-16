//frontend\console\list\src\infrastructure\repository\listAggregateHttpRepository.ts
import type { ListAggregateDTO } from "../dto/listAggregateDto";
import { requestJSON } from "../http/httpClient";

const toStringSafe = (value: unknown): string => {
  if (typeof value === "string") return value.trim();
  if (value == null) return "";
  return String(value).trim();
};

const toListId = (value: unknown): string => toStringSafe(value);

export async function fetchListAggregateHTTP(
  listId: string,
): Promise<ListAggregateDTO> {
  const id = toListId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListAggregateDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/aggregate`,
  });
}