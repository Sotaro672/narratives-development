// frontend/console/brand/src/infrastructure/query/brandQuery.ts

import type { Brand } from "../../domain/entity/brand";
import {
  brandRepositoryHTTP,
  type BrandFilter,
  type BrandSort,
} from "../http/brandRepositoryHTTP";

// ★ 共通のページング型／定数を利用
import {
  DEFAULT_PAGE_LIMIT,
  type PageResult,
} from "../../../../shell/src/shared/types/common/common";

/**
 * Brand 一覧取得時のオプション
 */
export type BrandListOptions = {
  /** true の場合は isActive = true のみ */
  isActive?: boolean;
  /** ページ番号 (1 始まり) */
  page?: number;
  /** 1ページあたり件数 */
  perPage?: number;
  /** ソート条件 */
  sort?: BrandSort;
};

/**
 * companyId を指定して Brand の PageResult を取得
 */
export async function fetchBrandListByCompanyId(
  companyId: string,
  options: BrandListOptions = {},
): Promise<PageResult<Brand>> {
  const trimmedCompanyId = companyId.trim();
  const page = options.page ?? 1;
  const perPage = options.perPage ?? DEFAULT_PAGE_LIMIT;

  if (!trimmedCompanyId) {
    // companyId が空の場合は空配列を返す（エラーにはしない）
    return {
      items: [],
      totalCount: 0,
      totalPages: 1,
      page,
      perPage,
    };
  }

  const filter: BrandFilter = {
    companyId: trimmedCompanyId,
  };

  if (typeof options.isActive === "boolean") {
    filter.isActive = options.isActive;
  }

  const sort: BrandSort =
    options.sort ?? ({ column: "created_at", order: "desc" } as const);

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
