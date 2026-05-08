// frontend/console/production/src/presentation/viewModels/toPrintCardRows.ts

import type { ProductionQuantityRowVM } from "./productionQuantityRowVM";

export type PrintCardRow = {
  modelId: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  quantity: number;
};

/**
 * VM → usePrintCard 用 row へ変換
 * VM の正キーは modelId
 */
export function toPrintCardRows(vms: ProductionQuantityRowVM[]): PrintCardRow[] {
  const safe = Array.isArray(vms) ? vms : [];

  return safe.map((vm, index) => {
    const modelId = String((vm as any).modelId ?? "").trim() || String(index);

    return {
      modelId,
      modelNumber: vm.modelNumber ?? "",
      size: vm.size ?? "",
      color: vm.color ?? "",
      rgb: vm.rgb ?? null,
      quantity: vm.quantity ?? 0,
    };
  });
}
