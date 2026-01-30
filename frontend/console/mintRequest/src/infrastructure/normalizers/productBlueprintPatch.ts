// frontend/console/mintRequest/src/infrastructure/normalizers/productBlueprintPatch.ts

import type {
  ProductBlueprintPatchDTO,
  ProductBlueprintModelRefDTO,
} from "../dto/mintRequestLocal.dto";
import { asMaybeString } from "./string";

function normalizeModelRefs(raw: any): ProductBlueprintModelRefDTO[] | null {
  if (!Array.isArray(raw)) return null;

  const out: ProductBlueprintModelRefDTO[] = [];
  const seen = new Set<string>();

  for (const r of raw) {
    const modelId = String(r?.modelId ?? r?.ModelID ?? "").trim();
    if (!modelId) continue;

    const orderRaw = r?.displayOrder ?? r?.DisplayOrder;
    const displayOrder =
      typeof orderRaw === "number"
        ? orderRaw
        : Number.isFinite(Number(orderRaw))
          ? Number(orderRaw)
          : NaN;

    if (!Number.isFinite(displayOrder) || displayOrder <= 0) continue;

    // 重複排除（先勝ち）
    if (seen.has(modelId)) continue;
    seen.add(modelId);

    out.push({ modelId, displayOrder });
  }

  // displayOrder で昇順に揃えておく（UI側ソートが単純になる）
  out.sort((a, b) => a.displayOrder - b.displayOrder);

  return out.length > 0 ? out : null;
}

/**
 * ProductBlueprintPatch normalize
 * - productIdTag を最終的に { type } に統一
 * - modelRefs を { modelId, displayOrder } に統一（DisplayOrder/ModelID の名揺れをここで吸収）
 */
export function normalizeProductBlueprintPatch(v: any): ProductBlueprintPatchDTO | null {
  if (!v) return null;

  // productIdTag はログ上 {Type: "..."} なので Type 対応は残す
  const rawTag = v?.productIdTag ?? null;

  let tagType: string | null = null;
  if (rawTag) {
    tagType =
      asMaybeString(rawTag?.type) ??
      asMaybeString(rawTag?.Type) ??
      asMaybeString(rawTag?.TYPE) ??
      (typeof rawTag === "string" ? asMaybeString(rawTag) : null);
  }

  const weightRaw = v?.weight;

  const out: ProductBlueprintPatchDTO = {
    productName: asMaybeString(v?.productName) ?? null,
    brandId: asMaybeString(v?.brandId) ?? null,
    brandName: asMaybeString(v?.brandName) ?? null,

    itemType: asMaybeString(v?.itemType) ?? null,
    fit: asMaybeString(v?.fit) ?? null,
    material: asMaybeString(v?.material) ?? null,

    weight:
      typeof weightRaw === "number"
        ? weightRaw
        : Number.isFinite(Number(weightRaw))
          ? Number(weightRaw)
          : null,

    qualityAssurance: (v?.qualityAssurance ?? null) as any,

    productIdTag: tagType ? { type: tagType } : null,

    assigneeId: asMaybeString(v?.assigneeId) ?? null,

    // ★ここで modelRefs を正規化して格納
    modelRefs: normalizeModelRefs(v?.modelRefs) ?? null,
  };

  return out;
}
