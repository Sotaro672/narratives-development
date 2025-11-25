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
 * currentMember.companyId に紐づく Brand の PageResult を取得。
 *
 * NOTE:
 *  - 引数 companyId は互換性のために残しているが **中では使用しない**。
 *  - 実際の companyId 絞り込みは backend 側の companyIDFromContext(ctx) に任せる。
 */
export async function fetchBrandListByCompanyId(
  _companyId: string,
  options: BrandListOptions = {},
): Promise<PageResult<Brand>> {
  const page = options.page ?? 1;
  const perPage = options.perPage ?? DEFAULT_PAGE_LIMIT;

  const filter: BrandFilter = {};
  // isActive フィルタだけフロント側から渡す
  if (typeof options.isActive === "boolean") {
    filter.isActive = options.isActive;
  }

  const sort: BrandSort =
    options.sort ?? ({ column: "created_at", order: "desc" } as const);

  // ★ companyId は送らず、backend が ctx の companyId で必ず絞る
  return brandRepositoryHTTP.list({
    filter,
    sort,
    page,
    perPage,
  });
}

/**
 * currentMember.companyId 単位で全ブランド（最大 200 件）だけ欲しい場合のヘルパー。
 * UI からはだいたいこちらを使う想定。
 *
 * NOTE:
 *  - 第一引数 companyId は互換性のために残しているが **無視される**。
 *  - backend が context.companyId を使ってフィルタする。
 */
export async function fetchAllBrandsForCompany(
  _companyId: string,
  isActiveOnly = false,
): Promise<Brand[]> {
  const result = await fetchBrandListByCompanyId("", {
    isActive: isActiveOnly ? true : undefined,
    page: 1,
    perPage: 200,
    sort: { column: "created_at", order: "desc" },
  });
  return result.items;
}

