// frontend/console/shell/src/features/productBlueprint/application/productBlueprintManagementService.ts

import {
  listProductBlueprintsHTTP,
  type ProductBlueprintListRow,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

export type ProductBlueprintSortKey = "createdAt" | "updatedAt" | null;
export type SortDirection = "asc" | "desc" | null;

/**
 * backend BFFのGET /product-blueprintsレスポンスをそのまま返す。
 * frontend側でresponse mapperによる再構築は行わない。
 */
export async function fetchProductBlueprintManagementRows(): Promise<ProductBlueprintListRow[]> {
  return listProductBlueprintsHTTP();
}

function toTimestamp(value: string): number {
  if (!value) {
    return 0;
  }

  const timestamp = Date.parse(value);
  return Number.isFinite(timestamp) ? timestamp : 0;
}

/**
 * printedFilter は UI 表示文字列で受ける。
 * - "未印刷"
 * - "印刷済み"
 */
function matchPrintedFilter(rowPrinted: boolean, printedFilter: string[]): boolean {
  if (printedFilter.length === 0) {
    return true;
  }

  const wantsPrinted = printedFilter.includes("印刷済み");
  const wantsNotPrinted = printedFilter.includes("未印刷");

  if (wantsPrinted && wantsNotPrinted) {
    return true;
  }

  if (wantsPrinted) {
    return rowPrinted;
  }

  if (wantsNotPrinted) {
    return !rowPrinted;
  }

  return false;
}

export function filterAndSortProductBlueprintRows(params: {
  allRows: ProductBlueprintListRow[];
  brandFilter: string[];
  assigneeFilter: string[];
  printedFilter: string[];
  sortedKey: ProductBlueprintSortKey;
  sortedDir: SortDirection;
}): ProductBlueprintListRow[] {
  const {
    allRows,
    brandFilter,
    assigneeFilter,
    printedFilter,
    sortedKey,
    sortedDir,
  } = params;

  let work = allRows;

  if (brandFilter.length > 0) {
    work = work.filter((row) => brandFilter.includes(row.brandName));
  }

  if (assigneeFilter.length > 0) {
    work = work.filter((row) => assigneeFilter.includes(row.assigneeName));
  }

  if (printedFilter.length > 0) {
    work = work.filter((row) => matchPrintedFilter(row.printed, printedFilter));
  }

  if (sortedKey && sortedDir) {
    work = [...work].sort((a, b) => {
      const aTimestamp = toTimestamp(a[sortedKey]);
      const bTimestamp = toTimestamp(b[sortedKey]);

      return sortedDir === "asc"
        ? aTimestamp - bTimestamp
        : bTimestamp - aTimestamp;
    });
  }

  return work;
}