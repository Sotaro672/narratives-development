// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";
import { updateProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import {
  getProductBlueprintDetailApi,
  type ProductBlueprintDetailResponse,
  type UpdateProductBlueprintParams,
  type NewModelVariationMeasurements,
} from "../infrastructure/api/productBlueprintDetailApi";

import { authorizedFetch } from "../infrastructure/httpClient/authorizedFetch";
import { hexToRgbInt } from "../../../shell/src/shared/util/color";

// ★ ModelVariation 更新サービスを利用（差分削除も利用）
import {
  updateModelVariation,
  type ModelVariationUpdateRequest,
  deleteRemovedModelVariations,
  type ModelVariationResponse as ModelUpdateServiceVariationResponse,
} from "../../../model/src/application/modelUpdateService";

// ★ 新規 ModelVariation 作成用 Repository を利用
import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../../../model/src/infrastructure/repository/modelRepositoryHTTP";

// size + color → 一意キー
const makeKey = (sizeLabel: string, color: string) => `${sizeLabel}__${color}`;

// -----------------------------------------
// itemType → measurements 組み立て（新規作成向け）
// -----------------------------------------
function buildMeasurements(itemType: ItemType, size: SizeRow): NewModelVariationMeasurements {
  const result: NewModelVariationMeasurements = {};

  if (itemType === "ボトムス") {
    result["ウエスト"] = size.waist ?? null;
    result["ヒップ"] = size.hip ?? null;
    result["股上"] = size.rise ?? null;
    result["股下"] = size.inseam ?? null;
    result["わたり幅"] = size.thigh ?? null;
    result["裾幅"] = size.hemWidth ?? null;
    return result;
  }

  result["着丈"] = size.length ?? null;
  result["身幅"] = size.width ?? null;
  result["胸囲"] = size.chest ?? null;
  result["肩幅"] = size.shoulder ?? null;
  result["袖丈"] = size.sleeveLength ?? null;

  return result;
}

// -----------------------------------------
// UPDATE 用: SizeRow → map[string]float64（null は除外）
// -----------------------------------------
function buildMeasurementsFromSizeRowForUpdate(
  itemType: ItemType,
  size: SizeRow,
): Record<string, number> | undefined {
  const base = buildMeasurements(itemType, size);
  const result: Record<string, number> = {};

  Object.entries(base).forEach(([k, v]) => {
    if (typeof v === "number" && !Number.isNaN(v)) {
      result[k] = v;
    }
  });

  return Object.keys(result).length > 0 ? result : undefined;
}

// -----------------------------------------
// CREATE 用: null を除外して map[string]float64 互換にする
// （backend dto.go: map[string]float64 へ送るため）
// -----------------------------------------
function buildMeasurementsForCreate(
  itemType: ItemType,
  size: SizeRow,
): Record<string, number> | undefined {
  return buildMeasurementsFromSizeRowForUpdate(itemType, size);
}

// -----------------------------------------
// GET: 商品設計 詳細
// ✅ 方針A: backend 正（camelCase + name 解決済み）をそのまま返す
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) throw new Error("getProductBlueprintDetail: id が空です");

  return await getProductBlueprintDetailApi(trimmed);
}

// -----------------------------------------
// UPDATE（Blueprint メタ情報 + ModelVariation）
// -----------------------------------------
export async function updateProductBlueprint(
  params: UpdateProductBlueprintParams & {
    sizes?: SizeRow[];
    modelNumbers?: { size: string; color: string; code: string }[];
    colorRgbMap?: Record<string, string>;
  },
): Promise<ProductBlueprintDetailResponse> {
  const {
    id,
    productName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,
    brandId,
    assigneeId,
    companyId,
    updatedBy,
    colors,
    colorRgbMap = {},
    sizes = [],
    modelNumbers = [],
  } = params;

  if (!id) {
    throw new Error("updateProductBlueprint: id が空です");
  }

  // 1) ProductBlueprint 本体のメタ情報を更新
  //    ✅ variations はこの API へは送らない（ModelVariation は別エンドポイントで更新する）
  const updated = await updateProductBlueprintHTTP(
    id,
    {
      id,
      productName,
      brandId,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTagType,
      companyId,
      assigneeId,
      colors: colors ?? [],
      colorRgbMap: colorRgbMap ?? {},
      updatedBy: updatedBy ?? null,
    } satisfies UpdateProductBlueprintParams,
  );

  // itemType が不明なら variations 更新はスキップ（メタ情報だけ更新）
  if (!itemType) {
    console.log(
      "[updateProductBlueprint] itemType が空のため、ModelVariation の更新はスキップします。",
    );
    return updated;
  }

  // UpdateProductBlueprintParams 側の itemType は string 扱いのため、ここで正の ItemType に寄せる
  const itemTypeValue = itemType as ItemType;

  // 2) 現在の ModelVariation 一覧を取得（backend は camelCase を必ず返す前提）
  const variations = await listModelVariationsByProductBlueprintId(id);

  // 3) 既存 variation を size×color → variation にマップ
  const existingMap = new Map<string, ModelVariationResponse>();
  variations.forEach((v) => {
    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();
    if (!sizeLabel || !colorName) return;

    existingMap.set(makeKey(sizeLabel, colorName), v);
  });

  // 4) size×color → modelNumber(code) のマップ（希望状態）
  const codeMap = new Map<string, string>();
  modelNumbers.forEach((m) => {
    if (!m.size || !m.color) return;
    codeMap.set(makeKey(m.size, m.color), m.code ?? "");
  });

  // 5) sizeLabel → measurements(map[string]float64) のマップ（UPDATE 用）
  const measurementsMap = new Map<string, Record<string, number>>();
  sizes.forEach((s) => {
    const ms = buildMeasurementsFromSizeRowForUpdate(itemTypeValue, s);
    if (ms) measurementsMap.set(s.sizeLabel, ms);
  });

  // 6) 既存 variation は updateModelVariation で更新
  // updateModelVariation は Promise<void> ではないため、void に正規化して積む
  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((v, key) => {
    const variationId = v.id;
    if (!variationId) return;

    const sizeLabel = (v.size ?? "").trim();
    const colorName = (v.color?.name ?? "").trim();
    if (!sizeLabel || !colorName) return;

    // 希望 side の modelNumber（なければ既存値を維持）
    const nextCode = codeMap.get(key) ?? (v.modelNumber ?? "");

    // RGB（hex -> int）。rgb は必須（colorPicker起点で常に number になる前提）
    const rgbHex = colorRgbMap[colorName];
    const rgb = hexToRgbInt(rgbHex);
    if (typeof rgb !== "number") {
      throw new Error(
        `updateProductBlueprint: rgb が解決できません（color="${colorName}", hex="${rgbHex ?? ""}"）`,
      );
    }

    // 採寸（SizeRow から起こした map）
    const measurements = measurementsMap.get(sizeLabel);

    const payload: ModelVariationUpdateRequest = {
      modelNumber: nextCode,
      size: sizeLabel,
      color: colorName,
      rgb, // ✅ 必須で常に送る
      ...(measurements ? { measurements } : {}),
    };

    console.log("[updateProductBlueprint] updateModelVariation payload:", {
      variationId,
      payload,
    });

    updateTasks.push(updateModelVariation(variationId, payload).then(() => undefined));
  });

  await Promise.all(updateTasks);

  // 7) 既存に存在しない（新規の） size×color は CreateModelVariation で作成
  const createPayloads: CreateModelVariationRequest[] = [];

  codeMap.forEach((code, key) => {
    if (existingMap.has(key)) return;

    const [sizeLabel, colorName] = key.split("__");
    if (!sizeLabel || !colorName) return;

    const sizeRow = sizes.find((s) => s.sizeLabel === sizeLabel);
    if (!sizeRow) return;

    const rgbHex = colorRgbMap[colorName];
    const rgb = hexToRgbInt(rgbHex);
    if (typeof rgb !== "number") {
      throw new Error(
        `updateProductBlueprint: rgb が解決できません（color="${colorName}", hex="${rgbHex ?? ""}"）`,
      );
    }

    // CREATE は null を除外して送る（backend dto.go: map[string]float64）
    const measurements = buildMeasurementsForCreate(itemTypeValue, sizeRow) ?? {};

    const createReq: CreateModelVariationRequest = {
      productBlueprintId: id,
      modelNumber: code,
      size: sizeLabel,
      color: colorName,
      rgb, // ✅ 必須で常に送る
      measurements,
    };

    createPayloads.push(createReq);
  });

  if (createPayloads.length > 0) {
    console.log("[updateProductBlueprint] createModelVariations payload:", createPayloads);
    await createModelVariations(id, createPayloads);
  }

  // 8) 差分削除の指令を modelUpdateService へ渡す
  const remainingIds = (variations as ModelUpdateServiceVariationResponse[])
    .filter((v) => {
      const key = makeKey(v.size, v.color?.name ?? "");
      return codeMap.has(key);
    })
    .map((v) => v.id);

  console.group(
    "%c[updateProductBlueprint] modelUpdateService 差分削除 指令",
    "color:#ff9500; font-weight:bold;",
  );
  console.log("📦 list 取得済み ModelVariation IDs:", variations.map((v) => v.id));
  console.log("📦 画面上に残すべき ModelVariation IDs (remainingIds):", remainingIds);
  console.groupEnd();

  await deleteRemovedModelVariations(
    variations as ModelUpdateServiceVariationResponse[],
    remainingIds,
  );

  console.log("[updateProductBlueprint] completed variations update");

  return updated;
}

// -----------------------------------------
// ModelVariation list
// -----------------------------------------
export type ModelVariationResponse = {
  id: string;
  productBlueprintId: string;
  modelNumber: string;
  size: string;
  color: { name: string; rgb: number }; // ✅ backend 正: 常に返す前提
  measurements?: Record<string, number>; // ✅ backend 正: map[string]int -> number
  createdAt?: string | null;
  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
};

export async function listModelVariationsByProductBlueprintId(
  productBlueprintId: string,
): Promise<ModelVariationResponse[]> {
  const id = productBlueprintId.trim();
  if (!id) throw new Error("productBlueprintId が空です");

  const res = await authorizedFetch(
    `/models/by-blueprint/${encodeURIComponent(id)}/variations`,
    {
      method: "GET",
      throwOnError: false,
      acceptJson: true,
    },
  );

  if (!res.ok) {
    throw new Error(
      `モデル一覧の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as ModelVariationResponse[] | null;
  return raw ?? [];
}

// -----------------------------------------
// 商品設計の履歴一覧取得（LogCard 用）
// -----------------------------------------
export type ProductBlueprintHistoryItem = {
  id: string;
  productName: string;
  brandId: string;
  assigneeId: string;
  updatedAt: string; // "YYYY/MM/DD HH:MM:SS"
  updatedBy?: string;
  deletedAt?: string;
  expireAt?: string;
};

export async function getProductBlueprintHistory(
  productBlueprintId: string,
): Promise<ProductBlueprintHistoryItem[]> {
  const id = productBlueprintId.trim();
  if (!id) {
    throw new Error("getProductBlueprintHistory: productBlueprintId が空です");
  }

  const res = await authorizedFetch(
    `/product-blueprints/${encodeURIComponent(id)}/history`,
    {
      method: "GET",
      throwOnError: false,
      acceptJson: true,
    },
  );

  if (!res.ok) {
    throw new Error(
      `商品設計履歴の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as any[] | null;
  if (!raw) return [];

  return raw.map((v: any): ProductBlueprintHistoryItem => ({
    id: v.id ?? v.ID ?? "",
    productName: v.productName ?? v.ProductName ?? "",
    brandId: v.brandId ?? v.BrandId ?? "",
    assigneeId: v.assigneeId ?? v.AssigneeId ?? "",
    updatedAt: v.updatedAt ?? v.UpdatedAt ?? "",
    updatedBy: v.updatedBy ?? v.UpdatedBy ?? undefined,
    deletedAt: v.deletedAt ?? v.DeletedAt ?? undefined,
    expireAt: v.expireAt ?? v.ExpireAt ?? undefined,
  }));
}

// -----------------------------------------
// DELETE: 商品設計 論理削除
// -----------------------------------------
export async function softDeleteProductBlueprint(
  productBlueprintId: string,
): Promise<void> {
  const id = productBlueprintId.trim();
  if (!id) {
    throw new Error("softDeleteProductBlueprint: productBlueprintId が空です");
  }

  const res = await authorizedFetch(
    `/product-blueprints/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
      throwOnError: false,
      acceptJson: true,
    },
  );

  if (!res.ok) {
    let detail = "";
    try {
      detail = await res.text();
    } catch {
      // ignore
    }

    throw new Error(
      `商品設計の削除に失敗しました（${res.status} ${res.statusText}）${
        detail ? `\n${detail}` : ""
      }`,
    );
  }
}
