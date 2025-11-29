// frontend/console/production/src/application/productionCreateService.tsx

import { fetchAllBrandsForCompany } from "../../../brand/src/infrastructure/query/brandQuery";
import type { Brand } from "../../../brand/src/domain/entity/brand";

import {
  fetchProductBlueprintManagementRows,
  type ProductBlueprintManagementRow,
} from "../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  type ModelVariationResponse,
} from "../../../productBlueprint/src/application/productBlueprintDetailService";

import type {
  ItemType,
  Fit,
} from "../../../productBlueprint/src/domain/entity/catalog";

import type { Member } from "../../../member/src/domain/entity/member";
import {
  scopedFilterByCompanyId,
  type MemberSort,
} from "../../../member/src/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";
import { getMemberFullName } from "../../../member/src/domain/entity/member";

import type { ProductionQuantityRow } from "../presentation/components/productionQuantityCard";

// ------------------------------------------------------------
// 型エクスポート（hook 側は service からだけ import すればよい）
// ------------------------------------------------------------
export type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ModelVariationResponse,
};

// ------------------------------------------------------------
// ProductBlueprintCard 用の型
// ------------------------------------------------------------
export type ProductBlueprintForCard = {
  id: string;
  productName: string;
  brand?: string;

  itemType?: ItemType;
  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

// ------------------------------------------------------------
// ブランド関連
// ------------------------------------------------------------
export async function loadBrands(): Promise<Brand[]> {
  try {
    const items = await fetchAllBrandsForCompany("", true);
    return items;
  } catch (e) {
    console.error("[productionCreateService] loadBrands error:", e);
    return [];
  }
}

export function buildBrandOptions(brands: Brand[]): string[] {
  return brands.map((b) => b.name).filter(Boolean) as string[];
}

// ------------------------------------------------------------
// 商品設計一覧関連
// ------------------------------------------------------------
export async function loadProductBlueprints(): Promise<
  ProductBlueprintManagementRow[]
> {
  try {
    const rows = await fetchProductBlueprintManagementRows();
    return rows;
  } catch (e) {
    console.error(
      "[productionCreateService] loadProductBlueprints error:",
      e,
    );
    return [];
  }
}

export function filterProductBlueprintsByBrand(
  all: ProductBlueprintManagementRow[],
  selectedBrand: string | null,
): ProductBlueprintManagementRow[] {
  if (!selectedBrand) return [];
  return all.filter((pb) => pb.brandName === selectedBrand);
}

export type ProductRow = { id: string; name: string };

export function buildProductRows(
  filtered: ProductBlueprintManagementRow[],
): ProductRow[] {
  return filtered.map((pb) => ({
    id: pb.id,
    name: pb.productName,
  }));
}

// ------------------------------------------------------------
// 詳細 + ModelVariation 一括取得
// ------------------------------------------------------------
export async function loadDetailAndModels(
  productBlueprintId: string,
): Promise<{ detail: any | null; models: ModelVariationResponse[] }> {
  const id = productBlueprintId.trim();
  if (!id) return { detail: null, models: [] };

  try {
    const [detail, models] = await Promise.all([
      getProductBlueprintDetail(id),
      listModelVariationsByProductBlueprintId(id),
    ]);

    console.log(
      "[productionCreateService] fetched model variations for productBlueprintId:",
      id,
      models,
    );

    return { detail, models };
  } catch (e) {
    console.error(
      "[productionCreateService] loadDetailAndModels error:",
      e,
    );
    return { detail: null, models: [] };
  }
}

// ------------------------------------------------------------
// ProductBlueprintCard 用データ組み立て
// ------------------------------------------------------------
export function buildSelectedForCard(
  selectedDetail: any | null,
  selectedMgmtRow: ProductBlueprintManagementRow | null,
): ProductBlueprintForCard {
  if (selectedDetail) {
    return {
      id: selectedDetail.id,
      productName: selectedDetail.productName,
      brand: selectedDetail.brandName ?? "",
      itemType: selectedDetail.itemType as ItemType,
      fit: selectedDetail.fit as Fit,
      materials: selectedDetail.material,
      weight: selectedDetail.weight,
      washTags: selectedDetail.qualityAssurance ?? [],
      productIdTag: selectedDetail.productIdTag?.type ?? "",
    };
  }

  if (selectedMgmtRow) {
    return {
      id: selectedMgmtRow.id,
      productName: selectedMgmtRow.productName,
      brand: selectedMgmtRow.brandName,
    };
  }

  return {
    id: "",
    productName: "",
    brand: "",
  };
}

// ------------------------------------------------------------
// 担当者候補関連
// ------------------------------------------------------------
export async function loadAssigneeCandidates(
  companyId: string,
): Promise<Member[]> {
  const id = companyId.trim();
  if (!id) return [];

  try {
    const filter = scopedFilterByCompanyId(id, {
      status: "active",
    });

    const sort: MemberSort = {
      column: "name",
      order: "asc",
    };

    const page: any = { number: 1, perPage: 200 };

    const repo = new MemberRepositoryHTTP();
    const result = await repo.list(page, filter);
    return result.items ?? [];
  } catch (e) {
    console.error(
      "[productionCreateService] loadAssigneeCandidates error:",
      e,
    );
    return [];
  }
}

export type AssigneeOption = { id: string; name: string };

export function buildAssigneeOptions(members: Member[]): AssigneeOption[] {
  return members.map((m) => {
    const full = getMemberFullName(m);
    return {
      id: m.id,
      name: full || m.email || m.id,
    };
  });
}

// ------------------------------------------------------------
// ModelVariation → ProductionQuantityCard 用 rows 変換
// ------------------------------------------------------------
export function mapModelVariationsToRows(
  modelVariations: ModelVariationResponse[],
): ProductionQuantityRow[] {
  return modelVariations.map((mv) => {
    const rgb = mv.color?.rgb ?? null;

    let colorCode: string | undefined;
    if (rgb === 0) {
      // ★ rgb:0 は黒として扱う
      colorCode = "#000000";
    } else if (typeof rgb === "number") {
      colorCode = `#${rgb.toString(16).padStart(6, "0")}`;
    } else {
      colorCode = "#FFFFFF";
    }

    return {
      modelCode: mv.modelNumber,
      size: mv.size,
      colorName: mv.color?.name ?? "",
      colorCode,
      // 生産数は初期値 0（編集カード側で更新）
      stock: 0,
    };
  });
}
