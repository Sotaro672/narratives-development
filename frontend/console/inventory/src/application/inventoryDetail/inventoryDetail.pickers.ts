// frontend/console/inventory/src/application/inventoryDetail/inventoryDetail.pickers.ts

import type {
  InventoryDetailDTO,
  TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";
import type {
  ProductBlueprintPatchDTOEx,
  TokenBlueprintPatchDTOEx,
} from "./inventoryDetail.types";
import { asString } from "./inventoryDetail.utils";

/**
 * ProductBlueprintPatch:
 * DTO 配列の中で最初に見つかった productBlueprintPatch を返す。
 * （B案 + 互換削除: 揺れ吸収なし）
 */
export function pickPatch(dtos: InventoryDetailDTO[]): ProductBlueprintPatchDTOEx {
  const found =
    dtos.find(
      (d) => d?.productBlueprintPatch && Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};
  return found as any;
}

/**
 * TokenBlueprintPatch:
 * B案では external(別取得) が実体なので external 優先。
 * DTO に埋め込みがあれば fallback で拾う。
 */
export function pickTokenBlueprintPatch(
  dtos: InventoryDetailDTO[],
  external?: TokenBlueprintPatchDTO | null,
): TokenBlueprintPatchDTOEx | undefined {
  const embedded =
    dtos.find(
      (d: any) => d?.tokenBlueprintPatch && Object.keys(d.tokenBlueprintPatch).length > 0,
    )?.tokenBlueprintPatch ?? undefined;

  const base = (external ?? embedded) as any;
  if (!base) return undefined;

  return {
    ...base,
    tokenName: asString(base?.tokenName) || undefined,
    symbol: asString(base?.symbol) || undefined,
    brandId: asString(base?.brandId) || undefined,
    brandName: asString(base?.brandName) || undefined,
    description: asString(base?.description) || undefined,
    metadataUri: asString(base?.metadataUri) || undefined,
    iconUrl: asString(base?.iconUrl) || undefined,
    minted: typeof base?.minted === "boolean" ? base.minted : base?.minted ?? undefined,
  } as any;
}

/**
 * updatedAt の最大（文字列比較でOKなフォーマット前提: ISO8601）
 */
export function pickUpdatedAtMax(dtos: InventoryDetailDTO[]): string | undefined {
  let maxUpdated: string | undefined = undefined;
  for (const d of dtos) {
    const t = d?.updatedAt ? String(d.updatedAt) : "";
    if (!t) continue;
    if (!maxUpdated || t > maxUpdated) maxUpdated = t;
  }
  return maxUpdated;
}

/** Patch から brandId を抜く（互換削除） */
export function pickBrandId(patch: any): string {
  return asString(patch?.brandId);
}

/** Patch から brandName を抜く（互換削除） */
export function pickBrandName(patch: any): string {
  return asString(patch?.brandName);
}

/** Patch から productName を抜く（互換削除） */
export function pickProductName(patch: any): string {
  return asString(patch?.productName);
}
