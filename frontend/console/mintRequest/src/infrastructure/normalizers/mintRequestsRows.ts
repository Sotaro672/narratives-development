// frontend/console/mintRequest/src/infrastructure/normalizers/mintRequestsRows.ts

import type { MintRequestRowRaw, MintRequestsPayloadRaw } from "../dto/mintRequestRaw.dto";

/**
 * /mint/requests の payload から rows 配列を抽出（名揺れ吸収）
 */
export function normalizeMintRequestsRows(json: any): MintRequestRowRaw[] {
  if (!json) return [];
  if (Array.isArray(json)) return json as MintRequestRowRaw[];

  const rows =
    (json as any)?.rows ??
    (json as any)?.Rows ??
    (json as any)?.items ??
    (json as any)?.Items ??
    (json as any)?.data ??
    (json as any)?.Data ??
    null;

  return Array.isArray(rows) ? (rows as MintRequestRowRaw[]) : [];
}

/**
 * row から productionId 相当のキーを抽出（inspectionId/id の揺れも吸収）
 */
export function extractRowKeyAsProductionId(row: any): string {
  return String(
    row?.productionId ??
      row?.ProductionID ??
      row?.ProductionId ??
      row?.inspectionId ??
      row?.InspectionID ??
      row?.InspectionId ??
      row?.id ??
      row?.ID ??
      "",
  ).trim();
}
