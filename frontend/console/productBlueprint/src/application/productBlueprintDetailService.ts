// frontend/console/productBlueprint/src/application/productBlueprintDetailService.ts

import type { ItemType } from "../domain/entity/catalog";
import type { SizeRow } from "../../../model/src/domain/entity/catalog";
import { updateProductBlueprintHTTP } from "../infrastructure/repository/productBlueprintRepositoryHTTP";

import type {
  ProductBlueprintDetailResponse,
  UpdateProductBlueprintParams,
  NewModelVariationMeasurements,
  NewModelVariationPayload,
} from "../infrastructure/api/productBlueprintDetailApi";

import { API_BASE } from "../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../shell/src/shared/http/authHeaders";
import { coerceRgbInt, hexToRgbInt } from "../../../shell/src/shared/util/color";

import { fetchAllBrandsForCompany } from "../../../brand/src/infrastructure/query/brandQuery";
import { formatLastFirst } from "../../../member/src/infrastructure/query/memberQuery";
import { MemberRepositoryHTTP } from "../../../member/src/infrastructure/http/memberRepositoryHTTP";

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
// variations payload builder（新規作成向け）
// -----------------------------------------
function toNewModelVariationPayload(
  itemType: ItemType,
  sizeRow: SizeRow,
  base: {
    sizeLabel: string;
    color: string;
    modelNumber: string;
    createdBy: string;
    rgb?: number;
  },
): NewModelVariationPayload {
  const measurements = buildMeasurements(itemType, sizeRow);

  return {
    sizeLabel: base.sizeLabel,
    color: base.color,
    modelNumber: base.modelNumber,
    createdBy: base.createdBy,
    rgb: base.rgb,
    measurements,
  };
}

// -----------------------------------------
// 生レスポンス（PascalCase）型
// -----------------------------------------
type RawProductBlueprintDetailResponse = {
  ID: string;
  ProductName: string;
  CompanyID: string;
  BrandID: string;
  ItemType: string;
  Fit: string;
  Material: string;
  Weight: number;
  QualityAssurance?: string[];
  ProductIdTag?: { Type?: string } | null;
  AssigneeID?: string | null;
  CreatedBy?: string | null;
  CreatedAt?: string | null;
  UpdatedBy?: string | null;
  UpdatedAt?: string | null;
  DeletedBy?: string | null;
  DeletedAt?: string | null;
};

// -----------------------------------------
// ブランド名取得ヘルパー
// -----------------------------------------
async function fetchBrandNameById(brandId: string): Promise<string> {
  const id = brandId.trim();
  if (!id) return "";
  try {
    const brands = await fetchAllBrandsForCompany("", false);
    return brands.find((b) => b.id === id)?.name ?? "";
  } catch (e) {
    console.error("[productBlueprintDetailService] fetchBrandNameById error:", e);
    return "";
  }
}

// -----------------------------------------
// メンバー名解決（Repository 経由）
// -----------------------------------------
async function resolveMemberNameById(
  _authHeaders: Record<string, string>,
  memberId?: string | null,
  fallback: string = "-",
): Promise<string> {
  const id = String(memberId ?? "").trim();
  if (!id) return fallback;

  try {
    const repo = new MemberRepositoryHTTP();
    const member = await repo.getById(id);
    if (!member) return fallback;

    const name = formatLastFirst(member.lastName, member.firstName)?.trim() || id;

    return name || fallback;
  } catch (e) {
    console.error("[productBlueprintDetailService] resolveMemberNameById error:", e);
    return fallback;
  }
}

// -----------------------------------------
// GET: 商品設計 詳細
// -----------------------------------------
export async function getProductBlueprintDetail(
  id: string,
): Promise<ProductBlueprintDetailResponse> {
  const trimmed = String(id ?? "").trim();
  if (!trimmed) throw new Error("getProductBlueprintDetail: id が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/product-blueprints/${encodeURIComponent(trimmed)}`, {
    method: "GET",
    headers: authHeaders,
  });

  if (!res.ok) {
    throw new Error(
      `商品設計詳細の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as RawProductBlueprintDetailResponse;

  const response: ProductBlueprintDetailResponse & {
    brandName?: string;
    assigneeName?: string;
    createdByName?: string;
  } = {
    id: raw.ID,
    productName: raw.ProductName,
    companyId: raw.CompanyID,
    brandId: raw.BrandID,
    itemType: raw.ItemType,
    fit: raw.Fit,
    material: raw.Material,
    weight: raw.Weight,
    qualityAssurance: raw.QualityAssurance ?? [],
    productIdTag: raw.ProductIdTag ? { type: raw.ProductIdTag.Type ?? "" } : undefined,
    assigneeId: raw.AssigneeID ?? "",
    createdBy: raw.CreatedBy ?? "",
    createdAt: raw.CreatedAt ?? "",
  };

  response.brandName = await fetchBrandNameById(response.brandId ?? "");
  response.assigneeName = await resolveMemberNameById(authHeaders, response.assigneeId, "-");
  response.createdByName = await resolveMemberNameById(authHeaders, response.createdBy, "作成者未設定");

  return response;
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
    productIdTag,
    brandId,
    assigneeId,
    updatedBy,
    sizes = [],
    modelNumbers = [],
    colorRgbMap = {},
  } = params as any;

  if (!id) {
    throw new Error("updateProductBlueprint: id が空です");
  }

  // 1) まず ProductBlueprint 本体のメタ情報を更新
  const updated = await updateProductBlueprintHTTP(
    id,
    {
      ...(params as any),
      id,
      productName,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTag,
      brandId,
      assigneeId,
      updatedBy,
    } as unknown as UpdateProductBlueprintParams,
  );

  // itemType が不明なら variations 更新はスキップ（メタ情報だけ更新）
  if (!itemType) {
    console.log("[updateProductBlueprint] itemType が空のため、ModelVariation の更新はスキップします。");
    return updated;
  }

  // 2) 現在の ModelVariation 一覧を取得
  const variations = await listModelVariationsByProductBlueprintId(id);
  const varsAny = variations as any[];

  // 3) 既存 variation を size×color → variation にマップ
  const existingMap = new Map<string, any>();
  varsAny.forEach((v) => {
    const sizeLabel: string =
      (typeof v.size === "string" ? v.size : (v.Size as string | undefined)) ?? "";
    const colorName: string =
      (typeof v.color?.name === "string" ? v.color.name : (v.Color?.Name as string | undefined)) ??
      "";

    if (!sizeLabel || !colorName) return;
    const key = makeKey(sizeLabel, colorName);
    existingMap.set(key, v);
  });

  // 4) size×color → modelNumber(code) のマップ（希望状態）
  const codeMap = new Map<string, string>();
  modelNumbers.forEach((m: { size: string; color: string; code: string }) => {
    if (!m.size || !m.color) return;
    const key = makeKey(m.size, m.color);
    codeMap.set(key, m.code ?? "");
  });

  // 5) sizeLabel → measurements(map[string]float64) のマップ
  const measurementsMap = new Map<string, Record<string, number>>();
  (sizes as SizeRow[]).forEach((s) => {
    const ms = buildMeasurementsFromSizeRowForUpdate(itemType as ItemType, s);
    if (ms) {
      measurementsMap.set(s.sizeLabel, ms);
    }
  });

  // 6) 既存 variation は updateModelVariation で更新
  const updateTasks: Promise<void>[] = [];

  existingMap.forEach((v, key) => {
    const variationId: string = v.id ?? v.ID;
    if (!variationId) return;

    const sizeLabel: string =
      (typeof v.size === "string" ? v.size : (v.Size as string | undefined)) ?? "";
    const colorName: string =
      (typeof v.color?.name === "string" ? v.color.name : (v.Color?.Name as string | undefined)) ??
      "";

    if (!sizeLabel || !colorName) return;

    // 希望 side の modelNumber（なければ既存値を維持）
    const nextCode: string =
      codeMap.get(key) ??
      (typeof v.modelNumber === "string"
        ? v.modelNumber
        : (v.ModelNumber as string | undefined) ?? "");

    // RGB（hex から int に変換。無ければ既存値を維持）
    const rgbHex = colorRgbMap[colorName];
    const rgbFromHex = hexToRgbInt(rgbHex);

    const existingRgb = coerceRgbInt(
      (v as any)?.color?.rgb ??
        (v as any)?.color?.RGB ??
        (v as any)?.Color?.rgb ??
        (v as any)?.Color?.RGB,
    );

    const rgb = rgbFromHex ?? existingRgb;

    // 採寸（SizeRow から起こした map）
    const measurements = measurementsMap.get(sizeLabel);

    const payload: ModelVariationUpdateRequest = {
      modelNumber: nextCode,
      size: sizeLabel,
      color: colorName,
      ...(typeof rgb === "number" ? { rgb } : {}),
      ...(measurements ? { measurements } : {}),
    };

    console.log("[updateProductBlueprint] updateModelVariation payload:", {
      variationId,
      payload,
    });

    updateTasks.push(
      (async () => {
        await updateModelVariation(variationId, payload);
      })(),
    );
  });

  // 既存分の更新を待つ
  await Promise.all(updateTasks);

  // 7) 既存に存在しない（新規の） size×color は CreateModelVariation で作成
  const createPayloads: CreateModelVariationRequest[] = [];

  codeMap.forEach((code, key) => {
    if (existingMap.has(key)) {
      // 既存 variation については上で更新済み
      return;
    }

    const [sizeLabel, colorName] = key.split("__");
    if (!sizeLabel || !colorName) return;

    const sizeRow = (sizes as SizeRow[]).find((s) => s.sizeLabel === sizeLabel);
    if (!sizeRow) return;

    const rgbHex = colorRgbMap[colorName];
    const rgb = hexToRgbInt(rgbHex);

    const measurements = buildMeasurements(itemType as ItemType, sizeRow);

    const createReq: CreateModelVariationRequest = {
      productBlueprintId: id,
      modelNumber: code,
      size: sizeLabel,
      color: colorName,
      ...(typeof rgb === "number" ? { rgb } : {}),
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
  color?: { name: string; rgb?: number | null };
  measurements?: Record<string, number | null>;
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

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/models/by-blueprint/${encodeURIComponent(id)}/variations`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  if (!res.ok) {
    throw new Error(
      `モデル一覧の取得に失敗しました（${res.status} ${res.statusText ?? ""}）`,
    );
  }

  const raw = (await res.json()) as any[] | null;
  if (!raw) return [];

  return raw.map((v: any) => {
    const colorRaw = v.color ?? v.Color ?? {};
    const measurementsRaw = v.measurements ?? v.Measurements ?? {};

    const rgbValue = coerceRgbInt(colorRaw.rgb ?? colorRaw.RGB) ?? null;

    return {
      id: v.id ?? v.ID ?? "",
      productBlueprintId: v.productBlueprintId ?? v.ProductBlueprintID ?? id,
      modelNumber: v.modelNumber ?? v.ModelNumber ?? "",
      size: v.size ?? v.Size ?? "",
      color: { name: colorRaw.name ?? colorRaw.Name ?? "", rgb: rgbValue },
      measurements:
        typeof measurementsRaw === "object"
          ? (measurementsRaw as Record<string, number | null>)
          : {},
      createdAt: v.createdAt ?? v.CreatedAt ?? null,
      createdBy: v.createdBy ?? v.CreatedBy ?? null,
      updatedAt: v.updatedAt ?? v.UpdatedAt ?? null,
      updatedBy: v.updatedBy ?? v.UpdatedBy ?? null,
    };
  });
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
  updatedBy?: string; // メンバーID（表示名は別途解決）
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

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}/history`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

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
export async function softDeleteProductBlueprint(productBlueprintId: string): Promise<void> {
  const id = productBlueprintId.trim();
  if (!id) {
    throw new Error("softDeleteProductBlueprint: productBlueprintId が空です");
  }

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/product-blueprints/${encodeURIComponent(id)}`;

  const res = await fetch(url, {
    method: "DELETE",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

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

  // handler 側は 204 No Content を返す想定なので、
  // 正常系では何も返さず終了（void）で問題ありません。
}
