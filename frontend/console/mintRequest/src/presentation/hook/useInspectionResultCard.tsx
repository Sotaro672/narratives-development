// frontend/console/mintRequest/src/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";
import type { InspectionBatch } from "../../domain/entity/inspections";

import {
  buildInspectionResultCardData,
  getInspectionModelIds,
  getMissingModelIds,
} from "../../application/mapper/buildInspectionResultCardData";
import { toMintModelMetaEntry } from "../../application/mapper/modelVariationMapper";

// ★ modelId -> ModelVariation を引く（modelNumber/size/color を解決する）
import { fetchModelVariationByIdForMintHTTP } from "../../infrastructure/repository";
import type { ModelVariationForMintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

// ✅ 共通カラー変換ユーティリティ（rgb int -> "#RRGGBB"）
import { rgbIntToHex as rgbIntToHexShared } from "../../../../shell/src/shared/util/color";

/**
 * バックエンドの MintInspectionView に相当する型のうち、
 * InspectionBatch に modelMeta / productBlueprintPatch を足したものだけをここで再定義して使う。
 * （TS の構造的型付けなので、API から追加で来る productName などとも両立します）
 */
export type MintModelMetaEntry = {
  modelNumber?: string | null;
  size?: string | null;
  colorName?: string | null;
  rgb?: number | null;
};

export type InspectionBatchWithModelMeta = InspectionBatch & {
  // modelId → { modelNumber, size, colorName, rgb }
  modelMeta?: Record<string, MintModelMetaEntry>;

  // ProductBlueprintPatch（modelRefs=displayOrder の唯一のソース）
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
 * application 分離後:
 * - rows / totalPassed / totalQuantity / title の構築は buildInspectionResultCardData に移譲
 * - ModelVariationForMintDTO -> MintModelMetaEntry の変換は modelVariationMapper に移譲
 * - hook 側は不足 modelMeta の HTTP 補完と React 状態管理のみ担当
 */
export function useInspectionResultCard(
  params: UseInspectionResultCardParams,
): UseInspectionResultCardResult {
  const { batch } = params;

  // API から来ない/不足している modelMeta をここで補完する
  const [resolvedMeta, setResolvedMeta] = React.useState<
    Record<string, MintModelMetaEntry>
  >({});

  // batch が変わったら補完キャッシュもリセット（production 切替時など）
  React.useEffect(() => {
    setResolvedMeta({});
  }, [batch?.productionId, (batch as any)?.id, (batch as any)?.inspectionId]);

  // inspections からユニークな modelId を抽出
  const modelIds = React.useMemo(() => {
    return getInspectionModelIds(batch);
  }, [batch]);

  // 既存 meta（APIから来た分 + こちらで解決した分）をマージ
  const mergedModelMeta = React.useMemo<Record<string, MintModelMetaEntry>>(() => {
    return {
      ...(batch?.modelMeta ?? {}),
      ...(resolvedMeta ?? {}),
    };
  }, [batch?.modelMeta, resolvedMeta]);

  // まだ meta が無い modelId だけを抽出
  const missingModelIds = React.useMemo(() => {
    return getMissingModelIds({
      modelIds,
      modelMeta: mergedModelMeta,
    });
  }, [modelIds, mergedModelMeta]);

  // missingModelIds を GetModelVariationByID（HTTP）で解決して modelMeta を埋める
  React.useEffect(() => {
    if (!batch) return;
    if (missingModelIds.length === 0) return;

    let cancelled = false;

    (async () => {
      const settled = await Promise.all(
        missingModelIds.map(async (modelId) => {
          try {
            const variation = await fetchModelVariationByIdForMintHTTP(modelId);
            return { modelId, variation };
          } catch {
            return {
              modelId,
              variation: null as ModelVariationForMintDTO | null,
            };
          }
        }),
      );

      if (cancelled) return;

      setResolvedMeta((prev) => {
        const next = { ...(prev ?? {}) };

        for (const item of settled) {
          const meta = toMintModelMetaEntry(item.variation);
          if (!meta) continue;

          next[item.modelId] = meta;
        }

        return next;
      });
    })();

    return () => {
      cancelled = true;
    };
    // missingModelIds は配列なので、内容変化を見るため string 化する
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [batch, JSON.stringify(missingModelIds)]);

  const cardData = React.useMemo(() => {
    return buildInspectionResultCardData({
      batch,
      resolvedMeta,
    });
  }, [batch, resolvedMeta]);

  // ✅ 共通 util を使用して RGB → HEX (#RRGGBB) 変換
  const rgbIntToHex = React.useCallback(
    (rgb: number | string | null | undefined): string | undefined => {
      return rgbIntToHexShared(rgb);
    },
    [],
  );

  return {
    title: cardData.title,
    rows: cardData.rows,
    totalPassed: cardData.totalPassed,
    totalQuantity: cardData.totalQuantity,
    rgbIntToHex,
  };
}