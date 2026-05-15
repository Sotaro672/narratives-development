// frontend/console/mintRequest/src/application/mapper/modelInspectionMapper.ts

import type { InspectionBatchDTO } from "../../domain/entity/inspections";
import { asNonEmptyString } from "../util/primitive";

// ============================================================
// Types
// ============================================================

export type ModelInspectionRow = {
  modelId: string;

  // 後段で modelMeta / modelVariation から解決して埋める想定
  modelNumber: string | null;
  size: string | null;
  colorName: string | null;
  rgb: number | null;

  /**
   * alcohol 対応:
   * model variation 側で容量と単位を扱う。
   *
   * 例:
   * - volume: 720
   * - volumeUnit: "ml"
   */
  volume: string | number | null;
  volumeUnit: string | null;

  passedCount: number; // 合格数
  totalCount: number; // 生産数（このモデルの対象件数）
};

// ============================================================
// helpers
// ============================================================

function isPassedResult(v: unknown): boolean {
  const s = asNonEmptyString(v).toLowerCase();
  return s === "passed";
}

// ============================================================
// mapper
// ============================================================

/**
 * InspectionBatchDTO から modelId 単位の集計行を作る。
 *
 * この mapper の責務:
 * - inspections を modelId 単位で集計する
 * - passedCount / totalCount を算出する
 *
 * この mapper では行わないこと:
 * - productBlueprintCategory の判定
 * - alcohol / apparel などカテゴリ別表示の切り替え
 * - modelNumber / size / color / volume の解決
 *
 * modelNumber / size / color / rgb / volume / volumeUnit は、
 * 後段で modelMeta / modelVariation から解決して埋める前提のため null 初期化する。
 */
export function buildModelRowsFromBatch(
  batch: InspectionBatchDTO | null,
): ModelInspectionRow[] {
  const inspections: any[] = Array.isArray((batch as any)?.inspections)
    ? ((batch as any).inspections as any[])
    : [];

  const agg = new Map<
    string,
    {
      modelId: string;
      passed: number;
      total: number;
    }
  >();

  for (const it of inspections) {
    const modelId = asNonEmptyString(it?.modelId);
    if (!modelId) continue;

    const prev = agg.get(modelId) ?? {
      modelId,
      passed: 0,
      total: 0,
    };

    prev.total += 1;

    const result = it?.inspectionResult ?? null;
    if (isPassedResult(result)) {
      prev.passed += 1;
    }

    agg.set(modelId, prev);
  }

  const rows: ModelInspectionRow[] = Array.from(agg.values()).map((g) => ({
    modelId: g.modelId,

    modelNumber: null,
    size: null,
    colorName: null,
    rgb: null,

    volume: null,
    volumeUnit: null,

    passedCount: g.passed,
    totalCount: g.total,
  }));

  rows.sort((a, b) => a.modelId.localeCompare(b.modelId));

  return rows;
}