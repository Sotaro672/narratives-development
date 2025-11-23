// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTag } from "../domain/entity/productBlueprint";

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
 *
 * backend/internal/domain/productBlueprint.ProductBlueprint に対応:
 *
 *   type ProductBlueprint struct {
 *     ID               string
 *     ProductName      string
 *     BrandID          string
 *     ItemType         ItemType
 *     VariationIDs     []string
 *     Fit              string
 *     Material         string
 *     Weight           float64
 *     QualityAssurance []string
 *     ProductIdTag     ProductIDTag
 *     CompanyID        string
 *     AssigneeID       string
 *     CreatedBy        *string
 *     CreatedAt        time.Time
 *     UpdatedBy        *string
 *     UpdatedAt        time.Time
 *     DeletedBy        *string
 *     DeletedAt        *time.Time
 *   }
 *
 * - ここでは ID / CreatedAt などは backend で採番・設定される前提。
 * - VariationIDs は model / size などから組み立てて渡す想定のため optional。
 * - CompanyID は currentMember などからフロントで取得して渡す。
 */
export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[]; // WASH_TAG_OPTIONS に対応

  /** backend: ProductIDTag に対応（type + logoDesignFile） */
  productIdTag: ProductIDTag;

  /** backend: VariationIDs に対応（Model 側で生成した ID 群） */
  variationIds?: string[];

  /** backend: CompanyID に対応（currentMember.companyId などから取得） */
  companyId: string;

  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  // 担当者など、必要に応じて付加（backend: AssigneeID）
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
    // backend: Fit, Material, Weight, QualityAssurance
    fit: params.fit,
    material: params.material,
    weight: params.weight,
    qualityAssurance: params.qualityAssurance,

    // backend の ProductIDTag 構造に合わせてそのまま送信
    productIdTag: params.productIdTag,

    // backend: VariationIDs に対応（未指定なら空配列）
    variationIds: params.variationIds ?? [],

    // backend: CompanyID に対応
    companyId: params.companyId,

    // モデル生成用の補助情報（colors / sizes / modelNumbers）は
    // backend の usecase 側で解釈して利用する想定
    colors: params.colors,
    sizes: params.sizes,
    modelNumbers: params.modelNumbers,

    // backend: AssigneeID（null の場合は usecase 側で補完してもよい）
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
