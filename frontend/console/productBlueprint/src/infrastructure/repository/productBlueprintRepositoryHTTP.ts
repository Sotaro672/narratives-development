// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import {
  getAuthHeadersOrThrow,
  getAuthJsonHeadersOrThrow,
} from "../../../../shell/src/shared/http/authHeaders";

// application 層の型だけを type import
import type { CreateProductBlueprintParams } from "../../application/productBlueprintCreateService";

import type {
  UpdateProductBlueprintParams,
  ProductBlueprintDetailResponse,
} from "../api/productBlueprintDetailApi";

// -----------------------------------------------------------
// POST: 商品設計 作成
// -----------------------------------------------------------
export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const headers = await getAuthJsonHeadersOrThrow();

  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    itemType: params.itemType,
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance,
    productIdTag: params.productIdTag,
    companyId: params.companyId,
    assigneeId: params.assigneeId ?? null,
    createdBy: params.createdBy ?? null,
    // printed は backend 側で false を設定する想定なので送らない
  };

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "POST",
    headers,
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// POST: 商品設計に modelRefs（modelIds）を追記（updatedAt/updatedBy を触らない）
//   - backend: POST /product-blueprints/{id}/model-refs
//   - body: { modelIds: string[] }
//   - resp: detail（toDetailOutput）
// -----------------------------------------------------------
export async function appendModelRefsHTTP(
  productBlueprintId: string,
  modelIds: string[],
): Promise<ProductBlueprintDetailResponse> {
  const trimmedId = String(productBlueprintId ?? "").trim();
  if (!trimmedId) {
    throw new Error("appendModelRefsHTTP: productBlueprintId が空です");
  }

  // trim + 空除外 + 重複除外（順序保持）
  const seen = new Set<string>();
  const cleaned: string[] = [];
  for (const raw of modelIds ?? []) {
    const v = String(raw ?? "").trim();
    if (!v) continue;
    if (seen.has(v)) continue;
    seen.add(v);
    cleaned.push(v);
  }

  if (cleaned.length === 0) {
    throw new Error("appendModelRefsHTTP: modelIds が空です");
  }

  const headers = await getAuthJsonHeadersOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(
    trimmedId,
  )}/model-refs`;

  const res = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify({ modelIds: cleaned }),
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `modelRefs の追記に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（論理削除されていないもの）
// -----------------------------------------------------------
export async function listProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    throw new Error(
      `商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse[];
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（printed == true のみ）
//   - backend: GET /product-blueprints/printed
// -----------------------------------------------------------
export async function listPrintedProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/product-blueprints/printed`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    throw new Error(
      `印刷済みの商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse[];
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（論理削除済みのみ）
//   - backend: GET /product-blueprints/deleted
// -----------------------------------------------------------
export async function listDeletedProductBlueprintsHTTP(): Promise<any[]> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/product-blueprints/deleted`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    throw new Error(
      `削除済み商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as any[];
}

// -----------------------------------------------------------
// PATCH: 商品設計 更新（DTO を正にして送る）
//   - backend: PATCH /product-blueprints/{id}
//   - ✅ variations/colors 等は送らない（別サービスで更新）
//   - ✅ productIdTagType は productIdTag に変換して送る
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

  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    itemType: params.itemType,
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance ?? [],
    productIdTag: {
      type: params.productIdTagType ?? "",
    },
    companyId: params.companyId,
    assigneeId: params.assigneeId,
    updatedBy: params.updatedBy ?? null,
    // printed は更新させない（印刷済み化は専用 endpoint）
    // variations / colors / sizes / modelNumbers も送らない
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

// -----------------------------------------------------------
// POST: 商品設計 復旧（deletedAt / deletedBy / expireAt をクリア）
//   - backend: POST /product-blueprints/{id}/restore
// -----------------------------------------------------------
export async function restoreProductBlueprintHTTP(id: string): Promise<void> {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) {
    throw new Error("restoreProductBlueprintHTTP: id が空です");
  }

  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}/restore`,
    {
      method: "POST",
      headers,
    },
  );

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品設計の復旧に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }
}
