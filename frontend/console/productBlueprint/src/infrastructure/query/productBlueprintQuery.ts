// frontend/console/productBlueprint/src/infrastructure/query/productBlueprintQuery.ts

import { listProductBlueprintsHTTP } from "../repository/productBlueprintRepositoryHTTP";
import { fetchBrandNameById } from "../../../../brand/src/infrastructure/http/brandRepositoryHTTP";
import { fetchMemberDisplayNameById } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";

export type ProductBlueprintManagementRow = {
  id: string;
  productName: string;
  brandName: string;
  assigneeName: string;
  productIdTag: string;
  createdAt: string; // YYYY/MM/DD
  updatedAt: string; // YYYY/MM/DD
};

type RawProductBlueprintListRow = {
  id?: string;
  productName?: string;

  brandId?: string;
  assigneeId?: string;

  productIdTag?: string | null;

  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string | null;
};

const toDisplayDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso ?? "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

export async function fetchProductBlueprintManagementRows(): Promise<ProductBlueprintManagementRow[]> {
  const list = await listProductBlueprintsHTTP();

  const uiRows: ProductBlueprintManagementRow[] = [];

  for (const pb of list as RawProductBlueprintListRow[]) {
    // 🚫 論理削除済みは絶対に一覧へ出さない
    if (pb.deletedAt != null) continue;

    const brandId = pb.brandId ?? "";
    const brandName = brandId ? await fetchBrandNameById(brandId) : "";

    const assigneeId = (pb.assigneeId ?? "").trim();
    let assigneeName = "-";
    if (assigneeId) {
      const displayName = await fetchMemberDisplayNameById(assigneeId);
      assigneeName = displayName.trim() || assigneeId;
    }

    const productIdTag = (pb.productIdTag ?? "").trim() || "-";

    const createdAtDisp = toDisplayDate(pb.createdAt ?? "");
    const updatedAtDisp = toDisplayDate(pb.updatedAt ?? pb.createdAt ?? "");

    uiRows.push({
      id: pb.id ?? "",
      productName: pb.productName ?? "",
      brandName,
      assigneeName,
      productIdTag,
      createdAt: createdAtDisp,
      updatedAt: updatedAtDisp,
    });
  }

  return uiRows;
}
