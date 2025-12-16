// frontend\console\inventory\src\application\inventoryDetailService.tsx
import type { InventoryRow } from "../presentation/components/inventoryCard";
import {
  fetchInventoryDetailDTO,
  type InventoryDetailDTO,
  type ProductBlueprintPatchDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ============================================================
// ViewModel (Screen-friendly shape)
// ============================================================

export type InventoryDetailViewModel = {
  inventoryId: string;

  tokenBlueprintId: string;
  productBlueprintId: string;
  modelId: string;

  // ProductBlueprintCard に流し込むため（GetPatchByID の結果）
  productBlueprintPatch: ProductBlueprintPatchDTO;

  // InventoryCard rows
  rows: InventoryRow[];
  totalStock: number;

  updatedAt?: string;
};

// ============================================================
// Mapper
// ============================================================

function mapDtoToRows(dto: InventoryDetailDTO): InventoryRow[] {
  const rows = Array.isArray(dto.rows) ? dto.rows : [];

  return rows.map((r) => ({
    token: r.token ?? undefined,
    modelNumber: String(r.modelNumber ?? ""),
    size: String(r.size ?? ""),
    color: String(r.color ?? ""),
    rgb: (r.rgb ?? null) as any,
    stock: Number(r.stock ?? 0),
  }));
}

function mapDtoToViewModel(dto: InventoryDetailDTO): InventoryDetailViewModel {
  return {
    inventoryId: String(dto.inventoryId ?? ""),

    tokenBlueprintId: String(dto.tokenBlueprintId ?? ""),
    productBlueprintId: String(dto.productBlueprintId ?? ""),
    modelId: String(dto.modelId ?? ""),

    productBlueprintPatch: dto.productBlueprintPatch ?? {},

    rows: mapDtoToRows(dto),
    totalStock: Number(dto.totalStock ?? 0),

    updatedAt: dto.updatedAt,
  };
}

// ============================================================
// Query Request (Application Layer)
// - Detail 画面が「HTTPを叩いていない」切り分けができるようにログを厚めに
// ============================================================

export async function queryInventoryDetail(
  inventoryId: string,
): Promise<InventoryDetailViewModel> {
  const id = String(inventoryId ?? "").trim();
  if (!id) {
    throw new Error("inventoryId is empty");
  }

  console.log("[inventory/queryInventoryDetail] start", { inventoryId: id });

  const dto = await fetchInventoryDetailDTO(id);

  console.log("[inventory/queryInventoryDetail] dto received", {
    inventoryId: id,
    tokenBlueprintId: dto.tokenBlueprintId,
    productBlueprintId: dto.productBlueprintId,
    modelId: dto.modelId,
    totalStock: dto.totalStock,
    rowsCount: Array.isArray(dto.rows) ? dto.rows.length : 0,
    dto,
  });

  const vm = mapDtoToViewModel(dto);

  console.log("[inventory/queryInventoryDetail] mapped viewModel", {
    inventoryId: id,
    totalStock: vm.totalStock,
    rowsCount: vm.rows.length,
    rowsSample: vm.rows.slice(0, 5),
    productBlueprintPatch: vm.productBlueprintPatch,
  });

  return vm;
}
