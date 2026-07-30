// frontend/console/shell/src/features/mintRequest/application/selector/applyMintRequestManagementQuery.ts

import type {
  ViewRow as MintRequestManagementRow,
} from "../usecase/loadMintRequestManagementRows";

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
  tokenNames:
    readonly string[];

  productNames:
    readonly string[];

  requesterNames:
    readonly string[];

  inspectionStatuses:
    readonly MintRequestManagementRow["inspectionStatus"][];

  sortKey:
    MintRequestManagementSortKey;

  sortDirection:
    MintRequestManagementSortDirection;
};

/**
 * Date文字列をtimestampへ変換する。
 *
 * - ISO 8601
 * - YYYY/MM/DD
 * - YYYY/MM/DD HH:mm
 * - YYYY/MM/DD HH:mm:ss
 *
 * 解析できない値はnullとし、
 * 日付ソートでは常に末尾へ配置する。
 */
function toTimestamp(
  value:
    | string
    | null
    | undefined,
): number | null {
  if (!value) {
    return null;
  }

  const parsedTimestamp =
    Date.parse(value);

  if (
    !Number.isNaN(
      parsedTimestamp,
    )
  ) {
    return parsedTimestamp;
  }

  const match =
    value.match(
      /^(\d{4})\/(\d{1,2})\/(\d{1,2})(?:\s+(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?$/,
    );

  if (!match) {
    return null;
  }

  const year =
    Number(match[1]);

  const month =
    Number(match[2]);

  const day =
    Number(match[3]);

  const hour =
    Number(match[4] ?? "0");

  const minute =
    Number(match[5] ?? "0");

  const second =
    Number(match[6] ?? "0");

  const date =
    new Date(
      year,
      month - 1,
      day,
      hour,
      minute,
      second,
    );

  const timestamp =
    date.getTime();

  return Number.isNaN(timestamp)
    ? null
    : timestamp;
}

function matchesTextFilter(
  value:
    | string
    | null
    | undefined,
  selectedValues:
    readonly string[],
): boolean {
  if (
    selectedValues.length === 0
  ) {
    return true;
  }

  if (!value) {
    return false;
  }

  return selectedValues.includes(
    value,
  );
}

function compareNullableTimestamps(
  left:
    | string
    | null
    | undefined,
  right:
    | string
    | null
    | undefined,
  direction:
    Exclude<
      MintRequestManagementSortDirection,
      null
    >,
): number {
  const leftTimestamp =
    toTimestamp(left);

  const rightTimestamp =
    toTimestamp(right);

  if (
    leftTimestamp === null &&
    rightTimestamp === null
  ) {
    return 0;
  }

  /**
   * 日付未設定または不正値は、
   * asc・descにかかわらず末尾へ配置する。
   */
  if (leftTimestamp === null) {
    return 1;
  }

  if (rightTimestamp === null) {
    return -1;
  }

  return direction === "asc"
    ? leftTimestamp -
        rightTimestamp
    : rightTimestamp -
        leftTimestamp;
}

/**
 * Mint申請一覧へフィルターとソートを適用する。
 *
 * 入力配列は変更せず、新しい配列を返す。
 * mintedAtは表示用文字列へ変換する前の値を使用する。
 */
export function applyMintRequestManagementQuery(
  rows:
    readonly MintRequestManagementRow[],
  query: MintRequestManagementQuery,
): MintRequestManagementRow[] {
  const filteredRows =
    rows.filter((row) => {
      const tokenMatches =
        matchesTextFilter(
          row.tokenName,
          query.tokenNames,
        );

      const productMatches =
        matchesTextFilter(
          row.productName,
          query.productNames,
        );

      const requesterMatches =
        matchesTextFilter(
          row.requestedByName,
          query.requesterNames,
        );

      const statusMatches =
        query.inspectionStatuses
          .length === 0 ||
        query.inspectionStatuses
          .includes(
            row.inspectionStatus,
          );

      return (
        tokenMatches &&
        productMatches &&
        requesterMatches &&
        statusMatches
      );
    });

  const {
    sortKey,
    sortDirection,
  } = query;

  if (
    !sortKey ||
    !sortDirection
  ) {
    return filteredRows;
  }

  return [
    ...filteredRows,
  ].sort((left, right) => {
    if (
      sortKey ===
      "mintQuantity"
    ) {
      return sortDirection ===
        "asc"
        ? left.mintQuantity -
            right.mintQuantity
        : right.mintQuantity -
            left.mintQuantity;
    }

    if (
      sortKey ===
      "productionQuantity"
    ) {
      return sortDirection ===
        "asc"
        ? left.productionQuantity -
            right.productionQuantity
        : right.productionQuantity -
            left.productionQuantity;
    }

    return compareNullableTimestamps(
      left.mintedAt,
      right.mintedAt,
      sortDirection,
    );
  });
}