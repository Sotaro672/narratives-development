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

export function pickPatch(dtos: InventoryDetailDTO[]): ProductBlueprintPatchDTOEx {
  const found =
    dtos.find(
      (d) =>
        d?.productBlueprintPatch && Object.keys(d.productBlueprintPatch).length > 0,
    )?.productBlueprintPatch ?? {};

  // brandName 等の “増えがちなフィールド” を any 経由でも保持する
  return (found ?? {}) as any;
}

// ✅ tokenBlueprint patch を DTO群 or 外部取得結果から拾う
export function pickTokenBlueprintPatch(
  dtos: InventoryDetailDTO[],
  external?: TokenBlueprintPatchDTO | null,
): TokenBlueprintPatchDTOEx | undefined {
  // 1) DTO 内（embedded）を優先…ただし「実体が薄い / 期待キーが無い」場合があるので注意して扱う
  const embedded =
    (dtos.find(
      (d: any) => d?.tokenBlueprintPatch && Object.keys(d.tokenBlueprintPatch).length > 0,
    ) as any)?.tokenBlueprintPatch ?? undefined;

  const embeddedTokenName = asString((embedded as any)?.tokenName ?? (embedded as any)?.name);
  const embeddedSymbol = asString((embedded as any)?.symbol);

  // embedded が “あるが中身が空っぽ” っぽい場合は external を優先
  const shouldPreferExternal =
    (!!embedded && !embeddedTokenName && !embeddedSymbol) || embedded === null;

  const base = (shouldPreferExternal ? (external ?? embedded) : (embedded ?? external)) as any;
  if (!base) return undefined;

  return {
    ...base,
    tokenName: asString(base?.tokenName ?? base?.name) || undefined,
    brandId: asString(base?.brandId) || undefined,
    brandName: asString(base?.brandName) || undefined,
  } as any;
}

export function pickTokenNameFromDTO(dto: any): string {
  return (
    asString(dto?.tokenBlueprint?.name) ||
    asString(dto?.TokenBlueprint?.name) ||
    asString(dto?.tokenBlueprintName) ||
    ""
  );
}

export function pickUpdatedAtMax(dtos: InventoryDetailDTO[]): string | undefined {
  let maxUpdated: string | undefined = undefined;
  for (const d of dtos as any[]) {
    const t = d?.updatedAt ? String(d.updatedAt) : "";
    if (!t) continue;
    if (!maxUpdated || t > maxUpdated) maxUpdated = t;
  }
  return maxUpdated;
}

// Patch から “ありがちな揺れ” を吸収して brandId/brandName/productName を抜く
export function pickBrandId(patch: any): string {
  return (
    asString(patch?.brandId) ||
    asString(patch?.BrandID) ||
    asString(patch?.BrandId) ||
    ""
  );
}

export function pickBrandName(patch: any): string {
  return asString(patch?.brandName) || asString(patch?.BrandName) || "";
}

export function pickProductName(patch: any): string {
  return asString(patch?.productName) || asString(patch?.ProductName) || "";
}
