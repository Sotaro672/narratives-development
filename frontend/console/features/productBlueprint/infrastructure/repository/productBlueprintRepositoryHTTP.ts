// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

// application 層の型だけを type import
import type { CreateProductBlueprintParams } from "../../application/productBlueprintCreateService";

import type { ProductBlueprintDetailResponse } from "../api/productBlueprintDetailApi";
import type { UpdateProductBlueprintParams } from "../api/productBlueprintUpdateApi";

// -----------------------------------------------------------
// internal helpers
// -----------------------------------------------------------

function assertProductBlueprintCategoryPayload(params: {
  productBlueprintCategoryId: string;
  productBlueprintCategory: unknown;
}) {
  const productBlueprintCategoryId = String(
    params.productBlueprintCategoryId ?? "",
  ).trim();

  if (!productBlueprintCategoryId) {
    throw new Error(
      "productBlueprintRepositoryHTTP: productBlueprintCategoryId が空です",
    );
  }

  if (!params.productBlueprintCategory) {
    throw new Error(
      "productBlueprintRepositoryHTTP: productBlueprintCategory が空です",
    );
  }
}

function buildProductBlueprintCategoryPayload(params: {
  productBlueprintCategoryId: string;
  productBlueprintCategory: unknown;
  categoryFields?: unknown;
}) {
  assertProductBlueprintCategoryPayload(params);

  return {
    productBlueprintCategoryId: params.productBlueprintCategoryId.trim(),
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields ?? null,
  };
}

function buildProductIdTagPayload(type: string | null | undefined): {
  type: string;
} {
  return {
    type: String(type ?? "").trim(),
  };
}

// -----------------------------------------------------------
// POST: 商品設計 作成
// -----------------------------------------------------------

export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const headers = await getAuthJsonHeadersOrThrow();

  const categoryPayload = buildProductBlueprintCategoryPayload({
    productBlueprintCategoryId: params.productBlueprintCategoryId,
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields,
  });

  const payload = {
    productName: params.productName,
    brandId: params.brandId,

    // itemType は廃止。
    // productBlueprintCategoryId / productBlueprintCategory / categoryFields を正として送る。
    ...categoryPayload,

    productIdTag: params.productIdTag,
    companyId: params.companyId,
    assigneeId: params.assigneeId ?? null,
    createdBy: params.createdBy ?? null,

    // printed は backend 側で false を設定する想定なので送らない。
  };

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "POST",
    headers,
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

export async function listProductBlueprintsHTTP(): Promise<
  ProductBlueprintDetailResponse[]
> {
  const headers = await getAuthHeadersOrThrow();

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

  return (await res.json()) as ProductBlueprintDetailResponse[];
}

// -----------------------------------------------------------
// PATCH: 商品設計 更新
// -----------------------------------------------------------

export async function updateProductBlueprintHTTP(
  id: string,
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const trimmedId = String(id ?? "").trim();
  if (!trimmedId) {
    throw new Error("updateProductBlueprintHTTP: id が空です");
  }

  const headers = await getAuthJsonHeadersOrThrow();
  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(trimmedId)}`;

  const categoryPayload = buildProductBlueprintCategoryPayload({
    productBlueprintCategoryId: params.productBlueprintCategoryId,
    productBlueprintCategory: params.productBlueprintCategory,
    categoryFields: params.categoryFields,
  });

  const payload = {
    productName: params.productName,
    brandId: params.brandId,

    // itemType は廃止。
    // productBlueprintCategoryId / productBlueprintCategory / categoryFields を正として送る。
    ...categoryPayload,

    productIdTag: buildProductIdTagPayload(params.productIdTagType),
    companyId: params.companyId,
    assigneeId: params.assigneeId,
    updatedBy: params.updatedBy ?? null,
  };

  const res = await fetch(url, {
    method: "PATCH",
    headers,
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
// POST: 商品設計 printed フラグ更新（false → true）
//   - backend: POST /product-blueprints/{id}/mark-printed
// -----------------------------------------------------------

export async function markProductBlueprintPrintedHTTP(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error("markProductBlueprintPrintedHTTP: id が空です");
  }

  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}/mark-printed`,
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