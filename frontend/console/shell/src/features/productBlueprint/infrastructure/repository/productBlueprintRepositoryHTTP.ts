// frontend/console/shell/src/features/productBlueprint/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";
import type { CreateProductBlueprintParams } from "../../application/productBlueprintCreateService";
import type { ProductBlueprintDetailResponse } from "../api/productBlueprintDetailApi";
import type { UpdateProductBlueprintParams } from "../api/productBlueprintUpdateApi";

// -----------------------------------------------------------
// Response types
// -----------------------------------------------------------

/**
 * GET /product-blueprints のBFFレスポンス。
 * backendで名前解決・日時変換済みの値をそのまま使用する。
 */
export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandName: string;
  assigneeName: string;
  printed: boolean;
  createdByName: string;
  updatedByName: string;
  createdAt: string;
  updatedAt: string;
};

// -----------------------------------------------------------
// Request payload helpers
// -----------------------------------------------------------

function assertProductBlueprintCategoryPayload(params: {
  productBlueprintCategoryId: string;
  productBlueprintCategory: { id?: string } | null | undefined;
}): void {
  if (!params.productBlueprintCategoryId) {
    throw new Error("productBlueprintRepositoryHTTP: productBlueprintCategoryId が空です");
  }

  if (!params.productBlueprintCategory?.id) {
    throw new Error("productBlueprintRepositoryHTTP: productBlueprintCategory が空です");
  }

  if (params.productBlueprintCategoryId !== params.productBlueprintCategory.id) {
    throw new Error(
      "productBlueprintRepositoryHTTP: productBlueprintCategoryId と productBlueprintCategory.id が一致しません",
    );
  }
}

function buildProductBlueprintCategoryPayload(params: {
  productBlueprintCategoryId: string;
  productBlueprintCategory: { id?: string } | null | undefined;
  categoryFields?: unknown;
}) {
  assertProductBlueprintCategoryPayload(params);

  return {
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields ?? null,
  };
}

function buildProductIdTagPayload(
  type: string | null | undefined,
): { type: string } {
  return {
    type: type ?? "",
  };
}

// -----------------------------------------------------------
// POST: 商品設計 作成
// -----------------------------------------------------------

export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const authHeaders = await getAuthHeaders();

  const categoryPayload = buildProductBlueprintCategoryPayload({
    productBlueprintCategoryId: params.productBlueprintCategoryId,
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields,
  });

  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    ...categoryPayload,
    productIdTag: params.productIdTag,
    assigneeId: params.assigneeId ?? "",
  };

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "POST",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");

    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// GET: 商品設計 一覧
// -----------------------------------------------------------

export async function listProductBlueprintsHTTP(): Promise<ProductBlueprintListRow[]> {
  const headers = await getAuthHeaders();

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");

    throw new Error(
      `商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductBlueprintListRow[];
}

// -----------------------------------------------------------
// PATCH: 商品設計 更新
// -----------------------------------------------------------

export async function updateProductBlueprintHTTP(
  id: string,
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  if (!id) {
    throw new Error("updateProductBlueprintHTTP: id が空です");
  }

  const authHeaders = await getAuthHeaders();
  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`;

  const categoryPayload = buildProductBlueprintCategoryPayload({
    productBlueprintCategoryId: params.productBlueprintCategoryId,
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields,
  });

  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    ...categoryPayload,
    productIdTag: buildProductIdTagPayload(params.productIdTagType),
    assigneeId: params.assigneeId,
  };

  const res = await fetch(url, {
    method: "PATCH",
    headers: {
      ...authHeaders,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");

    throw new Error(
      `商品設計の更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// DELETE: 商品設計 物理削除
// -----------------------------------------------------------

export async function deleteProductBlueprintHTTP(id: string): Promise<void> {
  if (!id) {
    throw new Error("deleteProductBlueprintHTTP: id が空です");
  }

  const headers = await getAuthHeaders();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
      headers,
    },
  );

  if (!res.ok) {
    const detail = await res.text().catch(() => "");

    throw new Error(
      `商品設計の削除に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }
}

// -----------------------------------------------------------
// POST: 商品設計 printed フラグ更新（false → true）
// -----------------------------------------------------------

export async function markProductBlueprintPrintedHTTP(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  if (!id) {
    throw new Error("markProductBlueprintPrintedHTTP: id が空です");
  }

  const headers = await getAuthHeaders();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(id)}/mark-printed`,
    {
      method: "POST",
      headers,
    },
  );

  if (!res.ok) {
    const detail = await res.text().catch(() => "");

    throw new Error(
      `商品設計のprinted更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse;
}