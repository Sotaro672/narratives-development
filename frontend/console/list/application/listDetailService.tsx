// frontend/console/list/src/application/listDetailService.tsx
import type * as React from "react";
import {
  fetchListByIdHTTP,
  updateListByIdHTTP,
} from "../infrastructure/repository";
import type { ListDTO } from "../infrastructure/dto";
import type { ListStatus } from "../domain/list";
import {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeImageUrls,
  normalizePriceRows,
  updatePriceRowPrice,
} from "./listDetail/listDetailMapper";
export type { ListStatus };
export {
  computeListDetailPageTitle,
  deriveListDetail,
  formatYMDHM,
  normalizeImageUrls,
  normalizePriceRows,
  updatePriceRowPrice,
};
// ---------------------------------------------------------
// Types (presentation hook から使う)
// ---------------------------------------------------------
export type ListDetailRouteParams = {
  listId?: string;
  id?: string;
  inventoryId?: string;
};
// hook 内では必要なフィールドだけ参照するので Record<any> でOK
export type ListDetailDTO = ListDTO;
/**
 * presentation hook が import しても落ちないように（ts2305 対策）
 * - 本来は hook 側の型でOKだが、移譲の過程で service から import しているケースがあるため export する
 */
export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;
  // loading/error
  loading: boolean;
  error: string;
  // raw dto
  dto: ListDetailDTO | null;
  // listing (view)
  listingTitle: string;
  description: string;
  // status (view)
  status: ListStatus | "";
  // display strings (already trimmed)
  productBrandId: string;
  productBrandName: string;
  productName: string;
  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;
  // images (view)
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  // price (PriceCard 用)
  priceRows: any[];
  priceCard: any;
  // admin (view)
  assigneeId: string;
  assigneeName: string;
  createdByName: string;
  createdAt: string;
  // 更新者/更新日時
  updatedByName: string;
  updatedAt: string;
  // edit helpers (optional)
  listId?: string;
  inventoryId?: string;
};
// ---------------------------------------------------------
// Route params
// ---------------------------------------------------------
export function resolveListDetailParams(
  params: ListDetailRouteParams | undefined,
) {
  // ルートパラメータ名の違い（listId / id）は吸収
  const listId = String(params?.listId || params?.id || "").trim();
  const inventoryId = String(params?.inventoryId ?? "").trim();
  return {
    listId,
    inventoryId,
    raw: params,
  };
}
// ---------------------------------------------------------
// Service API
// ---------------------------------------------------------
export async function loadListDetailDTO(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDetailDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");
  return await fetchListByIdHTTP(listId);
}
// ---------------------------------------------------------
// Update API
// - repositoryHTTP 側が priceRows を受けて prices を正規化するので、service からは priceRows を渡す
// ---------------------------------------------------------
export async function updateListDetailDTO(args: {
  listId: string;
  title?: string;
  description?: string;
  // PriceCard rows（id/modelId = modelId）
  priceRows?: any[];
  status?: ListStatus;
  assigneeId?: string;
  updatedBy?: string;
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");
  return await updateListByIdHTTP({
    listId,
    title: args.title,
    description: args.description,
    // prices ではなく priceRows を渡す
    priceRows: args.priceRows,
    status: args.status,
    assigneeId: args.assigneeId,
    updatedBy: args.updatedBy,
  });
}