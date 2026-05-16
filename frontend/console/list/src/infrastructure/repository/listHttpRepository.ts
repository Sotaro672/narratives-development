//frontend\console\list\src\infrastructure\repository\listHttpRepository.ts
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import type { CreateListInput } from "../dto/createListInput";
import type { UpdateListInput } from "../dto/updateListInput";
import type { ListDTO } from "../dto/listDto";
import { requestJSON } from "../http/httpClient";
import { buildCreateListPayloadArray } from "../payload/createListPayload";
import { buildUpdateListPayloadArray } from "../payload/updateListPayload";

const toStringSafe = (value: unknown): string => {
  if (typeof value === "string") return value.trim();
  if (value == null) return "";
  return String(value).trim();
};

const toListId = (value: unknown): string => toStringSafe(value);

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
  const listId = toListId(input?.listId);
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
  const json = await requestJSON<ListDTO[]>({
    method: "GET",
    path: "/lists",
  });

  return Array.isArray(json) ? json : [];
}

export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = toListId(listId);
  if (!id) {
    throw new Error("invalid_list_id");
  }

  return await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
    debug: {
      tag: `GET /lists/${id}`,
      url: `${API_BASE}/lists/${encodeURIComponent(id)}`,
      method: "GET",
    },
  });
}

export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = toListId(args.listId);
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return await fetchListByIdHTTP(listId);
}