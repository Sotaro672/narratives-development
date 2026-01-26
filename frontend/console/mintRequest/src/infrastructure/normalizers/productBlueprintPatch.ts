// frontend/console/mintRequest/src/infrastructure/normalizers/productBlueprintPatch.ts

import type { ProductBlueprintPatchDTO } from "../dto/mintRequestLocal.dto";
import { asMaybeString } from "./string";

/**
 * ProductBlueprintPatch normalize
 * - productIdTag を最終的に { type } に統一
 * - 受け取りは Type/type/TYPE, ネスト等に緩く対応
 */
export function normalizeProductBlueprintPatch(
  v: any,
): ProductBlueprintPatchDTO | null {
  if (!v) return null;

  const rawTag = v?.productIdTag ?? v?.ProductIdTag ?? v?.product_id_tag ?? null;

  let tagType: string | null = null;

  if (rawTag) {
    tagType =
      asMaybeString(rawTag?.type) ??
      asMaybeString(rawTag?.Type) ??
      asMaybeString(rawTag?.TYPE);

    if (!tagType && typeof rawTag?.type === "object") {
      tagType =
        asMaybeString(rawTag?.type?.type) ??
        asMaybeString(rawTag?.type?.Type) ??
        null;
    }
    if (!tagType && typeof rawTag?.Type === "object") {
      tagType =
        asMaybeString(rawTag?.Type?.type) ??
        asMaybeString(rawTag?.Type?.Type) ??
        null;
    }

    if (!tagType && typeof rawTag === "string") {
      tagType = asMaybeString(rawTag);
    }
  }

  const out: ProductBlueprintPatchDTO = {
    productName: asMaybeString(v?.productName ?? v?.ProductName) ?? null,
    brandId: asMaybeString(v?.brandId ?? v?.BrandID ?? v?.BrandId) ?? null,
    brandName: asMaybeString(v?.brandName ?? v?.BrandName) ?? null,

    itemType: asMaybeString(v?.itemType ?? v?.ItemType) ?? null,
    fit: asMaybeString(v?.fit ?? v?.Fit) ?? null,
    material: asMaybeString(v?.material ?? v?.Material) ?? null,

    weight:
      typeof (v?.weight ?? v?.Weight) === "number"
        ? (v?.weight ?? v?.Weight)
        : Number(v?.weight ?? v?.Weight) || null,

    qualityAssurance:
      (v?.qualityAssurance ??
        v?.QualityAssurance ??
        v?.washTags ??
        v?.WashTags ??
        null) ?? null,

    productIdTag: tagType ? { type: tagType } : null,

    assigneeId:
      asMaybeString(v?.assigneeId ?? v?.AssigneeID ?? v?.AssigneeId) ?? null,
  };

  return out;
}
