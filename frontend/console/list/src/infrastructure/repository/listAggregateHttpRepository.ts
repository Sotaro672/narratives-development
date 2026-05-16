// frontend\console\list\src\infrastructure\repository\listAggregateHttpRepository.ts
import type { ListAggregateDTO } from "../dto/listAggregateDto";
import { requestJSON } from "../http/httpClient";

export async function fetchListAggregateHTTP(
  listId: string,
): Promise<ListAggregateDTO> {
  if (!listId) throw new Error("invalid_list_id");

  return await requestJSON<ListAggregateDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(listId)}/aggregate`,
  });
}