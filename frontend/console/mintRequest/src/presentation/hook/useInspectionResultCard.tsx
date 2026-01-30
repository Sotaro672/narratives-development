// frontend/console/mintRequest/src/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";
import type { InspectionBatch } from "../../domain/entity/inspections";

// ★ 追加：modelId -> ModelVariation を引く（modelNumber/size/color を解決する）
import { fetchModelVariationByIdForMintHTTP } from "../../infrastructure/repository";
import type { ModelVariationForMintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

// ✅ 追加：共通カラー変換ユーティリティ（rgb int -> "#RRGGBB"）
import { rgbIntToHex as rgbIntToHexShared } from "../../../../shell/src/shared/util/color";

/**
 * バックエンドの MintInspectionView に相当する型のうち、
 * InspectionBatch に modelMeta / productBlueprintPatch を足したものだけをここで再定義して使う。
 * （TS の構造的型付けなので、API から追加で来る productName などとも両立します）
 */
export type MintModelMetaEntry = {
  // ★ 追加：modelNumber もここで解決できるようにする
  modelNumber?: string | null;

  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type InspectionBatchWithModelMeta = InspectionBatch & {
  // modelId → { modelNumber, size, colorName, rgb }
  modelMeta?: Record<string, MintModelMetaEntry>;

  // ★ 追加：ProductBlueprintPatch（modelRefs=displayOrder の唯一のソース）
  productBlueprintPatch?: {
    modelRefs?: Array<{ modelId: string; displayOrder: number }> | null;
    [k: string]: any;
  } | null;
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
  /** MintInspectionView 相当（InspectionBatch + modelMeta + productBlueprintPatch） */
  batch: InspectionBatchWithModelMeta | null | undefined;
};

export type UseInspectionResultCardResult = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;

  // ✅ 共通 util 互換に合わせる：undefined も返しうる
  rgbIntToHex: (rgb: number | string | null | undefined) => string | undefined;
};

/**
 * InspectionBatch（+ modelMeta）から
 * InspectionResultCard 用の行データ・集計値・RGB変換関数を提供するフック。
 *
 * ★ 期待値対応：
 * - inspections から取得できる modelId を使って
 *   GetModelVariationByID（HTTP）を叩き、modelNumber/size/color を補完する
 * - 行の並び順は ProductBlueprintPatch.modelRefs.displayOrder を正とする
 */
export function useInspectionResultCard(
  params: UseInspectionResultCardParams,
): UseInspectionResultCardResult {
  const { batch } = params;

  // ★ 追加：API から来ない/不足している modelMeta をここで補完する
  const [resolvedMeta, setResolvedMeta] = React.useState<
    Record<string, MintModelMetaEntry>
  >({});

  // batch が変わったら補完キャッシュもリセット（production 切替時など）
  React.useEffect(() => {
    setResolvedMeta({});
  }, [batch?.productionId, (batch as any)?.id, (batch as any)?.inspectionId]);

  // inspections からユニークな modelId を抽出
  const modelIds: string[] = React.useMemo(() => {
    if (!batch?.inspections) return [];
    const set = new Set<string>();

    for (const ins of batch.inspections ?? []) {
      const mid = String((ins as any)?.modelId ?? "").trim();
      if (mid) set.add(mid);
    }
    return Array.from(set);
  }, [batch]);

  // 既存 meta（APIから来た分 + こちらで解決した分）をマージ
  const mergedModelMeta: Record<string, MintModelMetaEntry> = React.useMemo(() => {
    return {
      ...(batch?.modelMeta ?? {}),
      ...(resolvedMeta ?? {}),
    };
  }, [batch, resolvedMeta]);

  // まだ meta が無い modelId だけを抽出
  const missingModelIds: string[] = React.useMemo(() => {
    if (modelIds.length === 0) return [];
    return modelIds.filter((id) => !mergedModelMeta[id]);
  }, [modelIds, mergedModelMeta]);

  // ★ NEW: ProductBlueprintPatch.modelRefs から modelId -> displayOrder を構築
  const displayOrderByModelId: Record<string, number> = React.useMemo(() => {
    const refs = batch?.productBlueprintPatch?.modelRefs ?? [];
    const out: Record<string, number> = {};
    for (const r of refs ?? []) {
      const mid = String((r as any)?.modelId ?? "").trim();
      const ord = (r as any)?.displayOrder;
      if (!mid) continue;
      if (typeof ord !== "number" || !Number.isFinite(ord)) continue;
      out[mid] = ord;
    }
    return out;
  }, [batch?.productBlueprintPatch]);

  // ★ 追加：missingModelIds を GetModelVariationByID（HTTP）で解決して modelMeta を埋める
  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[useInspectionResultCard] effect fired", {
      hasBatch: !!batch,
      missingModelIds,
    });

    if (!batch) return;
    if (missingModelIds.length === 0) return;

    let cancelled = false;

    (async () => {
      // まとめて叩く（N件でも Promise.all で並列）
      const settled = await Promise.all(
        missingModelIds.map(async (modelId) => {
          try {
            const v = await fetchModelVariationByIdForMintHTTP(modelId);
            return { modelId, v };
          } catch {
            return { modelId, v: null as ModelVariationForMintDTO | null };
          }
        }),
      );

      if (cancelled) return;

      setResolvedMeta((prev) => {
        const next = { ...(prev ?? {}) };

        for (const it of settled) {
          const modelId = it.modelId;
          const v = it.v;

          if (!v) continue;

          next[modelId] = {
            modelNumber: (v.modelNumber ?? "").trim() || null,
            size: (v.size ?? "").trim() || null,
            colorName: (v.colorName ?? "").trim() || null,
            rgb: typeof v.rgb === "number" ? v.rgb : null,
          };
        }

        return next;
      });
    })();

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batch, JSON.stringify(missingModelIds)]);

  const rows: InspectionResultRow[] = React.useMemo(() => {
    if (!batch) return [];

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
      const modelId = String((ins as any)?.modelId ?? "").trim();
      if (!modelId) continue;

      const modelNumberFromInspection = String((ins as any)?.modelNumber ?? "").trim();

      const entry =
        map.get(modelId) ?? {
          modelNumber: modelNumberFromInspection,
          passed: 0,
          total: 0,
        };

      entry.total += 1;
      if ((ins as any)?.inspectionResult === "passed") {
        entry.passed += 1;
      }

      // 途中で modelNumber が空だった場合でも、どこかで値が入れば更新
      if (!entry.modelNumber && modelNumberFromInspection) {
        entry.modelNumber = modelNumberFromInspection;
      }

      map.set(modelId, entry);
    }

    // 並び替えに ProductBlueprintPatch.modelRefs.displayOrder を使う
    const tmp: Array<InspectionResultRow & { __order: number }> = [];
    const INF = Number.POSITIVE_INFINITY;

    for (const [modelId, agg] of map.entries()) {
      const meta = mergedModelMeta[modelId];

      // ★ 表示優先順位：
      // meta.modelNumber（GetModelVariationByIDで解決） > inspections の modelNumber > modelId
      const displayModelNumber =
        (meta?.modelNumber ?? "").trim() ||
        (agg.modelNumber ?? "").trim() ||
        modelId;

      const order =
        typeof displayOrderByModelId[modelId] === "number" &&
        Number.isFinite(displayOrderByModelId[modelId])
          ? displayOrderByModelId[modelId]
          : INF;

      tmp.push({
        __order: order,
        modelNumber: displayModelNumber,
        size: (meta?.size ?? "").trim(),
        color: (meta?.colorName ?? "").trim(),
        rgb: meta?.rgb ?? null,
        passedQuantity: agg.passed,
        quantity: agg.total,
      });
    }

    // ✅ modelRefs.displayOrder のみに従って並べ替え（displayOrder 無しは末尾）
    tmp.sort((a, b) => a.__order - b.__order);

    return tmp.map(({ __order, ...row }) => row);
  }, [batch, mergedModelMeta, displayOrderByModelId]);

  const totalPassed = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.passedQuantity || 0), 0),
    [rows],
  );

  const totalQuantity = React.useMemo(
    () => rows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [rows],
  );

  // ✅ 共通 util を使用して RGB → HEX (#RRGGBB) 変換
  const rgbIntToHex = React.useCallback(
    (rgb: number | string | null | undefined): string | undefined => {
      return rgbIntToHexShared(rgb);
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
