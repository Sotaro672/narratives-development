// frontend/console/shell/src/features/production/application/productionManagementService.tsx

import { listProductionsHTTP } from "../infrastructure/query/productionQuery";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

export type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

export type ProductionRow = {
  id: string;
  productBlueprintId: string;
  productName: string;
  assigneeId: string;
  assigneeName: string;
  models: Array<{
    modelId: string;
    quantity: number;
  }>;
  printed: boolean;
  printedAt: string | null;
  printedBy: string | null;
  printedByName: string;
  createdBy: string | null;
  createdByName: string;
  createdAt: string;
  updatedBy: string | null;
  updatedByName: string;
  updatedAt: string | null;
  totalQuantity: number;
  brandName: string;
  printedAtLabel: string;
  createdAtLabel: string;
};

export type ProductionRowView = {
  id: string;
  productBlueprintId: string;
  productName: string;
  assigneeId: string;
  assigneeName: string;
  printed: boolean;
  totalQuantity: number;
  printedAtLabel: string;
  createdAtLabel: string;
  brandName: string;
};

function toTimestamp(value: string | null): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

/**
 * Production 一覧取得
 *
 * GET /productions の BFF response を正とする。
 * frontend では backend の値を再検証・補完せず、
 * UI が使用する lowerCamelCase への変換と
 * 表示用日時ラベルの生成だけを行う。
 */
export async function loadProductionRows(): Promise<ProductionRow[]> {
  const items = await listProductionsHTTP();

  return items.map((item): ProductionRow => ({
    id: item.ID,
    productBlueprintId: item.ProductBlueprintID,
    productName: item.productName ?? "",
    assigneeId: item.AssigneeID,
    assigneeName: item.assigneeName ?? "",
    models: item.Models.map((model) => ({
      modelId: model.ModelID,
      quantity: model.Quantity,
    })),
    printed: item.Printed,
    printedAt: item.PrintedAt ?? null,
    printedBy: item.PrintedBy ?? null,
    printedByName: item.printedByName ?? "",
    createdBy: item.CreatedBy ?? null,
    createdByName: item.createdByName ?? "",
    createdAt: item.CreatedAt,
    updatedBy: item.UpdatedBy ?? null,
    updatedByName: item.updatedByName ?? "",
    updatedAt: item.UpdatedAt ?? null,
    totalQuantity: item.totalQuantity,
    brandName: item.brandName ?? "",
    printedAtLabel: safeDateTimeLabelJa(item.PrintedAt ?? null, "-"),
    createdAtLabel: safeDateTimeLabelJa(item.CreatedAt, "-"),
  }));
}

/**
 * Production 一覧の frontend UI 状態に応じたフィルタ・ソート。
 * データの補完や backend response の正規化は行わない。
 */
export function buildRowsView(params: {
  baseRows: ProductionRow[];
  blueprintFilter: string[];
  assigneeFilter: string[];
  printedFilter: boolean[];
  sortKey: SortKey;
  sortDir: "asc" | "desc" | null;
}): ProductionRowView[] {
  const {
    baseRows,
    blueprintFilter,
    assigneeFilter,
    printedFilter,
    sortKey,
    sortDir,
  } = params;

  let rows = baseRows.filter((production) => {
    if (
      blueprintFilter.length > 0 &&
      !blueprintFilter.includes(production.productBlueprintId)
    ) {
      return false;
    }

    if (
      assigneeFilter.length > 0 &&
      !assigneeFilter.includes(production.assigneeId)
    ) {
      return false;
    }

    if (
      printedFilter.length > 0 &&
      !printedFilter.includes(production.printed)
    ) {
      return false;
    }

    return true;
  });

  if (sortKey && sortDir) {
    rows = [...rows].sort((a, b) => {
      if (sortKey === "totalQuantity") {
        return sortDir === "asc"
          ? a.totalQuantity - b.totalQuantity
          : b.totalQuantity - a.totalQuantity;
      }

      const aTimestamp = toTimestamp(
        sortKey === "printedAt" ? a.printedAt : a.createdAt,
      );
      const bTimestamp = toTimestamp(
        sortKey === "printedAt" ? b.printedAt : b.createdAt,
      );

      return sortDir === "asc"
        ? aTimestamp - bTimestamp
        : bTimestamp - aTimestamp;
    });
  }

  return rows.map((production) => ({
    id: production.id,
    productBlueprintId: production.productBlueprintId,
    productName: production.productName,
    assigneeId: production.assigneeId,
    assigneeName: production.assigneeName,
    printed: production.printed,
    totalQuantity: production.totalQuantity,
    printedAtLabel: production.printedAtLabel,
    createdAtLabel: production.createdAtLabel,
    brandName: production.brandName,
  }));
}