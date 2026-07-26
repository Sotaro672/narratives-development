// frontend/console/shell/src/features/production/infrastructure/api/productionCreateApi.ts
// ======================================================================
// Infrastructure API for Production Create
//   - 実際のHTTP / Firestoreなどの呼び出しを集約
// ======================================================================

import type { Brand } from "../../../brand/domain/entity/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { Member } from "../../../member/domain/entity/member";
import type { ModelVariationResponse } from "../../../productBlueprint/application/productBlueprintDetailService";

import { brandRepositoryHTTP } from "../../../brand/infrastructure/http/brandRepositoryHTTP";
import { fetchProductBlueprintManagementRows } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../productBlueprint/application/productBlueprintDetailService";
import { scopedFilterByCompanyId } from "../../../member/domain/repository/memberRepository";
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
export async function loadAssigneeCandidates(
  companyId: string,
): Promise<Member[]> {
  try {
    const filter = scopedFilterByCompanyId(
      companyId,
      {
        status: "active",
      },
    );

    const repository =
      new MemberRepositoryHTTP();

    const page = {
      number: 1,
      perPage: 200,
      totalPages: 1,
    };

    const result = await repository.list(
      page,
      filter,
    );

    return result.items ?? [];
  } catch {
    return [];
  }
}