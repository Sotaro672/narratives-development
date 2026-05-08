// frontend/console/production/src/infrastructure/api/productionCreateApi.ts
// ======================================================================
// Infrastructure API for Production Create
//   - 実際の HTTP / Firestore などの呼び出しを集約
// ======================================================================

import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { ProductBlueprintManagementRow } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";

import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";
import { fetchProductBlueprintManagementRows } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../../../productBlueprint/src/application/productBlueprintDetailService";
import { scopedFilterByCompanyId } from "../../../../member/src/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";

// 型を必要ならアプリ層に再エクスポート
export type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ModelVariationResponse,
};

// ======================================================================
// ブランド API
// ======================================================================
export async function loadBrands(): Promise<Brand[]> {
  try {
    return await fetchAllBrandsForCompany("", true);
  } catch {
    return [];
  }
}

// ======================================================================
// 商品設計一覧 API
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
export async function loadDetailAndModels(pbId: string): Promise<{
  detail: any;
  models: ModelVariationResponse[];
}> {
  const [detail, models] = await Promise.all([
    getProductBlueprintDetail(pbId),
    listModelVariationsByProductBlueprintId(pbId),
  ]);
  return { detail, models };
}

// ======================================================================
// 担当者一覧 API
// ======================================================================
export async function loadAssigneeCandidates(
  companyId: string,
): Promise<Member[]> {
  try {
    const filter = scopedFilterByCompanyId(companyId, { status: "active" });
    const repo = new MemberRepositoryHTTP();
    const page = { number: 1, perPage: 200, totalPages: 1 };
    const result = await repo.list(page, filter);
    return result.items ?? [];
  } catch {
    return [];
  }
}
