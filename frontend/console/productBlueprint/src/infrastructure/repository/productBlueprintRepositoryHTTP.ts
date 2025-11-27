// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// application 層の型だけを type import
import type {
  CreateProductBlueprintParams,
} from "../../application/productBlueprintCreateService";

import type {
  UpdateProductBlueprintParams,
  ProductBlueprintDetailResponse,
} from "../../infrastructure/api/productBlueprintDetailApi";

// BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)
    ?.replace(/\/+$/g, "") ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// -----------------------------------------------------------
// POST: 商品設計 作成
// -----------------------------------------------------------
export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");

  const idToken = await user.getIdToken();

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
  };

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  // ★ 詳細レスポンスとして返す
  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（論理削除されていないもの）
// -----------------------------------------------------------
export async function listProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/product-blueprints`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!res.ok) {
    throw new Error(
      `商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse[];
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（論理削除済みのみ）
//   - backend 側の GET /product-blueprints/deleted を想定
//   - 返却型は Deleted 用 service 側でキャストして利用する
// -----------------------------------------------------------
export async function listDeletedProductBlueprintsHTTP(): Promise<any[]> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/product-blueprints/deleted`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!res.ok) {
    throw new Error(
      `削除済み商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as any[];
}

// -----------------------------------------------------------
// PUT/PATCH: 商品設計 更新
// -----------------------------------------------------------
export async function updateProductBlueprintHTTP(
  id: string,
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");

  const idToken = await user.getIdToken();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify(params),
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品設計の更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  // ★ 返り値を ProductBlueprintDetailResponse に統一
  return (await res.json()) as ProductBlueprintDetailResponse;
}
