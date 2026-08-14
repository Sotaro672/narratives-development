// frontend/console/shell/src/features/announcement/application/announcement_token_list_service.tsx

import {
  listSales,
  type SalesRow,
} from "../infrastructure/sales_repository_http";

export type AnnouncementTokenListRow = SalesRow & {
  issueCount: number;
  distributionCount: number;
};

export type AnnouncementTokenListSortKey =
  | "tokenName"
  | "brandName"
  | "issueCount"
  | "distributionCount";

export type AnnouncementTokenListSortDir = "asc" | "desc";

export type AnnouncementTokenListNavigateState = {
  owners: SalesRow["owners"];
};

/**
 * GET /sales
 *
 * 対象企業はバックエンドが認証情報から判定する。
 */
export async function fetchAnnouncementTokenListRows(): Promise<SalesRow[]> {
  const result = await listSales();
  return result.rows;
}

export function enrichAnnouncementTokenListRows(
  rows: SalesRow[],
): AnnouncementTokenListRow[] {
  return rows.map((row) => ({
    ...row,
    issueCount: row.assetIds.length,
    distributionCount: row.owners.length,
  }));
}

export function sortAnnouncementTokenListRows(
  rows: AnnouncementTokenListRow[],
  sortKey: AnnouncementTokenListSortKey,
  sortDir: AnnouncementTokenListSortDir,
): AnnouncementTokenListRow[] {
  const next = [...rows];

  next.sort((a, b) => {
    let result: number;

    switch (sortKey) {
      case "tokenName":
        result = compareStrings(a.tokenName, b.tokenName);
        break;

      case "brandName":
        result = compareStrings(a.brandName, b.brandName);
        break;

      case "issueCount":
        result = compareNumbers(a.issueCount, b.issueCount);
        break;

      case "distributionCount":
        result = compareNumbers(a.distributionCount, b.distributionCount);
        break;

      default:
        result = 0;
        break;
    }

    return sortDir === "asc" ? result : -result;
  });

  return next;
}

export function normalizeAnnouncementTokenListSortKey(
  value: string,
): AnnouncementTokenListSortKey {
  if (value === "tokenName") {
    return "tokenName";
  }

  if (value === "brandName") {
    return "brandName";
  }

  if (value === "distributionCount") {
    return "distributionCount";
  }

  return "issueCount";
}

export function buildAnnouncementTokenListNavigateState(
  row: SalesRow | undefined,
): AnnouncementTokenListNavigateState {
  return {
    owners: row?.owners ?? [],
  };
}

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ja");
}

function compareNumbers(a: number, b: number): number {
  return a - b;
}