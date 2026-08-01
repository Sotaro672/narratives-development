// frontend/console/shell/src/features/list/infrastructure/repository/listHttpRepository.ts

import type { List } from "../../../../shared/types/list";

import type {
  PageResult,
} from "../../../../shared/types/common/common";

import type { CreateListInput } from "../dto/createListInput";
import type { ListDetailDTO } from "../dto/listDetailDto";

import { requestJSON } from "../http/httpClient";
import { buildCreateListPayloadArray } from "../payload/createListPayload";

export type ListManagementRowDTO = Pick<
  ListDetailDTO,
  | "id"
  | "inventoryId"
  | "title"
  | "productBlueprintId"
  | "tokenBlueprintId"
  | "productName"
  | "productBrandId"
  | "productBrandName"
  | "tokenName"
  | "tokenBrandId"
  | "tokenBrandName"
  | "assigneeId"
  | "assigneeName"
  | "status"
  | "createdAt"
>;

export async function createListHTTP(
  input: CreateListInput,
): Promise<List> {
  const payload =
    buildCreateListPayloadArray(input);

  return requestJSON<List>({
    method: "POST",
    path: "/lists",
    body: payload,
  });
}

export async function fetchListsHTTP(): Promise<
  ListManagementRowDTO[]
> {
  const response =
    await requestJSON<
      PageResult<ListManagementRowDTO>
    >({
      method: "GET",
      path: "/lists",
    });

  return response.items;
}

export async function fetchListByIdHTTP(
  listId: string,
): Promise<ListDetailDTO> {
  if (!listId) {
    throw new Error(
      "invalid_list_id",
    );
  }

  return requestJSON<ListDetailDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(listId)}`,
  });
}