// frontend/console/brand/src/infrastructure/query/brandQuery.ts

import type { Brand } from "../../domain/entity/brand";
import {
  brandRepositoryHTTP,
  type BrandFilter,
  type BrandSort,
  type PageParams,
  type PageResult,
} from "../http/brandRepositoryHTTP";

/**
 * 現在の Member と同じ companyId のブランド一覧を取得するための
 * 共通クエリ関数。
 *
 * - backend/auth.go でスコープされている companyId を
 *   フロント側でも明示的に指定して同じ brands を取得したい場合に使う。
 */

/**
 * companyId を指定して Brand の PageResult を取得
 */
export async function fetchBrandListByCompanyId(
  companyId: string,
  options: {
    isActive?: boolean;
    page?: PageParams["page"];
    perPage?: PageParams["perPage"];
    sort?: BrandSort;
  } = {},
): Promise<PageResult<Brand>> {
  const trimmedCompanyId = companyId.trim();
  if (!trimmedCompanyId) {
    // companyId が空の場合は空配列を返しておく（エラーにはしない）
    return {
      items: [],
      totalCount: 0,
      totalPages: 1,
      page: options.page ?? 1,
      perPage: options.perPage ?? 0,
    };
  }

  const filter: BrandFilter = {
    companyId: trimmedCompanyId,
  };

  if (typeof options.isActive === "boolean") {
    filter.isActive = options.isActive;
  }

  const sort: BrandSort = options.sort ?? {
    column: "created_at",
    order: "desc",
  };

  const page = options.page ?? 1;
  const perPage = options.perPage ?? 200;

  return brandRepositoryHTTP.list({
    filter,
    sort,
    page,
    perPage,
  });
}

/**
 * companyId 単位で全ブランド（最大 200 件）だけ欲しい場合の薄いヘルパー。
 * UI からはだいたいこちらを使う想定。
 */
export async function fetchAllBrandsForCompany(
  companyId: string,
  isActiveOnly = false,
): Promise<Brand[]> {
  const result = await fetchBrandListByCompanyId(companyId, {
    isActive: isActiveOnly ? true : undefined,
    page: 1,
    perPage: 200,
    sort: { column: "created_at", order: "desc" },
  });
  return result.items;
}
