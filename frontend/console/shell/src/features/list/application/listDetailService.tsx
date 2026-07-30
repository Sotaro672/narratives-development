// frontend/console/shell/src/features/list/application/listDetailService.tsx

import { fetchListByIdHTTP } from "../infrastructure/repository";

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
  const listId = String(
    params?.listId ?? "",
  ).trim();

  return {
    listId,
  };
}

export async function loadListDetailDTO(
  args: {
    listId: string;
  },
): Promise<ListDetailDTO> {
  const listId = String(
    args.listId ?? "",
  ).trim();

  if (!listId) {
    throw new Error("invalid_list_id");
  }

  return fetchListByIdHTTP(listId);
}