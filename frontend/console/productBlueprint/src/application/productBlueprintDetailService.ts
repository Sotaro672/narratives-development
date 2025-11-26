// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";

// ⭐ API レイヤーは廃止 → Repository HTTP を使用
import {
  updateProductBlueprintHTTP,
} from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import type {
  ProductBlueprintDetailResponse,
  UpdateProductBlueprintParams,
  NewModelVariationMeasurements,
  NewModelVariationPayload,
} from "../infrastructure/api/productBlueprintDetailApi"; // ← 型のみ維持

import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import { fetchAllBrandsForCompany } from "../../../brand/src/infrastructure/query/brandQuery";
import { formatLastFirst } from "../../../member/src/infrastructure/query/memberQuery";
import { MemberRepositoryHTTP } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";

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
    result["わたり幅"] = size.thigh ?? size.thighWidth ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  result["着丈"] = size.length ?? size.lengthTop ?? null;
  result["身幅"] = size.chest ?? size.bodyWidth ?? null;
  result["胸囲"] = size.chest ?? size.bodyWidth ?? null;
  result["肩幅"] = size.shoulder ?? size.shoulderWidth ?? null;
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
// ブランド名取得ヘルパー
// -----------------------------------------
async function fetchBrandNameById(brandId: string): Promise<string> {
  const id = brandId.trim();
  if (!id) return "";
  try {
    const brands = await fetchAllBrandsForCompany("", false);
    return brands.find((b) => b.id === id)?.name ?? "";
  } catch (e) {
    console.error("[productBlueprintDetailService] fetchBrandNameById error:", e);
    return "";
  }
}

// -----------------------------------------
// メンバー名解決（Repository 経由）
// -----------------------------------------
async function resolveMemberNameById(
  _idToken: string,
  memberId?: string | null,
  fallback: string = "-",
): Promise<string> {
  const id = String(memberId ?? "").trim();
  if (!id) return fallback;

  try {
    const repo = new MemberRepositoryHTTP();
    const member = await repo.getById(id);
    if (!member) return fallback;

    const name = formatLastFirst(member.lastName, member.firstName)?.trim() || id;

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
// GET: 商品設計 詳細
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const user = auth.currentUser;
  if (!user) throw new Error("ログイン情報が見つかりません（未ログイン）");

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/product-blueprints/${id}`, {
    method: "GET",
    headers: { Authorization: `Bearer ${idToken}` },
  });

  if (!res.ok) {
    throw new Error(
      `商品設計詳細の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as RawProductBlueprintDetailResponse;

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

  response.brandName = await fetchBrandNameById(response.brandId ?? "");
  response.assigneeName = await resolveMemberNameById(
    idToken,
    response.assigneeId,
    "-",
  );
  response.createdByName = await resolveMemberNameById(
    idToken,
    response.createdBy,
    "作成者未設定",
  );

  return response;
}

// -----------------------------------------
// UPDATE（Repository HTTP を使用）
// -----------------------------------------
export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const variations: NewModelVariationPayload[] = [];

  const colorRgbMap = params.colorRgbMap ?? {};
  const itemType = params.itemType as ItemType;

  if (params.modelNumbers && params.sizes) {
    for (const v of params.modelNumbers) {
      const sizeRow = params.sizes.find(
        (s: SizeRow) => s.sizeLabel === v.size,
      );
      if (!sizeRow) continue;

      const rgbInt = hexToRgbInt(colorRgbMap[v.color]);

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

  // ⭐ Repository 経由に一本化
  return await updateProductBlueprintHTTP(params.id, {
    ...params,
    variations,
  } as any);
}

// -----------------------------------------
// ModelVariation list
// -----------------------------------------
export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color?: { name: string; rgb?: number | null };
  measurements?: Record<string, number | null>;
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
};

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const id = productBlueprintId.trim();
  if (!id) throw new Error("productBlueprintId が空です");

  const user = auth.currentUser;
  if (!user) throw new Error("ログイン情報が見つかりません（未ログイン）");

  const idToken = await user.getIdToken();
  const url = `${API_BASE}/models/by-blueprint/${encodeURIComponent(
    id,
  )}/variations`;

  const res = await fetch(url, {
    method: "GET",
    headers: { Authorization: `Bearer ${idToken}`, Accept: "application/json" },
  });

  if (!res.ok) {
    throw new Error(
      `モデル一覧の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as any[] | null;
  if (!raw) return [];

  return raw.map((v: any) => {
    const colorRaw = v.color ?? v.Color ?? {};
    const measurementsRaw = v.measurements ?? v.Measurements ?? {};

    const rgbValue =
      typeof colorRaw.rgb === "number"
        ? colorRaw.rgb
        : typeof colorRaw.RGB === "number"
          ? colorRaw.RGB
          : null;

    return {
      id: v.id ?? v.ID ?? "",
      productBlueprintId: v.productBlueprintId ?? v.ProductBlueprintID ?? id,
      modelNumber: v.modelNumber ?? v.ModelNumber ?? "",
      size: v.size ?? v.Size ?? "",
      color: { name: colorRaw.name ?? colorRaw.Name ?? "", rgb: rgbValue },
      measurements:
        typeof measurementsRaw === "object"
          ? (measurementsRaw as Record<string, number | null>)
          : {},
      createdAt: v.createdAt ?? v.CreatedAt ?? null,
      createdBy: v.createdBy ?? v.CreatedBy ?? null,
      updatedAt: v.updatedAt ?? v.UpdatedAt ?? null,
      updatedBy: v.updatedBy ?? v.UpdatedBy ?? null,
      deletedAt: v.deletedAt ?? v.DeletedAt ?? null,
      deletedBy: v.deletedBy ?? v.DeletedBy ?? null,
    };
  });
}
