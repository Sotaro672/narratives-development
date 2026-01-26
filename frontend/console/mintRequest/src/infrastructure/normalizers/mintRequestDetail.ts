// frontend/console/mintRequest/src/infrastructure/normalizers/mintRequestDetail.ts

import type { InspectionBatchDTO, MintDTO } from "../api/mintRequestApi";
import type { MintRequestDetailDTO } from "../dto/mintRequestLocal.dto";

import { asMaybeString } from "./string";
import { normalizeMintDTO } from "./mint";
import { normalizeProductBlueprintPatch } from "./productBlueprintPatch";

/**
 * detail 取得時の inspection が “InspectionBatchDTO っぽい” かの判定
 * - inspections/results/items 等の配列フィールドがあるか
 *
 * ✅ 名揺れ（Inspections/Results/Items 等）は削除し、lowerCamelCase のみを許容
 */
export function looksLikeInspectionBatchDTO(x: any): boolean {
  if (!x || typeof x !== "object") return false;
  return (
    Array.isArray((x as any).inspections) ||
    Array.isArray((x as any).results) ||
    Array.isArray((x as any).items)
  );
}

/**
 * /mint/inspections/{productionId} または /mint/requests/{productionId} のレスポンスを
 * MintRequestDetailDTO に整形
 *
 * ✅ 名揺れを削除し、Backend DTO の json tag（lowerCamelCase）に揃える
 */
export function normalizeMintRequestDetail(v: any): MintRequestDetailDTO | null {
  if (!v || typeof v !== "object") return null;

  // ✅ canonical: productionId（fallback は id のみ）
  const productionId =
    asMaybeString(v?.productionId) ?? asMaybeString(v?.id) ?? null;

  if (!productionId) return null;

  // ✅ canonical: inspectionId（無い場合は productionId を採用）
  const inspectionId = asMaybeString(v?.inspectionId) ?? productionId;

  // inspection 本体（inspection / inspectionBatch のみ許容）
  const inspectionRaw = v?.inspection ?? v?.inspectionBatch ?? null;

  const inspection: InspectionBatchDTO | null =
    (inspectionRaw as any) ??
    (looksLikeInspectionBatchDTO(v) ? (v as any) : null) ??
    null;

  // mint 本体（mint / mintDTO のみ許容）
  const mintRaw = v?.mint ?? v?.mintDTO ?? null;
  const mint: MintDTO | null = mintRaw ? normalizeMintDTO(mintRaw) : null;

  // productBlueprintPatch（productBlueprintPatch / productBlueprint のみ許容）
  const pbpRaw = v?.productBlueprintPatch ?? v?.productBlueprint ?? null;
  const productBlueprintPatch = normalizeProductBlueprintPatch(pbpRaw);

  // modelMeta（modelMeta のみ許容）
  const modelMetaRaw = v?.modelMeta ?? null;
  const modelMeta =
    modelMetaRaw && typeof modelMetaRaw === "object" ? modelMetaRaw : null;

  // ✅ detail の主要フィールド（UI 側で使うキー）
  const tokenBlueprintIdFromTop = asMaybeString(v?.tokenBlueprintId) ?? null;
  const tokenBlueprintIdFromMint =
    asMaybeString((mint as any)?.tokenBlueprintId) ?? null;

  const tokenBlueprintId =
    tokenBlueprintIdFromTop ?? tokenBlueprintIdFromMint ?? null;

  const productName =
    asMaybeString(v?.productName) ??
    asMaybeString((productBlueprintPatch as any)?.productName) ??
    null;

  const tokenName =
    asMaybeString(v?.tokenName) ?? asMaybeString((mint as any)?.tokenName) ?? null;

  // ✅ requester（Backend: requestedBy / requestedByName）
  const requestedBy =
    asMaybeString(v?.requestedBy) ??
    null;

  const requestedByName =
    asMaybeString(v?.requestedByName) ??
    asMaybeString((mint as any)?.requestedByName) ??
    null;

  return {
    productionId,
    inspectionId,

    tokenBlueprintId,
    productName,
    tokenName,

    requestedBy,
    requestedByName,

    inspection: inspection ?? null,
    mint,
    productBlueprintPatch,
    modelMeta,
  } as any;
}
