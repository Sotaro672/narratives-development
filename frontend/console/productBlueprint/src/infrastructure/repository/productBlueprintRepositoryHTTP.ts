// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// application 層の型だけを type import
import type { CreateProductBlueprintParams } from "../../application/productBlueprintCreateService";

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
// 共通: Firebase 認証トークン取得
// -----------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

// -----------------------------------------------------------
// POST: 商品設計 作成
// -----------------------------------------------------------
export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const idToken = await getIdTokenOrThrow();

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
    // printed は backend 側でデフォルト "notYet" を設定する想定なので、
    // フロントからは明示的には渡さない（必要になればここに追加）
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

  // ★ 詳細レスポンスとして返す（printed を含む想定）
  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（論理削除されていないもの）
// -----------------------------------------------------------
export async function listProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const idToken = await getIdTokenOrThrow();

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
// GET: 商品設計 一覧（printed == notYet）
//   - backend: GET /product-blueprints/printed/notYet
// -----------------------------------------------------------
export async function listNotYetPrintedProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const idToken = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/printed/notYet`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  if (!res.ok) {
    throw new Error(
      `未印刷の商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  return (await res.json()) as ProductBlueprintDetailResponse[];
}

// -----------------------------------------------------------
// GET: 商品設計 一覧（printed == printed）
//   - backend: GET /product-blueprints/printed/printed
// -----------------------------------------------------------
export async function listPrintedProductBlueprintsHTTP(): Promise<ProductBlueprintDetailResponse[]> {
  const idToken = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/printed/printed`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  if (!res.ok) {
    throw new Error(
      `印刷済みの商品設計一覧の取得に失敗しました（${res.status} ${res.statusText}）`,
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
  const idToken = await getIdTokenOrThrow();

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
// PUT: 商品設計 更新
// -----------------------------------------------------------
export async function updateProductBlueprintHTTP(
  id: string,
  params: UpdateProductBlueprintParams,
): Promise<ProductBlueprintDetailResponse> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    // params 内に printed があれば、そのまま backend に渡される
    body: JSON.stringify(params),
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品設計の更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  // ★ 返り値を ProductBlueprintDetailResponse に統一（printed を含む想定）
  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// POST: 商品設計 printed フラグ更新（notYet → printed）
//   - backend: POST /product-blueprints/{id}/mark-printed
// -----------------------------------------------------------
export async function markProductBlueprintPrintedHTTP(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const trimmed = id?.trim();
  if (!trimmed) {
    throw new Error("markProductBlueprintPrintedHTTP: id が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}/mark-printed`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品設計のprinted更新に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  // updated な ProductBlueprint（printed: "printed" を含む）を受け取る想定
  return (await res.json()) as ProductBlueprintDetailResponse;
}

// -----------------------------------------------------------
// POST: 商品設計 復旧（deletedAt / deletedBy / expireAt をクリア）
//   - backend 側の POST /product-blueprints/{id}/restore を想定
//   - 戻り値は特に使わない前提なので void で定義
// -----------------------------------------------------------
export async function restoreProductBlueprintHTTP(id: string): Promise<void> {
  const trimmed = id?.trim();
  if (!trimmed) {
    throw new Error("restoreProductBlueprintHTTP: id が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const res = await fetch(
    `${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}/restore`,
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品設計の復旧に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }
}
