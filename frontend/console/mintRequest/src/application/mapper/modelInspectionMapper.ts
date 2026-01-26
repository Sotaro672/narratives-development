// frontend/console/mintRequest/src/application/mapper/modelInspectionMapper.ts

import type { InspectionBatchDTO } from "../../infrastructure/api/mintRequestApi";

// ============================================================
// Types
// ============================================================

export type ModelInspectionRow = {
  modelId: string;

  // 現状は未解決（別途 /models 側から解決して埋める想定）
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  passedCount: number; // 合格数
  totalCount: number; // 生産数（このモデルの対象件数）
};

// ============================================================
// helpers
// ============================================================

export function asNonEmptyString(v: any): string {
  return typeof v === "string" && v.trim() ? v.trim() : "";
}

function isPassedResult(v: any): boolean {
  const s = asNonEmptyString(v).toLowerCase();
  return s === "passed";
}

// ============================================================
// mapper
// ============================================================

/**
 * InspectionBatchDTO（MintInspectionView）から modelId 単位の集計行を作る。
 * - modelNumber/size/color は後段で meta 解決して埋める想定のため null 初期化
 */
export function buildModelRowsFromBatch(
  batch: InspectionBatchDTO | null,
): ModelInspectionRow[] {
  const inspections: any[] = Array.isArray((batch as any)?.inspections)
    ? ((batch as any).inspections as any[])
    : [];

  const agg = new Map<string, { modelId: string; passed: number; total: number }>();

  for (const it of inspections) {
    const modelId = asNonEmptyString(it?.modelId);
    if (!modelId) continue;

    const prev = agg.get(modelId) ?? { modelId, passed: 0, total: 0 };
    prev.total += 1;

    const result = it?.inspectionResult ?? null;
    if (isPassedResult(result)) prev.passed += 1;

    agg.set(modelId, prev);
  }

  const rows: ModelInspectionRow[] = Array.from(agg.values()).map((g) => ({
    modelId: g.modelId,
    modelNumber: null,
    size: null,
    colorName: null,
    rgb: null,
    passedCount: g.passed,
    totalCount: g.total,
  }));

  rows.sort((a, b) => a.modelId.localeCompare(b.modelId));
  return rows;
}
