// frontend/console/shell/src/features/tokenBlueprint/application/tokenBlueprintManagementService.tsx

import type { TokenBlueprint } from "../../../shared/types/tokenBlueprint";

import { fetchTokenBlueprints } from "../infrastructure/repository/tokenBlueprintRepositoryHTTP";

/**
 * ISO 8601文字列をtimestampへ変換する。
 *
 * 未設定または不正な値は0として扱う。
 */
function toTimestamp(
  iso: string | undefined,
): number {
  if (!iso) {
    return 0;
  }

  const timestamp =
    Date.parse(iso);

  return Number.isNaN(timestamp)
    ? 0
    : timestamp;
}

export type SortKey =
  | "createdAt"
  | null;

export type SortDir =
  | "asc"
  | "desc"
  | null;

export type TokenBlueprintFilterState = {
  brandFilter: string[];
  assigneeFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;
};

/**
 * 認証中の会社に属するTokenBlueprint一覧を取得する。
 *
 * companyIdはfrontendから指定しない。
 * backendが認証コンテキストから会社境界を判定する。
 *
 * mintedによる絞り込みは行わず、
 * true / falseの両方を返す。
 */
export async function fetchTokenBlueprintsForCompany(): Promise<
  TokenBlueprint[]
> {
  const result =
    await fetchTokenBlueprints({
      page: 1,
      perPage: 200,
    });

  return result.items;
}

/**
 * TokenBlueprint一覧から、
 * brandとassigneeのフィルター選択肢を生成する。
 */
export function buildOptionsFromTokenBlueprints(
  rows: TokenBlueprint[],
): {
  brandOptions: {
    value: string;
    label: string;
  }[];

  assigneeOptions: {
    value: string;
    label: string;
  }[];
} {
  const brandIds =
    new Set<string>();

  const assigneeIds =
    new Set<string>();

  for (const row of rows) {
    if (row.brandId) {
      brandIds.add(
        row.brandId,
      );
    }

    if (row.assigneeId) {
      assigneeIds.add(
        row.assigneeId,
      );
    }
  }

  const brandOptions =
    Array.from(brandIds).map(
      (brandId) => {
        return {
          value: brandId,
          label: brandId,
        };
      },
    );

  const assigneeOptions =
    Array.from(assigneeIds).map(
      (assigneeId) => {
        return {
          value: assigneeId,
          label: assigneeId,
        };
      },
    );

  return {
    brandOptions,
    assigneeOptions,
  };
}

/**
 * TokenBlueprint一覧へフィルターとソートを適用する。
 *
 * mintedによる絞り込みは行わない。
 */
export function filterAndSortTokenBlueprints(
  rows: TokenBlueprint[],
  state: TokenBlueprintFilterState,
): TokenBlueprint[] {
  const {
    brandFilter,
    assigneeFilter,
    sortKey,
    sortDir,
  } = state;

  const filtered =
    rows.filter((row) => {
      const matchesBrand =
        brandFilter.length === 0 ||
        brandFilter.includes(
          row.brandId,
        );

      const matchesAssignee =
        assigneeFilter.length === 0 ||
        assigneeFilter.includes(
          row.assigneeId,
        );

      return (
        matchesBrand &&
        matchesAssignee
      );
    });

  if (
    sortKey !== "createdAt" ||
    sortDir === null
  ) {
    return filtered;
  }

  return [...filtered].sort(
    (
      first,
      second,
    ) => {
      const firstTimestamp =
        toTimestamp(
          first.createdAt,
        );

      const secondTimestamp =
        toTimestamp(
          second.createdAt,
        );

      return sortDir === "asc"
        ? firstTimestamp -
            secondTimestamp
        : secondTimestamp -
            firstTimestamp;
    },
  );
}