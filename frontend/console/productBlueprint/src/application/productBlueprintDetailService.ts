// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";

import {
  updateProductBlueprintApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
  type NewModelVariationMeasurements,
  type NewModelVariationPayload,
} from "../infrastructure/api/productBlueprintDetailApi";

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import { fetchAllBrandsForCompany } from "../../../brand/src/infrastructure/query/brandQuery";
import {
  fetchMemberByIdWithToken,
  formatLastFirst,
} from "../../../member/src/infrastructure/query/memberQuery";

// -----------------------------------------
// HEX -> number(RGB) 変換
// -----------------------------------------
function hexToRgbInt(hex?: string): number | undefined {
  if (!hex) return undefined;
  const trimmed = hex.trim();
  const h = trimmed.startsWith("#") ? trimmed.slice(1) : trimmed;

  if (!/^[0-9a-fA-F]{6}$/.test(h)) return undefined;

  const parsed = parseInt(h, 16);
  if (Number.isNaN(parsed)) return undefined;

  return parsed;
}

// -----------------------------------------
// itemType → measurements 組み立て
// -----------------------------------------
function buildMeasurements(
  itemType: ItemType,
  size: SizeRow,
): NewModelVariationMeasurements {
  const result: NewModelVariationMeasurements = {};

  if (itemType === "ボトムス") {
    result["ウエスト"] = size.waist ?? null;
    result["ヒップ"] = size.hip ?? null;
    result["股上"] = size.rise ?? null;
    result["股下"] = size.inseam ?? null;
    result["わたり幅"] = size.thighWidth ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  // トップス
  result["着丈"] = size.lengthTop ?? null;
  result["身幅"] = size.bodyWidth ?? null;
  result["肩幅"] = size.shoulderWidth ?? null;
  result["袖丈"] = size.sleeveLength ?? null;

  return result;
}

// -----------------------------------------
// variations payload builder
// -----------------------------------------
function toNewModelVariationPayload(
  itemType: ItemType,
  sizeRow: SizeRow,
  base: {
    sizeLabel: string;
    color: string;
    modelNumber: string;
    createdBy: string;
    rgb?: number;
  },
): NewModelVariationPayload {
  const measurements = buildMeasurements(itemType, sizeRow);

  return {
    sizeLabel: base.sizeLabel,
    color: base.color,
    modelNumber: base.modelNumber,
    createdBy: base.createdBy,
    rgb: base.rgb,
    measurements,
  };
}

// -----------------------------------------
// 生レスポンス（PascalCase）型
// -----------------------------------------
type RawProductBlueprintDetailResponse = {
  ID: string;
  ProductName: string;
  CompanyID: string;
  BrandID: string;
  ItemType: string;
  Fit: string;
  Material: string;
  Weight: number;
  QualityAssurance?: string[];
  ProductIdTag?: { Type?: string } | null;
  AssigneeID?: string | null;
  CreatedBy?: string | null;
  CreatedAt?: string | null;
  UpdatedBy?: string | null;
  UpdatedAt?: string | null;
  DeletedBy?: string | null;
  DeletedAt?: string | null;
};

// -----------------------------------------
// ブランド名取得ヘルパー（service 内実装）
// -----------------------------------------
async function fetchBrandNameById(brandId: string): Promise<string> {
  const id = brandId.trim();
  if (!id) return "";
  try {
    const brands = await fetchAllBrandsForCompany("", false);
    const hit = brands.find((b) => b.id === id);
    return hit?.name ?? "";
  } catch (e) {
    console.error(
      "[productBlueprintDetailService] fetchBrandNameById error:",
      e,
    );
    return "";
  }
}

// -----------------------------------------
// メンバーID → 表示名 解決ヘルパー
// （assigneeId / createdBy 共通で利用）
// -----------------------------------------
async function resolveMemberNameById(
  idToken: string,
  memberId?: string | null,
  fallback: string = "-",
): Promise<string> {
  const id = String(memberId ?? "").trim();
  if (!id) return fallback;

  try {
    const member = await fetchMemberByIdWithToken(idToken, id);
    const displayName = member
      ? formatLastFirst(member.lastName, member.firstName)
      : "";
    const name = displayName.trim() || id;
    return name || fallback;
  } catch (e) {
    console.error(
      "[productBlueprintDetailService] resolveMemberNameById error:",
      e,
    );
    return fallback;
  }
}

// -----------------------------------------
// GET: 商品設計 詳細（blueprintId で取得）
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/product-blueprints/${id}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!res.ok) {
    let detail: unknown;
    try {
      detail = await res.json();
    } catch {
      /* ignore */
    }
    throw new Error(
      `商品設計詳細の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as RawProductBlueprintDetailResponse;

  console.log(
    "[productBlueprintDetailService] GET raw detail response:",
    raw,
  );

  // ProductBlueprintDetailResponse に brandName / assigneeName / createdByName を“追加”した形で返す
  const response: ProductBlueprintDetailResponse & {
    brandName?: string;
    assigneeName?: string;
    createdByName?: string;
  } = {
    id: raw.ID,
    productName: raw.ProductName,
    companyId: raw.CompanyID,
    brandId: raw.BrandID,
    itemType: raw.ItemType,
    fit: raw.Fit,
    material: raw.Material,
    weight: raw.Weight,
    qualityAssurance: raw.QualityAssurance ?? [],
    productIdTag: raw.ProductIdTag
      ? { type: raw.ProductIdTag.Type ?? "" }
      : undefined,
    assigneeId: raw.AssigneeID ?? "",
    createdBy: raw.CreatedBy ?? "",
    createdAt: raw.CreatedAt ?? "",
  };

  // ブランド名変換
  const brandId = response.brandId ?? "";
  const brandName = brandId ? await fetchBrandNameById(brandId) : "";
  response.brandName = brandName;

  // 担当者名変換 (assigneeId -> displayName)
  const assigneeName = await resolveMemberNameById(
    idToken,
    response.assigneeId,
    "-",
  );
  response.assigneeName = assigneeName;

  // 作成者名変換 (createdBy -> displayName)
  const createdByName = await resolveMemberNameById(
    idToken,
    response.createdBy,
    "作成者未設定",
  );
  response.createdByName = createdByName;

  console.log("[productBlueprintDetailService] resolved names:", {
    brandId,
    brandName,
    assigneeId: response.assigneeId,
    assigneeName,
    createdById: response.createdBy,
    createdByName,
  });

  console.log(
    "[productBlueprintDetailService] GET mapped detail response:",
    response,
  );

  return response;
}

// -----------------------------------------
// UPDATE: 商品設計 更新
// -----------------------------------------
export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};
  const itemType = params.itemType as ItemType;

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      const sizeRow = params.sizes.find((s: SizeRow) => s.sizeLabel === v.size);
      if (!sizeRow) continue;

      const hex = colorRgbMap[v.color];
      const rgbInt = hexToRgbInt(hex);

      variations.push(
        toNewModelVariationPayload(itemType, sizeRow, {
          sizeLabel: v.size,
          color: v.color,
          modelNumber: v.code,
          createdBy: params.updatedBy ?? "",
          rgb: rgbInt,
        }),
      );
    }
  }

  console.log("[productBlueprintDetailService] UPDATE params:", params);
  console.log(
    "[productBlueprintDetailService] UPDATE variations payload:",
    variations,
  );

  const response = await updateProductBlueprintApi(params, variations);

  console.log(
    "[productBlueprintDetailService] UPDATE result response:",
    response,
  );

  return response;
}
