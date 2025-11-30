// frontend/console/production/src/application/productionDetailService.tsx

import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import {
  ProductionRepositoryHTTP,
  API_BASE as BACKEND_API_BASE,
} from "../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * 詳細表示用型（Production）
 */
export type ProductionDetail = Production & {
  totalQuantity: number;
  assigneeName?: string;
  productBlueprintName?: string;
  brandName?: string;
};

/**
 * 「モデル別 生産数一覧」カード用のモデル情報
 *   - modelVariationId ごとに 1 行
 *   - modelNumber / size / color / rgb は model モジュール側の情報から解決する
 */
export type ModelVariationSummary = {
  id: string; // modelVariationId
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
};

export type ProductionQuantityRow = {
  id: string; // modelVariationId
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  quantity: number;
};

/**
 * ProductBlueprint 詳細用型
 * Go 側:
 *   type ProductBlueprint struct {
 *     ID string
 *     ProductName string
 *     CompanyID   string
 *     BrandID     string
 *     ItemType    ItemType
 *     Fit         string
 *     Material    string
 *     Weight      float64
 *     QualityAssurance []string
 *     ProductIdTag     ProductIDTag
 *     AssigneeID       string
 *     CreatedBy *string
 *     CreatedAt time.Time
 *     UpdatedBy *string
 *     UpdatedAt time.Time
 *     DeletedBy *string
 *     DeletedAt *time.Time
 *     ExpireAt *time.Time
 *   }
 */
export type ProductBlueprintDetail = {
  id: string;

  productName: string;
  companyId: string;
  brandId: string;
  itemType: string;
  fit: string;
  material: string;
  weight: number;

  qualityAssurance: string[];
  productIdTag: string; // ← 画面表示用に文字列として扱う
  assigneeId: string;

  createdBy?: string | null;
  createdAt: string;
  updatedBy?: string | null;
  updatedAt: string;

  deletedBy?: string | null;
  deletedAt?: string | null;

  expireAt?: string | null;
};

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

// ---------------------------------------------------------
// Production 詳細取得
// ---------------------------------------------------------
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();
  const raw = (await repo.getById(productionId)) as any;
  if (!raw) return null;

  const rawModels = Array.isArray(raw.models)
    ? raw.models
    : Array.isArray(raw.Models)
    ? raw.Models
    : [];

  const totalQuantity = rawModels.reduce(
    (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
    0,
  );

  const blueprintId =
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "";

  let detail: ProductionDetail = {
    ...(raw as Production),
    id: raw.id ?? raw.ID ?? "",
    productBlueprintId: blueprintId,
    productBlueprintName:
      raw.productBlueprintName ??
      raw.ProductBlueprintName ??
      blueprintId,
    brandName:
      raw.brandName ??
      raw.BrandName ??
      raw.brand ??
      raw.Brand ??
      "",
    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",
    assigneeName: raw.assigneeName ?? raw.AssigneeName ?? "",
    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,
    printedAt: raw.printedAt ?? raw.PrintedAt ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? null,
    models: rawModels,
    totalQuantity,
  };

  // =====================================================
  // ★ 一覧データからの名前解決
  //   - productBlueprintName / brandName / assigneeName
  // =====================================================
  try {
    if (
      !detail.productBlueprintName ||
      detail.productBlueprintName === detail.productBlueprintId ||
      !detail.brandName ||
      !detail.assigneeName
    ) {
      const listItems = await listProductionsHTTP();

      const match = (listItems as any[]).find((item) => {
        const itemId = item.id ?? item.ID ?? "";
        const itemBlueprintId =
          item.productBlueprintId ?? item.ProductBlueprintID ?? "";
        return (
          itemId === detail.id ||
          (itemBlueprintId &&
            itemBlueprintId === detail.productBlueprintId)
        );
      });

      if (match) {
        const resolvedBlueprintName =
          match.productBlueprintName ??
          match.ProductBlueprintName ??
          detail.productBlueprintId;

        const resolvedBrandName =
          match.brandName ?? match.BrandName ?? "";

        const resolvedAssigneeName =
          match.assigneeName ?? match.AssigneeName ?? "";

        detail = {
          ...detail,
          productBlueprintName:
            detail.productBlueprintName &&
            detail.productBlueprintName !== detail.productBlueprintId
              ? detail.productBlueprintName
              : resolvedBlueprintName,
          brandName: detail.brandName || resolvedBrandName,
          assigneeName: detail.assigneeName || resolvedAssigneeName,
        };
      }
    }
  } catch (e) {
    console.warn(
      "[productionDetailService] failed to resolve names from list:",
      e,
    );
  }

  console.log("[productionDetailService] loaded detail:", detail);
  return detail;
}

// ---------------------------------------------------------
// ProductBlueprint 詳細取得
//   - /product-blueprints/{id} から productBlueprint 全体を取得
// ---------------------------------------------------------
export async function loadProductBlueprintDetail(
  productBlueprintId: string,
): Promise<ProductBlueprintDetail | null> {
  const id = productBlueprintId?.trim();
  if (!id) return null;

  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(id);

  const url = `${BACKEND_API_BASE}/product-blueprints/${safeId}`;

  console.log("[productionDetailService] loadProductBlueprintDetail request:", {
    url,
    productBlueprintId: id,
  });

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    console.error(
      "[productionDetailService] loadProductBlueprintDetail error:",
      res.status,
      res.statusText,
      body,
    );
    throw new Error(
      `ProductBlueprint API error: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = (await res.json()) as any;

  const qa =
    raw.qualityAssurance ??
    raw.QualityAssurance ??
    [];

  // ★ productIdTag は string / object 両対応で Type を抜き出す
  const rawTag =
    raw.productIdTag ??
    raw.ProductIdTag ??
    raw.ProductIDTag ??
    null;

  let productIdTag = "";
  if (typeof rawTag === "string") {
    productIdTag = rawTag;
  } else if (rawTag && typeof rawTag === "object") {
    productIdTag =
      rawTag.Type ??
      rawTag.type ??
      rawTag.tag ??
      "";
  }

  const detail: ProductBlueprintDetail = {
    id: raw.id ?? raw.ID ?? id,

    productName: raw.productName ?? raw.ProductName ?? "",
    companyId: raw.companyId ?? raw.CompanyID ?? "",
    brandId: raw.brandId ?? raw.BrandID ?? "",
    itemType: raw.itemType ?? raw.ItemType ?? "",
    fit: raw.fit ?? raw.Fit ?? "",
    material: raw.material ?? raw.Material ?? "",
    weight: Number(
      raw.weight ?? raw.Weight ?? 0,
    ),

    qualityAssurance: Array.isArray(qa) ? qa : [],
    productIdTag,
    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",

    createdBy: raw.createdBy ?? raw.CreatedBy ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? "",
    updatedBy: raw.updatedBy ?? raw.UpdatedBy ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? "",

    deletedBy: raw.deletedBy ?? raw.DeletedBy ?? null,
    deletedAt: raw.deletedAt ?? raw.DeletedAt ?? null,

    expireAt: raw.expireAt ?? raw.ExpireAt ?? null,
  };

  console.log(
    "[productionDetailService] loaded productBlueprint detail:",
    detail,
  );

  return detail;
}

// ---------------------------------------------------------
// モデル別 生産数一覧用の行を構築
//   - Production.Models の id / quantity と
//     「variationId → ModelVariationSummary」の index を突き合わせる
// ---------------------------------------------------------
export function buildQuantityRowsFromModels(
  models: any[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRow[] {
  const safeModels = Array.isArray(models) ? models : [];

  const rows: ProductionQuantityRow[] = safeModels.map((m: any) => {
    const id =
      m.id ??
      m.ID ??
      m.modelVariationId ??
      m.ModelVariationID ??
      "";

    const quantity = m.quantity ?? m.Quantity ?? 0;

    const meta = id ? modelIndex[id] : undefined;

    return {
      id,
      modelNumber: meta?.modelNumber ?? "",
      size: meta?.size ?? "",
      color: meta?.color ?? "",
      rgb: meta?.rgb ?? null,
      quantity,
    };
  });

  console.log(
    "[productionDetailService] buildQuantityRowsFromModels:",
    { models: safeModels, modelIndex, rows },
  );

  return rows;
}
