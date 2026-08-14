// frontend/console/shell/src/features/mint/application/selector/applyMintRequestManagementQuery.ts

import type { ViewRow as MintRequestManagementRow } from "../usecase/loadMintRequestManagementRows";

export type MintRequestManagementSortKey =
  | "mintedAt"
  | "mintQuantity"
  | "productionQuantity"
  | null;

export type MintRequestManagementSortDirection =
  | "asc"
  | "desc"
  | null;

export type MintRequestManagementQuery = {
  tokenNames: readonly string[];
  productNames: readonly string[];
  requesterNames: readonly string[];
  inspectionStatuses: readonly MintRequestManagementRow["inspectionStatus"][];
  sortKey: MintRequestManagementSortKey;
  sortDirection: MintRequestManagementSortDirection;
};

/**
 * Backendのtime.Time JSONをtimestampへ変換する。
 * mintedAtはRFC3339系のAPI値を正とし、Frontend独自の日付形式は補完しない。
 * 解析できない値はnullとし、日付ソートでは常に末尾へ配置する。
 */
function toTimestamp(
  value: string | null | undefined,
): number | null {
  if (!value) return null;

  const timestamp = Date.parse(value);
  return Number.isNaN(timestamp) ? null : timestamp;
}

function matchesTextFilter(
  value: string | null | undefined,
  selectedValues: readonly string[],
): boolean {
  if (selectedValues.length === 0) return true;
  if (!value) return false;

  return selectedValues.includes(value);
}

function compareNullableTimestamps(
  left: string | null | undefined,
  right: string | null | undefined,
  direction: Exclude<MintRequestManagementSortDirection, null>,
): number {
  const leftTimestamp = toTimestamp(left);
  const rightTimestamp = toTimestamp(right);

  if (leftTimestamp === null && rightTimestamp === null) return 0;

  /**
   * 日付未設定または不正値は、
   * asc・descにかかわらず末尾へ配置する。
   */
  if (leftTimestamp === null) return 1;
  if (rightTimestamp === null) return -1;

  return direction === "asc"
    ? leftTimestamp - rightTimestamp
    : rightTimestamp - leftTimestamp;
}

/**
 * Mint申請一覧へフィルターとソートを適用する。
 *
 * 入力配列は変更せず、新しい配列を返す。
 * mintedAtはBackendから受け取った値をそのままソートに使用する。
 */
export function applyMintRequestManagementQuery(
  rows: readonly MintRequestManagementRow[],
  query: MintRequestManagementQuery,
): MintRequestManagementRow[] {
  const filteredRows = rows.filter((row) => {
    const tokenMatches = matchesTextFilter(
      row.tokenName,
      query.tokenNames,
    );

    const productMatches = matchesTextFilter(
      row.productName,
      query.productNames,
    );

    const requesterMatches = matchesTextFilter(
      row.requestedByName,
      query.requesterNames,
    );

    const statusMatches =
      query.inspectionStatuses.length === 0 ||
      query.inspectionStatuses.includes(
        row.inspectionStatus,
      );

    return (
      tokenMatches &&
      productMatches &&
      requesterMatches &&
      statusMatches
    );
  });

  const { sortKey, sortDirection } = query;

  if (!sortKey || !sortDirection) {
    return filteredRows;
  }

  return [...filteredRows].sort((left, right) => {
    if (sortKey === "mintQuantity") {
      return sortDirection === "asc"
        ? left.mintQuantity - right.mintQuantity
        : right.mintQuantity - left.mintQuantity;
    }

    if (sortKey === "productionQuantity") {
      return sortDirection === "asc"
        ? left.productionQuantity - right.productionQuantity
        : right.productionQuantity - left.productionQuantity;
    }

    return compareNullableTimestamps(
      left.mintedAt,
      right.mintedAt,
      sortDirection,
    );
  });
}