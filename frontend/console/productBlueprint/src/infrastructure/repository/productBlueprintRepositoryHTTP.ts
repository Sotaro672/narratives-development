// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// application 層の型だけを type import で参照（実行時の循環依存を避ける）
import type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
} from "../../application/productBlueprintCreateService";

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ------------------------------
// HTTP: ProductBlueprint 作成
// ------------------------------

export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {

  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

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
    let detail: unknown;
    try {
      detail = await res.json();
    } catch {
      /* ignore */
    }
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const json = (await res.json()) as ProductBlueprintResponse;
  return json;
}

// ------------------------------
// HTTP: ProductBlueprint 一覧取得
// ------------------------------

export async function listProductBlueprintsHTTP(): Promise<
  ProductBlueprintResponse[]
> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/product-blueprints`, {
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
      `商品設計一覧の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const json = (await res.json()) as ProductBlueprintResponse[];
  return json;
}
