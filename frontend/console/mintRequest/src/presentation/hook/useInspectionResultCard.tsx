// frontend/console/mintRequest/src/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";
import type {
  InspectionBatch,
  InspectionItem,
  InspectionResult,
} from "../../domain/entity/inspections";

/**
 * 検査結果カード 1 行分
 * - modelNumber: 型番（InspectionItem.modelNumber を利用）
 * - size / color / rgb は今後 ModelVariation などと join する前提で、現状はプレースホルダ
 * - passedQuantity: 合格数
 * - quantity      : 生産数（＝該当行の検査件数）
 */
export type InspectionResultRow = {
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  passedQuantity: number;
  quantity: number;
};

export type UseInspectionResultCardParams = {
  /**
   * InspectionBatch（inspection.ts と対応）
   * Detail 画面では /products/inspections?productionId=... のレスポンス
   * (= InspectionBatchDTO) をそのまま渡す想定。
   */
  batch: InspectionBatch | null | undefined;
};

export type UseInspectionResultCardResult = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;
  rgbIntToHex: (rgb: number | string | null | undefined) => string | null;
};

/**
 * InspectionBatch（inspection.ts 準拠）から
 * InspectionResultCard 用の行データ・集計値・RGB変換関数を提供するフック。
 */
export function useInspectionResultCard(
  params: UseInspectionResultCardParams,
): UseInspectionResultCardResult {
  const { batch } = params;

  const rows: InspectionResultRow[] = React.useMemo(() => {
    if (!batch) return [];

    // modelNumber 単位で集計
    const map = new Map<
      string,
      { passed: number; total: number; items: InspectionItem[] }
    >();

    for (const ins of batch.inspections ?? []) {
      const modelNumber = (ins.modelNumber ?? "").trim();
      if (!modelNumber) continue;

      const entry =
        map.get(modelNumber) ?? { passed: 0, total: 0, items: [] };

      entry.total += 1;
      if (ins.inspectionResult === "passed") {
        entry.passed += 1;
      }
      entry.items.push(ins);

      map.set(modelNumber, entry);
    }

    const result: InspectionResultRow[] = [];
    for (const [modelNumber, agg] of map.entries()) {
      // TODO: 将来ここで ModelVariation 情報（size / color / rgb）を join して埋める
      result.push({
        modelNumber,
        size: "", // API 拡張後に埋める
        color: "", // API 拡張後に埋める
        rgb: null, // API 拡張後に埋める
        passedQuantity: agg.passed,
        quantity: agg.total,
      });
    }

    return result;
  }, [batch]);

  const totalPassed = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.passedQuantity || 0), 0),
    [rows],
  );

  const totalQuantity = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [rows],
  );

  // RGB → HEX (#RRGGBB) 変換
  const rgbIntToHex = React.useCallback(
    (rgb: number | string | null | undefined): string | null => {
      if (rgb === null || rgb === undefined) return null;
      const n = typeof rgb === "string" ? Number(rgb) : rgb;
      if (!Number.isFinite(n)) return null;

      const clamped = Math.max(0, Math.min(0xffffff, Math.floor(n)));
      const hex = clamped.toString(16).padStart(6, "0");
      return `#${hex}`;
    },
    [],
  );

  return {
    title: "検品結果",
    rows,
    totalPassed,
    totalQuantity,
    rgbIntToHex,
  };
}
