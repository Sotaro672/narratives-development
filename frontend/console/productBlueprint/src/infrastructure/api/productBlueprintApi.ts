// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

// ─────────────────────────────────────────────
// 作成系 API 用の型・依存
// ─────────────────────────────────────────────
import type { ItemType, Fit } from "../../domain/entity/catalog";
import type { ProductIDTag } from "../../../../productBlueprint/src/domain/entity/productBlueprint";
import type {
  SizeRow as CatalogSizeRow,
  MeasurementKey,
} from "../../../../model/src/domain/entity/catalog";
import type { ModelNumber } from "../../../../model/src/application/modelCreateService";

import { createProductBlueprintHTTP } from "../repository/productBlueprintRepositoryHTTP";
import { createModelVariationsFromProductBlueprint } from "../../../../model/src/infrastructure/api/modelCreateApi";

// Firebase Auth から ID トークンを取得（append 用）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ISO8601 → "YYYY/M/D" 表示 ※詳細画面用（元の挙動を維持）
export const formatProductBlueprintDate = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

// 一覧表示用のUI行モデル（API が返す形）
export type ProductBlueprintListRow = {
  id: string;
  productName: string;
  brandLabel: string;
  assigneeLabel: string;
  tagLabel: string;
  createdAt: string; // YYYY/MM/DD
  lastModifiedAt: string; // YYYY/MM/DD
};

// 詳細画面用：サイズ行モデル
// ★ model ドメインの SizeRow をそのまま使う
export type SizeRow = CatalogSizeRow;

// 詳細画面用：モデルナンバー行モデル
export type ModelNumberRow = {
  size: string;
  color: string;
  code: string;
};

/* =========================================================
 * 作成系 API（createProductBlueprint + variations 作成 + modelRefs append）
 * =======================================================*/

// ProductBlueprint 作成時の入力パラメータ
export type CreateProductBlueprintParams = {
  productName: string;
  brandId: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];

  productIdTag: ProductIDTag;

  companyId: string;
  assigneeId?: string;
  createdBy?: string;

  // 商品設計画面から渡されるバリエーション情報
  colors: string[];
  sizes: CatalogSizeRow[];
  modelNumbers: ModelNumber[];

  // ColorVariationCard から渡される color 名 → HEX(RGB) のマップ
  // 例: { "グリーン": "#417505" }
  colorRgbMap?: Record<string, string>;
};

// backend から返ってくる ProductBlueprint 作成レスポンス（暫定：キー揺れ吸収）
export type ProductBlueprintResponse = {
  ID?: string;
  id?: string;
  productBlueprintId?: string;
  productBlueprintID?: string;
  [key: string]: unknown;
};

/**
 * measurements 部分の型
 * - modelCreateService.ts 側と同じく、MeasurementKey をキーにしたマップ
 */
export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

/**
 * ModelVariation 用 Payload
 *
 * - modelCreateService.ts 側の NewModelVariationPayload と構造互換
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string;
  rgb?: number; // 色の RGB 値（0xRRGGBB）
  modelNumber: string;
  /** 新規作成時の version （基本 1 から開始） */
  version?: number;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

/**
 * ProductBlueprint の ID 抽出（backend のキー揺れを吸収）
 */
function extractProductBlueprintId(json: unknown): string {
  const anyJson = json as any;
  const raw =
    anyJson?.productBlueprintId ??
    anyJson?.productBlueprintID ??
    anyJson?.id ??
    anyJson?.ID;

  return typeof raw === "string" ? raw.trim() : "";
}

function dedupKeepOrder(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of xs ?? []) {
    const v = String(raw ?? "").trim();
    if (!v) continue;
    if (seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }
  return out;
}

// 🔙 BACKEND の BASE URL（modelRepositoryHTTP と合わせる：暫定で api.ts 側に置く）
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = ENV_BASE || FALLBACK_BASE;

async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return user.getIdToken();
}

/**
 * append API（案1）
 * POST /product-blueprints/{id}/model-refs
 * body: { modelIds: string[] }
 * resp: detail（toDetailOutput）
 *
 * NOTE:
 * - repository 層に寄せたいが、まずは api.ts 側で最短実装する。
 * - 次手順で productBlueprintRepositoryHTTP に移管する。
 */
async function appendModelIdsToProductBlueprint(
  productBlueprintId: string,
  modelIds: string[],
): Promise<ProductBlueprintResponse> {
  const id = String(productBlueprintId ?? "").trim();
  if (!id) throw new Error("productBlueprintId is empty");

  const cleaned = dedupKeepOrder(modelIds);
  if (cleaned.length === 0) {
    throw new Error("modelIds is empty");
  }

  const token = await getIdTokenOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}/model-refs`;

  const resp = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
    },
    body: JSON.stringify({ modelIds: cleaned }),
  });

  const text = await resp.text().catch(() => "");

  if (!resp.ok) {
    // backend は {error:"..."} を返す想定だが、ここでは text をそのまま載せる
    throw new Error(
      `append modelIds failed: ${resp.status} ${resp.statusText}${
        text ? ` - ${text}` : ""
      }`,
    );
  }

  return (text ? JSON.parse(text) : {}) as ProductBlueprintResponse;
}

/**
 * ProductBlueprint + ModelVariations をまとめて作成し、
 * さらに modelRefs（modelIds）を append する API 呼び出し（案1）。
 *
 * - ProductBlueprint 自体の作成は createProductBlueprintHTTP に委譲
 * - 生成された productBlueprintId を使って variations を作成
 * - variations 作成で得られた modelIds を順序付きで append
 * - append の返り値（detail）を最終結果として返す
 */
export async function createProductBlueprintApi(
  params: CreateProductBlueprintParams,
  variations: NewModelVariationPayload[],
): Promise<ProductBlueprintResponse> {
  // 1. ProductBlueprint の作成（HTTP）
  const created = await createProductBlueprintHTTP(params);

  // 2. productBlueprintId 抽出
  const productBlueprintId = extractProductBlueprintId(created);

  if (!productBlueprintId) {
    // ID が取れない場合は後続をスキップ
    return created as ProductBlueprintResponse;
  }

  // 3. variations が無いなら append もしない（modelRefs も空のまま）
  if (variations.length === 0) {
    return created as ProductBlueprintResponse;
  }

  // 4. variations 作成 → modelIds（string[]）を取得（ここが “型崩れ解消” の本命）
  const modelIds = await createModelVariationsFromProductBlueprint({
    productBlueprintId,
    variations,
  });

  const cleaned = dedupKeepOrder(modelIds);
  if (cleaned.length === 0) {
    // variations は作成したが modelIds が取れないのは異常系として扱う
    throw new Error("createProductBlueprintApi: modelIds が空です");
  }

  // 5. append（返り値は detail）
  const detail = await appendModelIdsToProductBlueprint(productBlueprintId, cleaned);
  return detail;
}
