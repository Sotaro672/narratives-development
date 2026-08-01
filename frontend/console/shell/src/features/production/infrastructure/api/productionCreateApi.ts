// frontend/console/shell/src/features/production/infrastructure/api/productionCreateApi.ts
// ======================================================================
// Infrastructure API for Production Create
//   - 実際のHTTP / Firestoreなどの呼び出しを集約
// ======================================================================

import type { Brand } from "../../../../shared/types/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { Member } from "../../../../shared/types/member";
import type { ModelVariationResponse } from "../../../productBlueprint/application/productBlueprintDetailService";

import { createPageFromCurrent } from "../../../../shared/types/common/common";

import { brandRepositoryHTTP } from "../../../brand/infrastructure/http/brandRepositoryHTTP";
import { fetchProductBlueprintManagementRows } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../productBlueprint/application/productBlueprintDetailService";
import { MemberRepositoryHTTP } from "../../../member/infrastructure/http/memberRepositoryHTTP";

// 型をアプリケーション層へ再エクスポート
export type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ModelVariationResponse,
};

// ======================================================================
// ブランドAPI
// ======================================================================
export async function loadBrands(): Promise<Brand[]> {
  try {
    const result = await brandRepositoryHTTP.list({
      page: 1,
      perPage: 200,
    });

    return result.items.filter(
      (brand) => brand.isActive,
    );
  } catch {
    return [];
  }
}

// ======================================================================
// 商品設計一覧API
// ======================================================================
export async function loadProductBlueprints(): Promise<
  ProductBlueprintManagementRow[]
> {
  try {
    return await fetchProductBlueprintManagementRows();
  } catch {
    return [];
  }
}

// ======================================================================
// 詳細 + ModelVariations API
// ======================================================================
export async function loadDetailAndModels(
  pbId: string,
): Promise<{
  detail: unknown;
  models: ModelVariationResponse[];
}> {
  const [detail, models] = await Promise.all([
    getProductBlueprintDetail(pbId),
    listModelVariationsByProductBlueprintId(pbId),
  ]);

  return {
    detail,
    models,
  };
}

// ======================================================================
// 担当者一覧API
// ======================================================================
// companyIdによるスコープはBackend側で、
// 認証中MemberのcompanyIdを基に適用する。
// ======================================================================
export async function loadAssigneeCandidates(): Promise<
  Member[]
> {
  try {
    const repository =
      new MemberRepositoryHTTP();

    const page = createPageFromCurrent(
      1,
      200,
    );

    const result = await repository.list(
      page,
      {
        status: "active",
      },
    );

    return result.items;
  } catch {
    return [];
  }
}