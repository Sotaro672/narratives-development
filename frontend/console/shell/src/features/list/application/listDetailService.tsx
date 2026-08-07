// frontend/console/shell/src/features/list/application/listDetailService.tsx

import {
  deleteListHTTP,
  fetchListByIdHTTP,
} from "../infrastructure/repository";

import type { ListDetailDTO } from "../infrastructure/dto";

import { deriveListDetail } from "./listDetail/listDetailMapper";

export type {
  ListDetailDTO,
};

export {
  deriveListDetail,
};

export type ListDetailRouteParams = {
  listId?: string;
};

export function resolveListDetailParams(
  params: ListDetailRouteParams | undefined,
) {
  const listId = String(params?.listId ?? "").trim();

  return {
    listId,
  };
}

export async function loadListDetailDTO(
  args: {
    listId: string;
  },
): Promise<ListDetailDTO> {
  const listId = String(args.listId ?? "").trim();

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return fetchListByIdHTTP(listId);
}

export async function deleteListDetail(
  listId: string,
): Promise<void> {
  const id = String(listId ?? "").trim();

  if (!id) {
    throw new Error("invalid_list_id");
  }

  await deleteListHTTP(id);
}