// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintDetailApi.ts

import { getAuthHeaders } from "../../../../shell/src/auth/application/authService";
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { fetchJSON } from "../../../../shell/src/shared/http/fetchJSON";

import type {
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

export type { UpdateProductBlueprintParams } from "./productBlueprintUpdateApi";

export type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder: number;
};

export type ProductBlueprintModelVariationResponse = {
  id?: string;
  productBlueprintId?: string;
  kind?: "apparel" | "alcohol" | string;

  modelNumber?: string;

  size?: string;
  color?: string | { name?: string; rgb?: number | null };
  rgb?: number | null;
  measurements?: Record<string, number | null>;

  volume?: {
    value?: number | null;
    unit?: string | null;
  } | null;

  version?: number;
  createdAt?: string | null;
  updatedAt?: string | null;
};

type ProductBlueprintCategorySnapshotRaw =
  | ProductBlueprintCategorySnapshot
  | {
      ID?: string;
      Code?: string;
      NameJa?: string;
      NameEn?: string;
      Kind?: ProductBlueprintCategoryKind;
      Path?: string[];
    }
  | null
  | undefined;

export type ProductBlueprintDetailResponse = {
  id: string;
  productName: string;
  description?: string | null;

  companyId?: string;
  brandId: string;
  brandName?: string | null;

  productBlueprintCategoryId: string;
  productBlueprintCategory: ProductBlueprintCategorySnapshot;

  categoryFields?: CategoryFieldValues | null;

  productIdTag?: {
    type?: string | null;
  } | null;

  assigneeId?: string;
  assigneeName?: string | null;

  printed?: boolean | null;

  createdBy?: string | null;
  createdByName?: string | null;
  createdAt?: string | null;

  updatedBy?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;

  deletedAt?: string | null;

  modelRefs?: ProductBlueprintModelRef[];
  modelVariations?: ProductBlueprintModelVariationResponse[];
};

type ProductBlueprintDetailRawResponse = Omit<
  ProductBlueprintDetailResponse,
  "productBlueprintCategory"
> & {
  productBlueprintCategory: ProductBlueprintCategorySnapshotRaw;
};

export type { ProductBlueprintCategoryKind, ProductBlueprintCategorySnapshot };

function normalizeProductBlueprintCategorySnapshot(
  raw: ProductBlueprintCategorySnapshotRaw,
): ProductBlueprintCategorySnapshot {
  const record = (raw ?? {}) as Record<string, unknown>;

  const id = String(record.id ?? record.ID ?? "");
  const code = String(record.code ?? record.Code ?? id);
  const nameJa = String(record.nameJa ?? record.NameJa ?? "");
  const nameEn = String(record.nameEn ?? record.NameEn ?? "");
  const kind = String(
    record.kind ?? record.Kind ?? "",
  ) as ProductBlueprintCategoryKind;

  const rawPath = record.path ?? record.Path;
  const path = Array.isArray(rawPath)
    ? rawPath.map((value) => String(value)).filter(Boolean)
    : code
      ? code.split(".").filter(Boolean)
      : [];

  return {
    id,
    code,
    nameJa,
    nameEn,
    kind,
    path,
  };
}

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

export async function getProductBlueprintDetailApi(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const headers = await getAuthHeaders();

  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error("getProductBlueprintDetailApi: id が空です");
  }

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}`;

  const json = await fetchJSON<ProductBlueprintDetailRawResponse>(url, {
    method: "GET",
    headers,
  });

  return {
    ...json,
    productBlueprintCategory: normalizeProductBlueprintCategorySnapshot(
      json.productBlueprintCategory,
    ),
    categoryFields: json.categoryFields ?? null,
    productIdTag: json.productIdTag ?? null,
    modelRefs: json.modelRefs ?? [],
    modelVariations: json.modelVariations ?? [],
  };
}