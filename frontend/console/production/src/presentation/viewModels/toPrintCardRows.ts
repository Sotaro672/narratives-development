// frontend/console/production/src/presentation/viewModels/toPrintCardRows.ts

import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

export type PrintCardRow = {
  modelVariationId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  quantity: number;
};

/**
 * VM → usePrintCard 用 row へ変換
 * VM の正キーは id（= modelVariationId と同一として扱う）
 */
export function toPrintCardRows(vms: ProductionQuantityRowVM[]): PrintCardRow[] {
  const safe = Array.isArray(vms) ? vms : [];

  return safe.map((vm, index) => {
    const id = String(vm.id ?? "").trim() || String(index);

    return {
      modelVariationId: id,
      modelNumber: vm.modelNumber ?? "",
      size: vm.size ?? "",
      color: vm.color ?? "",
      rgb: vm.rgb ?? null,
      quantity: vm.quantity ?? 0,
    };
  });
}

