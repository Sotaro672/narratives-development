// frontend/console/shell/src/features/production/application/productionManagementService.tsx

import type {
  ProductionListRow,
  ProductionListRowView,
  ProductionSortDirection,
  ProductionSortKey,
} from "../../../shared/types/production";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";
import { listProductionsHTTP } from "../infrastructure/api/productionManagementApi";

function toTimestamp(value?: string | null): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? 0 : timestamp;
}

/**
 * Production 一覧取得。
 *
 * GET /productions の lowerCamelCase BFF response を正とし、
 * frontend では表示用日時ラベルだけを追加する。
 */
export async function loadProductionRows(): Promise<ProductionListRow[]> {
  const items = await listProductionsHTTP();

  return items.map((item) => ({
    ...item,
    printedAtLabel: safeDateTimeLabelJa(item.printedAt, "-"),
    createdAtLabel: safeDateTimeLabelJa(item.createdAt, "-"),
  }));
}

/**
 * Production 一覧の frontend UI 状態に応じたフィルタ・ソート。
 * Backend response の変換・補完は行わない。
 */
export function buildRowsView(params: {
  baseRows: ProductionListRow[];
  blueprintFilter: string[];
  assigneeFilter: string[];
  printedFilter: boolean[];
  sortKey: ProductionSortKey;
  sortDir: ProductionSortDirection;
}): ProductionListRowView[] {
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