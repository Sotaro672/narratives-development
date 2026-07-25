// frontend\console\list\src\infrastructure\repository\listHttpRepository.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import type { CreateListInput } from "../dto/createListInput";
import type { UpdateListInput } from "../dto/updateListInput";
import type { ListDTO } from "../dto/listDto";
import { requestJSON } from "../http/httpClient";
import { buildCreateListPayloadArray } from "../payload/createListPayload";
import { buildUpdateListPayloadArray } from "../payload/updateListPayload";

type ListPageResponseDTO = {
  items: ListDTO[];
  page: number;
  perPage: number;
  totalCount: number;
  totalPages: number;
};

export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

  return await requestJSON<ListDTO>({
    method: "POST",
    path: "/lists",
    body: payloadArray,
    debug: {
      tag: "POST /lists",
      url: `${API_BASE}/lists`,
      method: "POST",
      body: payloadArray,
    },
  });
}

export async function updateListByIdHTTP(input: UpdateListInput): Promise<ListDTO> {
  const listId = input?.listId;
  if (!listId) throw new Error("invalid_list_id");

  const payloadArray = buildUpdateListPayloadArray(input);

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}`,
    body: payloadArray,
    debug: {
      tag: `PUT /lists/${listId}`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
      method: "PUT",
      body: payloadArray,
    },
  });
}

export async function fetchListsHTTP(): Promise<ListDTO[]> {
  const json = await requestJSON<ListPageResponseDTO>({
    method: "GET",
    path: "/lists",
  });

  return Array.isArray(json.items) ? json.items : [];
}

export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(listId)}`,
    debug: {
      tag: `GET /lists/${listId}`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
      method: "GET",
    },
  });
}

export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = args.listId;
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return await fetchListByIdHTTP(listId);
}