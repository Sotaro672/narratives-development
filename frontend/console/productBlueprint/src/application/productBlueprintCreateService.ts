// frontend/console/productBlueprint/src/application/productBlueprintCreateService.ts

import type { ItemType, Fit } from "../domain/entity/catalog";
import type { ProductIDTag } from "../domain/entity/productBlueprint";

// Size / ModelNumber の型だけ借りる
import type { SizeRow } from "../../../model/src/presentation/hook/useModelCard";
import type { ModelNumber } from "../../../model/src/application/modelCreateService";

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
  // backend が Go のデフォルトエンコード（フィールド名そのまま）なので、
  // 大文字の "ID" 等を含めて幅広く許容しておく
  ID?: string;
  id?: string;
  productId?: string;
  productID?: string;
  [key: string]: unknown;
};

/**
 * CreateModelVariation 用のリクエストペイロード。
 * 実際の backend の modeldom.NewModelVariation 構造に合わせて
 * フィールド名は後から調整してください。
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  modelNumber: string;
  measurements: {
    chest?: number | null;
    waist?: number | null;
    length?: number | null;
    shoulder?: number | null;
  };
};

// ------------------------------
// 内部ヘルパー: ModelVariation 作成 API
// ------------------------------

/**
 * CreateModelVariation (POST /models/{productID}/variations) を叩くヘルパー。
 *
 * backend 側:
 *   func (u *ModelUsecase) CreateModelVariation(ctx context.Context, productID string, v modeldom.NewModelVariation)
 * に対応。
 */
async function createModelVariation(
  productId: string,
  variation: NewModelVariationPayload,
  idToken: string,
): Promise<void> {
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
    console.error("[productBlueprintCreateService] CreateModelVariation failed", {
      status: res.status,
      statusText: res.statusText,
      detail,
    });
    throw new Error(
      `モデルバリエーションの作成に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }
}

// ------------------------------
// Service 本体
// ------------------------------

/**
 * 商品設計を作成する HTTP サービス
 *
 * フロー:
 * 1. POST /product-blueprints で ProductBlueprint を作成
 * 2. 返ってきた ID を ModelUsecase の productID とみなし、
 *    POST /models/{productId}/variations で CreateModelVariation を
 *    modelNumbers / sizes から組み立てて複数回叩く
 *
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
    // ここでは直接 CreateModelVariation にも利用する
    colors: params.colors,
    sizes: params.sizes,
    modelNumbers: params.modelNumbers,

    // backend: AssigneeID（null の場合は usecase 側で補完してもよい）
    assigneeId: params.assigneeId ?? null,
  };

  // 1. ProductBlueprint 作成
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
    console.error("[productBlueprintCreateService] POST /product-blueprints failed", {
      status: res.status,
      statusText: res.statusText,
      detail,
    });
    throw new Error(
      `商品設計の作成に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const json = (await res.json()) as ProductBlueprintResponse;

  // ★ backend のレスポンスから productId を推測する
  //   あなたのログでは { ID: '6njrfelq2lU4T01Fe37t', ... } なので、
  //   最後に大文字の ID も見るようにしている。
  const anyJson = json as any;
  const productIdRaw =
    anyJson.productId ??
    anyJson.productID ??
    anyJson.id ??
    anyJson.ID;

  const productId =
    typeof productIdRaw === "string" ? productIdRaw.trim() : "";

  if (!productId) {
    // ProductBlueprint 作成は成功しているが、Model 側の ID がわからないため
    // CreateModelVariation はスキップしておく
    console.warn(
      "[productBlueprintCreateService] productId not found in response; skip CreateModelVariation",
      json,
    );
    return json;
  }

  // 2. CreateModelVariation をサイズ・カラーごとに叩く
  // modelNumbers と sizes から NewModelVariationPayload を組み立てる。
  // - modelNumbers: { size, color, code }
  // - sizes:        { id, sizeLabel, chest, waist, length, shoulder }
  //
  // 実際の modeldom.NewModelVariation の定義に合わせてマッピングは調整してください。
  const sizeMap = new Map<string, SizeRow>();
  for (const s of params.sizes ?? []) {
    if (s.sizeLabel) {
      sizeMap.set(s.sizeLabel, s);
    }
  }

  const variations: NewModelVariationPayload[] = (params.modelNumbers ?? []).map(
    (mn) => {
      const size = sizeMap.get(mn.size);
      return {
        sizeLabel: mn.size,
        color: mn.color,
        modelNumber: mn.code,
        measurements: {
          chest:
            typeof size?.chest === "number" && !Number.isNaN(size.chest)
              ? size.chest
              : null,
          waist:
            typeof size?.waist === "number" && !Number.isNaN(size.waist)
              ? size.waist
              : null,
          length:
            typeof size?.length === "number" && !Number.isNaN(size.length)
              ? size.length
              : null,
          shoulder:
            typeof size?.shoulder === "number" && !Number.isNaN(size.shoulder)
              ? size.shoulder
              : null,
        },
      };
    },
  );

  if (variations.length > 0) {
    try {
      await Promise.all(
        variations.map((v) => createModelVariation(productId, v, idToken)),
      );
    } catch (e) {
      console.error(
        "[productBlueprintCreateService] one or more CreateModelVariation calls failed",
        e,
      );
      // ProductBlueprint の作成は成功しているので、ここでは例外をそのまま投げるか、
      // 必要に応じてロールバック戦略を検討する。
      throw e instanceof Error
        ? e
        : new Error("モデルバリエーションの作成に失敗しました。");
    }
  }

  return json;
}
