// frontend/console/sales/application/sales_management_service.tsx
import {
  listSalesByCompanyId,
  type SalesRow,
} from "../infrastructure/sales_repository_http";

export type SalesManagementRow = SalesRow & {
  issueCount: number;
};

export type SalesManagementSortKey = "tokenName" | "brandName" | "issueCount";
export type SalesManagementSortDir = "asc" | "desc";

export type SalesManagementNavigateState = {
  tokenName: string;
  brandId: string;
  brandName: string;
  mintAddresses: string[];
  modelIds: string[];
  productBlueprints: SalesRow["productBlueprints"];
  owners: SalesRow["owners"];
};

export async function fetchSalesManagementRows(
  companyId: string,
): Promise<SalesRow[]> {
  const id = String(companyId ?? "").trim();

  if (!id) {
    return [];
  }

  const result = await listSalesByCompanyId(id);
  return Array.isArray(result.rows) ? result.rows : [];
}

export function enrichSalesManagementRows(
  rows: SalesRow[],
): SalesManagementRow[] {
  return rows.map((row) => ({
    ...row,
    issueCount: Array.isArray(row.mintAddresses)
      ? row.mintAddresses.length
      : 0,
  }));
}

export function sortSalesManagementRows(
  rows: SalesManagementRow[],
  sortKey: SalesManagementSortKey,
  sortDir: SalesManagementSortDir,
): SalesManagementRow[] {
  const next = [...rows];

  next.sort((a, b) => {
    let result = 0;

    switch (sortKey) {
      case "tokenName":
        result = compareStrings(a.tokenName ?? "", b.tokenName ?? "");
        break;
      case "brandName":
        result = compareStrings(a.brandName ?? "", b.brandName ?? "");
        break;
      case "issueCount":
        result = compareNumbers(a.issueCount ?? 0, b.issueCount ?? 0);
        break;
      default:
        result = 0;
        break;
    }

    return sortDir === "asc" ? result : -result;
  });

  return next;
}

export function normalizeSalesManagementSortKey(
  value: string,
): SalesManagementSortKey {
  if (value === "tokenName") {
    return "tokenName";
  }

  if (value === "brandName") {
    return "brandName";
  }

  return "issueCount";
}

export function buildSalesManagementNavigateState(
  row: SalesRow | undefined,
): SalesManagementNavigateState {
  return {
    tokenName: row?.tokenName ?? "",
    brandId: row?.brandId ?? "",
    brandName: row?.brandName ?? "",
    mintAddresses: row?.mintAddresses ?? [],
    modelIds: row?.modelIds ?? [],
    productBlueprints: row?.productBlueprints ?? [],
    owners: row?.owners ?? [],
  };
}

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ja");
}

function compareNumbers(a: number, b: number): number {
  return a - b;
}