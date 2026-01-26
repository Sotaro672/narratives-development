// frontend/console/mintRequest/src/infrastructure/normalizers/mintRequestDetail.ts

import type { InspectionBatchDTO, MintDTO } from "../api/mintRequestApi";
import type { MintRequestDetailDTO } from "../dto/mintRequestLocal.dto";

import { asMaybeString } from "./string";
import { normalizeMintDTO } from "./mint";
import { normalizeProductBlueprintPatch } from "./productBlueprintPatch";

/**
 * detail 取得時の inspection が “InspectionBatchDTO っぽい” かの判定
 * - inspections/results/items 等の配列フィールドがあるか
 */
export function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray((x as any).inspections) ||
    Array.isArray((x as any).Inspections) ||
    Array.isArray((x as any).results) ||
    Array.isArray((x as any).Results) ||
    Array.isArray((x as any).items) ||
    Array.isArray((x as any).Items)
  );
}

/**
 * /mint/inspections/{productionId} のレスポンス揺れを吸収して MintRequestDetailDTO に整形
 */
export function normalizeMintRequestDetail(v: any): MintRequestDetailDTO | null {
  if (!v) return null;

  const pid =
    asMaybeString(
      v?.productionId ?? v?.ProductionID ?? v?.ProductionId ?? v?.id ?? v?.ID,
    ) ?? null;

  const inspectionId =
    asMaybeString(
      v?.inspectionId ??
        v?.InspectionID ??
        v?.InspectionId ??
        v?.inspectionID ??
        v?.productionId ??
        v?.ProductionID ??
        v?.ProductionId,
    ) ?? null;

  // inspection 本体の取り出し（揺れ吸収）
  const inspectionRaw =
    v?.inspection ?? v?.inspectionBatch ?? v?.Inspection ?? v?.InspectionBatch ?? null;

  const looksLikeBatch =
    typeof v === "object" &&
    (Array.isArray((v as any)?.inspections) ||
      Array.isArray((v as any)?.Inspections) ||
      Array.isArray((v as any)?.results) ||
      Array.isArray((v as any)?.Results) ||
      Array.isArray((v as any)?.items) ||
      Array.isArray((v as any)?.Items));

  const inspection: InspectionBatchDTO | null =
    (inspectionRaw as any) ?? (looksLikeBatch ? (v as any) : null) ?? null;

  // mint 本体（揺れ吸収）
  const mintRaw = v?.mint ?? v?.Mint ?? v?.mintDTO ?? v?.MintDTO ?? null;
  const mint: MintDTO | null = mintRaw ? normalizeMintDTO(mintRaw) : null;

  // productBlueprintPatch（揺れ吸収）
  const pbpRaw =
    v?.productBlueprintPatch ??
    v?.productBlueprint ??
    v?.ProductBlueprintPatch ??
    v?.patch ??
    v?.Patch ??
    null;

  const productBlueprintPatch = normalizeProductBlueprintPatch(pbpRaw);

  // modelMeta（揺れ吸収）
  const modelMetaRaw =
    v?.modelMeta ?? v?.ModelMeta ?? v?.model_meta ?? v?.modelmeta ?? null;

  const modelMeta =
    modelMetaRaw && typeof modelMetaRaw === "object" ? modelMetaRaw : null;

  // ✅ detail の主要フィールド（UI 側で使うキー）
  const tokenBlueprintIdFromTop = asMaybeString(v?.tokenBlueprintId) ?? null;
  const tokenBlueprintIdFromMint =
    asMaybeString((mint as any)?.tokenBlueprintId) ?? null;

  const tokenBlueprintId = tokenBlueprintIdFromTop ?? tokenBlueprintIdFromMint ?? null;

  const productName =
    asMaybeString(v?.productName ?? v?.ProductName) ??
    asMaybeString((productBlueprintPatch as any)?.productName) ??
    null;

  const tokenName =
    asMaybeString(v?.tokenName) ?? asMaybeString((mint as any)?.tokenName) ?? null;

  return {
    ...(v ?? {}),
    productionId: pid,
    inspectionId,

    tokenBlueprintId,
    productName,
    tokenName,

    inspection: inspection ?? null,
    mint,
    productBlueprintPatch,
    modelMeta,
  };
}
