// frontend/console/productBlueprint/src/application/productBlueprintManagementService.ts

import {
  // ★ インフラ層の一覧取得関数
  fetchProductBlueprintManagementRows as fetchRowsInfra,
  type ProductBlueprintManagementRow,
} from "../infrastructure/query/productBlueprintQuery";

export type UiRow = ProductBlueprintManagementRow;

export type ProductBlueprintSortKey = "createdAt" | "updatedAt" | null;
export type SortDirection = "asc" | "desc" | null;

// ★ ここから呼び出すと、内部で
//   （将来的に）ListIDsByCompany → ListNotYetPrinted を要求する HTTP が
//   叩かれる想定。現状は infra 側の fetchProductBlueprintManagementRows を
//   そのまま利用する。
export async function fetchProductBlueprintManagementRows(): Promise<UiRow[]> {
  return await fetchRowsInfra();
}

const toTs = (yyyyMd: string) => {
  if (!yyyyMd) return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

/**
 * printedFilter は UI 表示文字列で受ける（presentation 側と合わせる）
 * - "未印刷"
 * - "印刷済み"
 */
function matchPrintedFilter(rowPrinted: boolean, printedFilter: string[]): boolean {
  if (printedFilter.length === 0) return true;

  const wantsPrinted = printedFilter.includes("印刷済み");
  const wantsNotPrinted = printedFilter.includes("未印刷");

  // 両方選択されているならフィルタしない（=全件通す）
  if (wantsPrinted && wantsNotPrinted) return true;

  if (wantsPrinted) return rowPrinted === true;
  if (wantsNotPrinted) return rowPrinted === false;

  // 想定外値のみの場合は落とす（安全側）
  return false;
}

export function filterAndSortProductBlueprintRows(params: {
  allRows: UiRow[];
  brandFilter: string[];
  assigneeFilter: string[];
  printedFilter: string[];
  sortedKey: ProductBlueprintSortKey;
  sortedDir: SortDirection;
}): UiRow[] {
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
    work = work.filter((r) => brandFilter.includes(r.brandName));
  }

  if (assigneeFilter.length > 0) {
    work = work.filter((r) => assigneeFilter.includes(r.assigneeName));
  }

  // ✅ tagFilter を廃止して printedFilter に置換
  if (printedFilter.length > 0) {
    work = work.filter((r) => matchPrintedFilter(Boolean((r as any).printed), printedFilter));
  }

  if (sortedKey && sortedDir) {
    work = [...work].sort((a, b) => {
      const av = toTs(a[sortedKey]);
      const bv = toTs(b[sortedKey]);
      return sortedDir === "asc" ? av - bv : bv - av;
    });
  }

  return work;
}
