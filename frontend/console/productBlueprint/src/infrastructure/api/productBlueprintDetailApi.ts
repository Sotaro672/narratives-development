// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintDetailApi.ts

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { fetchJSON } from "../../../../shell/src/shared/http/fetchJSON";

import type {
  ApparelModelNumberRow,
  ApparelSizeInput,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

// ------------------------------------------------------
// apparel model variation 共通型
// ------------------------------------------------------

export type ApparelModelVariationResponse = {
  id?: string;
  size?: string;
  color?: string;
  modelNumber?: string;
  rgb?: number | null;
  measurements?: Record<string, number | null>;
  productBlueprintId?: string;
  version?: number;
  createdAt?: string | null;
  updatedAt?: string | null;
};

export type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
};

// ------------------------------------------------------
// ProductBlueprint 詳細レスポンス
// ------------------------------------------------------

export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;

  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  /**
   * 旧固定 field。
   *
   * backend 新仕様では categoryFields 側へ寄せていくが、
   * 既存画面・既存データ互換のため response 型としては残す。
   */
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTag?: {
    type?: string | null;
  } | null;

  companyId?: string;
  assigneeId?: string;

  printed?: boolean | null;

  brandName?: string | null;
  assigneeName?: string | null;
  createdByName?: string | null;

  createdBy?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
  deletedAt?: string | null;

  modelRefs?: ProductBlueprintModelRef[];

  modelVariations?: ApparelModelVariationResponse[];

  /**
   * backend ProductBlueprint.CategoryFields。
   *
   * brandId / productName / productIdTagType / description は含めない。
   * color / size / measurements も model variation 側なので含めない。
   */
  categoryFields?: CategoryFieldValues | null;
};

// ------------------------------------------------------
// 更新用パラメータ
// ------------------------------------------------------

export type UpdateProductBlueprintParams = {
  id: string;

  productName: string;
  brandId: string;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  /**
   * 旧固定 field。
   *
   * 新仕様では categoryFields に寄せるが、
   * 既存 UI / repository の段階移行のため optional として残す。
   */
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;

  productIdTagType: string | null;

  companyId: string;
  assigneeId: string;

  colors: string[];
  colorRgbMap?: Record<string, string>;

  sizes?: ApparelSizeInput[];
  modelNumbers?: ApparelModelNumberRow[];
  updatedBy?: string | null;

  categoryFields?: CategoryFieldValues | null;
};

export type { ProductBlueprintCategoryKind, ProductBlueprintCategorySnapshot };

// ------------------------------------------------------
// raw response helpers
// ------------------------------------------------------

type RawProductBlueprintCategorySnapshot =
  Partial<ProductBlueprintCategorySnapshot> & {
    ID?: string | null;
    Code?: string | null;
    NameJa?: string | null;
    NameEn?: string | null;
    Kind?: ProductBlueprintCategoryKind | string | null;
    Path?: string[] | null;
  };

function normalizeProductBlueprintCategoryKind(
  value: unknown,
): ProductBlueprintCategoryKind {
  const kind = String(value ?? "").trim();

  switch (kind) {
    case "apparel":
    case "alcohol":
    case "cosmetics":
    case "healthcare":
    case "other":
      return kind;
    default:
      return "other";
  }
}

function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => String(item ?? "").trim())
    .filter((item) => item !== "");
}

function normalizeProductBlueprintCategorySnapshot(
  raw: RawProductBlueprintCategorySnapshot | null | undefined,
): ProductBlueprintCategorySnapshot {
  const id = String(raw?.id ?? raw?.ID ?? "").trim();
  const code = String(raw?.code ?? raw?.Code ?? "").trim();

  return {
    id,
    code,
    nameJa: String(raw?.nameJa ?? raw?.NameJa ?? "").trim(),
    nameEn: String(raw?.nameEn ?? raw?.NameEn ?? "").trim(),
    kind: normalizeProductBlueprintCategoryKind(raw?.kind ?? raw?.Kind),
    path:
      Array.isArray(raw?.path) && raw.path.length > 0
        ? normalizeStringArray(raw.path)
        : normalizeStringArray(raw?.Path),
  };
}

function resolveProductBlueprintCategoryId(args: {
  productBlueprintCategoryId?: string | null;
  productBlueprintCategory?: RawProductBlueprintCategorySnapshot | null;
}): string {
  const fromField = String(args.productBlueprintCategoryId ?? "").trim();

  if (fromField) {
    return fromField;
  }

  return String(
    args.productBlueprintCategory?.id ??
      args.productBlueprintCategory?.ID ??
      args.productBlueprintCategory?.code ??
      args.productBlueprintCategory?.Code ??
      "",
  ).trim();
}

// ------------------------------------------------------
// category kind helpers
// ------------------------------------------------------

export function getProductBlueprintDetailCategoryKind(
  detail: ProductBlueprintDetailResponse | null | undefined,
): ProductBlueprintCategoryKind | null {
  return detail?.productBlueprintCategory?.kind ?? null;
}

export function isApparelProductBlueprintDetail(
  detail: ProductBlueprintDetailResponse | null | undefined,
): boolean {
  return getProductBlueprintDetailCategoryKind(detail) === "apparel";
}

export function isAlcoholProductBlueprintDetail(
  detail: ProductBlueprintDetailResponse | null | undefined,
): boolean {
  return getProductBlueprintDetailCategoryKind(detail) === "alcohol";
}

export function isCosmeticsProductBlueprintDetail(
  detail: ProductBlueprintDetailResponse | null | undefined,
): boolean {
  return getProductBlueprintDetailCategoryKind(detail) === "cosmetics";
}

export function isHealthcareProductBlueprintDetail(
  detail: ProductBlueprintDetailResponse | null | undefined,
): boolean {
  return getProductBlueprintDetailCategoryKind(detail) === "healthcare";
}

export function isOtherProductBlueprintDetail(
  detail: ProductBlueprintDetailResponse | null | undefined,
): boolean {
  return getProductBlueprintDetailCategoryKind(detail) === "other";
}

// ------------------------------------------------------
// GET /product-blueprints/{id} 詳細取得
// ------------------------------------------------------

export async function getProductBlueprintDetailApi(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const headers = await getAuthHeaders();

  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error("getProductBlueprintDetailApi: id が空です");
  }

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}`;

  const json = await fetchJSON<ProductBlueprintDetailResponse>(url, {
    method: "GET",
    headers,
  });

  const rawCategory =
    json.productBlueprintCategory as RawProductBlueprintCategorySnapshot;

  return {
    ...json,
    productBlueprintCategoryId: resolveProductBlueprintCategoryId({
      productBlueprintCategoryId: json.productBlueprintCategoryId,
      productBlueprintCategory: rawCategory,
    }),
    productBlueprintCategory:
      normalizeProductBlueprintCategorySnapshot(rawCategory),
    fit: json.fit ?? null,
    material: json.material ?? null,
    weight: json.weight ?? null,
    qualityAssurance: json.qualityAssurance ?? [],
    categoryFields: json.categoryFields ?? null,
    modelRefs: json.modelRefs ?? [],
    modelVariations: json.modelVariations ?? [],
  };
}