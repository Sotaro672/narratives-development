//frontend\console\production\src\application\productBlueprint\loadProductBlueprintDetail.ts
import type { ProductBlueprintDetail } from "./types";
import { getIdTokenOrThrow } from "../../infrastructure/auth/firebaseAuth";
import { fetchProductBlueprintById } from "../../infrastructure/http/productBlueprintClient";

/* ---------------------------------------------------------
 * ProductBlueprint 詳細取得（usecase）
 * --------------------------------------------------------- */
export async function loadProductBlueprintDetail(
  productBlueprintId: string,
): Promise<ProductBlueprintDetail | null> {
  const id = productBlueprintId?.trim();
  if (!id) return null;

  const token = await getIdTokenOrThrow();

  const raw = (await fetchProductBlueprintById({
    productBlueprintId: id,
    token,
  })) as any;

  const qa = raw.qualityAssurance ?? raw.QualityAssurance ?? [];

  const rawTag = raw.productIdTag ?? raw.ProductIdTag ?? raw.ProductIDTag ?? null;

  let productIdTag = "";
  if (typeof rawTag === "string") {
    productIdTag = rawTag;
  } else if (rawTag && typeof rawTag === "object") {
    productIdTag = rawTag.Type ?? rawTag.type ?? rawTag.tag ?? "";
  }

  const detail: ProductBlueprintDetail = {
    id: raw.id ?? raw.ID ?? id,

    productName: raw.productName ?? raw.ProductName ?? "",
    companyId: raw.companyId ?? raw.CompanyID ?? "",
    brandId: raw.brandId ?? raw.BrandID ?? "",
    itemType: raw.itemType ?? raw.ItemType ?? "",
    fit: raw.fit ?? raw.Fit ?? "",
    material: raw.material ?? raw.Material ?? "",
    weight: Number(raw.weight ?? raw.Weight ?? 0),

    qualityAssurance: Array.isArray(qa) ? qa : [],
    productIdTag,
    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",

    createdBy: raw.createdBy ?? raw.CreatedBy ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? "",
    updatedBy: raw.updatedBy ?? raw.UpdatedBy ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? "",

    deletedBy: raw.deletedBy ?? raw.DeletedBy ?? null,
    deletedAt: raw.deletedAt ?? raw.DeletedAt ?? null,

    expireAt: raw.expireAt ?? raw.ExpireAt ?? null,
  };

  return detail;
}
