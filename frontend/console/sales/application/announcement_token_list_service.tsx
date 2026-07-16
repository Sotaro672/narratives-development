// frontend/console/sales/application/announcement_token_list_service.tsx
import {
  listSales,
  type SalesRow,
} from "../infrastructure/sales_repository_http";

export type AnnouncementTokenListRow =
  SalesRow & {
    issueCount: number;
  };

export type AnnouncementTokenListSortKey =
  | "tokenName"
  | "brandName"
  | "issueCount";

export type AnnouncementTokenListSortDir =
  | "asc"
  | "desc";

export type AnnouncementTokenListNavigateState = {
  tokenName: string;
  brandId: string;
  brandName: string;
  mintAddresses: string[];
  modelIds: string[];
  productBlueprints: SalesRow["productBlueprints"];
  owners: SalesRow["owners"];
};

/**
 * GET /salesは認証情報から対象企業を判定する。
 *
 * companyIdは呼び出し元との互換性を維持するため
 * 引数として受け取るが、APIには送信しない。
 */
export async function fetchAnnouncementTokenListRows(
  _companyId: string,
): Promise<SalesRow[]> {
  const result = await listSales();

  return result.rows;
}

export function enrichAnnouncementTokenListRows(
  rows: SalesRow[],
): AnnouncementTokenListRow[] {
  return rows.map((row) => ({
    ...row,
    issueCount: row.mintAddresses.length,
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
        result = compareStrings(
          a.tokenName,
          b.tokenName,
        );
        break;

      case "brandName":
        result = compareStrings(
          a.brandName,
          b.brandName,
        );
        break;

      case "issueCount":
        result = compareNumbers(
          a.issueCount,
          b.issueCount,
        );
        break;

      default:
        result = 0;
        break;
    }

    return sortDir === "asc"
      ? result
      : -result;
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

  return "issueCount";
}

export function buildAnnouncementTokenListNavigateState(
  row: SalesRow | undefined,
): AnnouncementTokenListNavigateState {
  if (!row) {
    return {
      tokenName: "",
      brandId: "",
      brandName: "",
      mintAddresses: [],
      modelIds: [],
      productBlueprints: [],
      owners: [],
    };
  }

  return {
    tokenName: row.tokenName,
    brandId: row.brandId,
    brandName: row.brandName,
    mintAddresses: row.mintAddresses,
    modelIds: row.modelIds,
    productBlueprints: row.productBlueprints,
    owners: row.owners,
  };
}

function compareStrings(
  a: string,
  b: string,
): number {
  return a.localeCompare(b, "ja");
}

function compareNumbers(
  a: number,
  b: number,
): number {
  return a - b;
}