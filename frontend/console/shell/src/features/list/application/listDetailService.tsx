// frontend/console/shell/src/features/list/application/listDetailService.tsx

import { fetchListByIdHTTP } from "../infrastructure/repository";

import type { ListDetailDTO } from "../infrastructure/dto";
import type { ListStatus } from "../../../shared/types/list";

import {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeImageUrls,
  normalizePriceRows,
  updatePriceRowPrice,
} from "./listDetail/listDetailMapper";

export type {
  ListDetailDTO,
  ListStatus,
};

export {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeImageUrls,
  normalizePriceRows,
  updatePriceRowPrice,
};

export type ListDetailRouteParams = {
  listId?: string;
  id?: string;
  inventoryId?: string;
};

export function resolveListDetailParams(
  params: ListDetailRouteParams | undefined,
) {
  const listId = String(
    params?.listId ||
      params?.id ||
      "",
  ).trim();

  const inventoryId = String(
    params?.inventoryId ?? "",
  ).trim();

  return {
    listId,
    inventoryId,
    raw: params,
  };
}

export async function loadListDetailDTO(
  args: {
    listId: string;
    inventoryIdHint?: string;
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