// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTagType } from "../domain/entity/productBlueprint";

// Size / ModelNumber の型だけ借りる
import type { SizeRow } from "../../../model/src/presentation/components/SizeVariationCard";
import type { ModelNumber } from "../../../model/src/presentation/components/ModelNumberCard";

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

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
// 型定義
// ------------------------------

/**
 * 商品設計作成で backend に渡すペイロード
 * （まずはフロントの状態をそのまま投げる DTO として定義）
 */
export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[]; // WASH_TAG_OPTIONS に対応
  productIdTagType: ProductIDTagType;

  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  // 担当者など、必要に応じて付加
  assigneeId?: string;
};

export type ProductBlueprintResponse = {
  id: string;
  // backend の ProductBlueprint ドメインをそのまま返してくる想定なので、
  // 他のフィールドはとりあえずゆるく許容しておく
  [key: string]: unknown;
};

// ------------------------------
// Service 本体
// ------------------------------

/**
 * 商品設計を作成する HTTP サービス
 * - POST /product-blueprints
 * - Firebase Auth の ID トークンを Authorization に付与
 */
export async function createProductBlueprint(
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
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance,
    productIdTagType: params.productIdTagType,
    colors: params.colors,
    sizes: params.sizes,
    modelNumbers: params.modelNumbers,
    assigneeId: params.assigneeId ?? null,
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
    console.error("[productBlueprintCreateService] POST failed", {
      status: res.status,
      statusText: res.statusText,
      detail,
    });
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText}）`,
    );
  }

  const json = (await res.json()) as ProductBlueprintResponse;
  return json;
}
