// frontend/console/productBlueprint/src/infrastructure/repository/productBlueprintRepositoryHTTP.ts

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// application 層の型だけを type import で参照（実行時の循環依存を避ける）
import type {
  CreateProductBlueprintParams,
  ProductBlueprintResponse,
  NewModelVariationPayload,
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

/**
 * HTTP リポジトリ:
 *   POST /product-blueprints
 *
 * - Firebase Auth の ID トークンを自前で取得
 * - Backend からの JSON をそのまま返す
 * - productId の解釈や ModelVariation 生成などのビジネスロジックは
 *   application 層（productBlueprintCreateService）側に任せる。
 */
export async function createProductBlueprintHTTP(
  params: CreateProductBlueprintParams,
): Promise<ProductBlueprintResponse> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  // backend に渡すペイロード
  // ここではフロントの状態をほぼそのまま JSON にして送る。
  // backend 側の handler / adapter で domain.ProductBlueprint へマッピングする想定。
  const payload = {
    productName: params.productName,
    brandId: params.brandId,
    itemType: params.itemType,
    // backend: Fit, Material, Weight, QualityAssurance
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance,

    // backend の ProductIDTag 構造に合わせてそのまま送信
    productIdTag: params.productIdTag,

    // backend: CompanyID に対応
    companyId: params.companyId,

    // backend: AssigneeID（null の場合は usecase 側で補完してもよい）
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
    // backend が { error: string } を返してくる想定
    let detail: unknown;
    try {
      detail = await res.json();
    } catch {
      // ignore json parse error
    }
    console.error(
      "[productBlueprintRepositoryHTTP] POST /product-blueprints failed",
      {
        status: res.status,
        statusText: res.statusText,
        detail,
      },
    );
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const json = (await res.json()) as ProductBlueprintResponse;
  return json;
}

// ------------------------------
// HTTP: ModelVariation 作成（将来用）
// ------------------------------

/**
 * CreateModelVariation (POST /models/{productID}/variations) を叩く HTTP ヘルパー。
 *
 * - 現時点では application 層からは未使用だが、
 *   将来 ProductBlueprint 作成後にモデルも同時作成する際に利用する想定。
 */
export async function createModelVariationHTTP(
  productId: string,
  variation: NewModelVariationPayload,
): Promise<void> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  const res = await fetch(`${API_BASE}/models/${productId}/variations`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${idToken}`,
    },
    body: JSON.stringify(variation),
  });

  if (!res.ok) {
    let detail: unknown;
    try {
      detail = await res.json();
    } catch {
      // ignore json parse error
    }
    console.error(
      "[productBlueprintRepositoryHTTP] CreateModelVariation failed",
      {
        status: res.status,
        statusText: res.statusText,
        detail,
      },
    );
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }
}
