// frontend/console/production/src/presentation/viewModels/toProductionDetailUpdateRows.ts

import type { ProductionQuantityRow as DetailQuantityRow } from "../../application/detail/types";
import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

/**
 * VM → updateProductionDetail 用 DTO（detail row）へ変換
 * VM の正キーは modelId
 */
export function toProductionDetailUpdateRows(
  vms: ProductionQuantityRowVM[],
): DetailQuantityRow[] {
  const safe = Array.isArray(vms) ? vms : [];

  return safe.map((vm, index) => {
    const modelId = String((vm as any).modelId ?? "").trim() || String(index);

    return {
      modelId,
      modelNumber: vm.modelNumber ?? "",
      size: vm.size ?? "",
      color: vm.color ?? "",
      rgb: vm.rgb ?? null,
      displayOrder: vm.displayOrder,
      quantity: vm.quantity ?? 0,
    };
  });
}
