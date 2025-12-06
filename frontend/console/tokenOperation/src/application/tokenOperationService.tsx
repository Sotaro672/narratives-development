// frontend/console/tokenOperation/src/application/tokenOperationService.tsx

import type {
  TokenOperationExtended,
} from "../../../shell/src/shared/types/tokenOperation";
import { listTokenOperationsMintedByCompanyId } from "../infrastructure/repository/tokenOperationRepositoryHTTP";

export type SortKey =
  | "tokenName"
  | "symbol"
  | "brandName"
  | "assigneeName"
  | null;

export type SortDir = "asc" | "desc" | null;

export type FilterOption = {
  value: string;
  label: string;
};

export type TokenOperationFilterState = {
  brandFilter: string[];
  assigneeFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;
};

// ─────────────────────────────────────────────────────────────
// 一覧取得: ListByCompanyID → ListMintedCompleted 相当を呼び出し
// ─────────────────────────────────────────────────────────────

/**
 * companyId を指定して、ミント済みトークン設計（運用一覧）を取得する。
 * - backend: GET /token-blueprints?minted=minted
 * - middleware により companyId スコープが効く前提
 */
export async function fetchTokenOperationsForCompany(
  companyId: string,
): Promise<TokenOperationExtended[]> {
  const cid = companyId.trim();
  if (!cid) return [];
  return listTokenOperationsMintedByCompanyId(cid);
}

// ─────────────────────────────────────────────────────────────
// Filter option builder
// ─────────────────────────────────────────────────────────────

export function buildOptionsFromTokenOperations(
  rows: TokenOperationExtended[],
): {
  brandOptions: FilterOption[];
  assigneeOptions: FilterOption[];
} {
  const brandSet = new Set<string>();
  const assigneeSet = new Set<string>();

  for (const r of rows) {
    if (r.brandName) brandSet.add(r.brandName);
    if (r.assigneeName) assigneeSet.add(r.assigneeName);
  }

  const brandOptions: FilterOption[] = Array.from(brandSet).map((v) => ({
    value: v,
    label: v,
  }));

  const assigneeOptions: FilterOption[] = Array.from(assigneeSet).map((v) => ({
    value: v,
    label: v,
  }));

  return { brandOptions, assigneeOptions };
}

// ─────────────────────────────────────────────────────────────
// Filter + Sort
// ─────────────────────────────────────────────────────────────

export function filterAndSortTokenOperations(
  rows: TokenOperationExtended[],
  state: TokenOperationFilterState,
): TokenOperationExtended[] {
  const { brandFilter, assigneeFilter, sortKey, sortDir } = state;

  let data = rows.filter(
    (r) =>
      (brandFilter.length === 0 || brandFilter.includes(r.brandName)) &&
      (assigneeFilter.length === 0 ||
        assigneeFilter.includes(r.assigneeName)),
  );

  if (sortKey && sortDir) {
    data = [...data].sort((a, b) => {
      const av = (a[sortKey] ?? "") as string;
      const bv = (b[sortKey] ?? "") as string;
      const cmp = av.localeCompare(bv, "ja");
      return sortDir === "asc" ? cmp : -cmp;
    });
  }

  return data;
}
