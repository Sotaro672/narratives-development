// frontend/console/list/src/infrastructure/repository/listHttpRepository.ts

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

export async function createListHTTP(
  input: CreateListInput,
): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

  return requestJSON<ListDTO>({
    method: "POST",
    path: "/lists",
    body: payloadArray,
  });
}

export async function updateListByIdHTTP(
  input: UpdateListInput,
): Promise<ListDTO> {
  const listId = input.listId;

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  const payloadArray = buildUpdateListPayloadArray(input);

  return requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}`,
    body: payloadArray,
  });
}

export async function fetchListsHTTP(): Promise<ListDTO[]> {
  const response = await requestJSON<ListPageResponseDTO>({
    method: "GET",
    path: "/lists",
  });

  return response.items;
}

export async function fetchListByIdHTTP(
  listId: string,
): Promise<ListDTO> {
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(listId)}`,
  });
}