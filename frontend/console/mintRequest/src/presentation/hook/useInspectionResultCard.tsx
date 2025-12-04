// frontend/console/mintRequest/src/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";
import type {
  InspectionBatch,
  InspectionItem,
} from "../../domain/entity/inspections";

/**
 * バックエンドの MintInspectionView に相当する型のうち、
 * InspectionBatch に modelMeta を足したものだけをここで再定義して使う。
 * （TS の構造的型付けなので、API から追加で来る productName などとも両立します）
 */
export type MintModelMetaEntry = {
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type InspectionBatchWithModelMeta = InspectionBatch & {
  // modelId → { size, colorName, rgb }
  modelMeta?: Record<string, MintModelMetaEntry>;
};

/**
 * 検査結果カード 1 行分
 * - modelNumber: 型番
 * - size / color / rgb: modelId → ModelMeta から取得
 * - passedQuantity: 合格数
 * - quantity      : 生産数（＝該当モデルの検査件数）
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
  /** MintInspectionView 相当（InspectionBatch + modelMeta） */
  batch: InspectionBatchWithModelMeta | null | undefined;
};

export type UseInspectionResultCardResult = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;
  rgbIntToHex: (rgb: number | string | null | undefined) => string | null;
};

/**
 * InspectionBatch（+ modelMeta）から
 * InspectionResultCard 用の行データ・集計値・RGB変換関数を提供するフック。
 */
export function useInspectionResultCard(
  params: UseInspectionResultCardParams,
): UseInspectionResultCardResult {
  const { batch } = params;

  const rows: InspectionResultRow[] = React.useMemo(() => {
    if (!batch) return [];

    const modelMeta = batch.modelMeta ?? {};

    // modelId 単位で集計（同じ modelId の inspection をまとめる）
    const map = new Map<
      string,
      {
        modelNumber: string;
        passed: number;
        total: number;
      }
    >();

    for (const ins of batch.inspections ?? []) {
      const modelId = (ins.modelId ?? "").trim();
      if (!modelId) continue;

      const modelNumber = (ins.modelNumber ?? "").trim();

      const entry =
        map.get(modelId) ?? {
          modelNumber,
          passed: 0,
          total: 0,
        };

      entry.total += 1;
      if (ins.inspectionResult === "passed") {
        entry.passed += 1;
      }

      // 途中で modelNumber が空だった場合でも、どこかで値が入れば更新
      if (!entry.modelNumber && modelNumber) {
        entry.modelNumber = modelNumber;
      }

      map.set(modelId, entry);
    }

    const result: InspectionResultRow[] = [];

    for (const [modelId, agg] of map.entries()) {
      const meta = modelMeta[modelId];

      result.push({
        modelNumber: agg.modelNumber || modelId,
        size: meta?.size?.trim() ?? "",
        color: meta?.colorName?.trim() ?? "",
        rgb: meta?.rgb ?? null,
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

  // タイトルに productName があれば補足として付けてもよい
  const title =
    (batch as any)?.productName && typeof (batch as any).productName === "string"
      ? `検査結果：${(batch as any).productName}`
      : "モデル別検査結果";

  return {
    title,
    rows,
    totalPassed,
    totalQuantity,
    rgbIntToHex,
  };
}
