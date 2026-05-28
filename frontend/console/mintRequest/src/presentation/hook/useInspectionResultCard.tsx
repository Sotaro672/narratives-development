// frontend/console/mintRequest/src/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";

import {
  buildInspectionResultCardData,
  getInspectionModelIds,
  getMissingModelIds,
  InspectionBatchForCard,
  InspectionResultRow,
} from "../../application/mapper/buildInspectionResultCardData";

import { toMintModelMetaEntry } from "../../application/mapper/modelVariationMapper";

// ★ modelId -> ModelVariation を引く（modelNumber/size/color/volume を解決する）
import { fetchModelVariationByIdForMintHTTP } from "../../infrastructure/repository";

import type {
  MintModelMetaEntryDTO,
  ModelVariationForMintDTO,
} from "../../infrastructure/dto/mintRequestLocal.dto";

// ✅ 共通カラー変換ユーティリティ（rgb int -> "#RRGGBB"）
import { rgbIntToHex as rgbIntToHexShared } from "../../../../shell/src/shared/util/color";

export type UseInspectionResultCardParams = {
  /**
   * MintInspectionView 相当。
   *
   * InspectionBatch + modelMeta + productBlueprintPatch を含む、
   * 検品結果カード構築用の application input 型。
   */
  batch: InspectionBatchForCard | null | undefined;
};

export type UseInspectionResultCardResult = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;

  /**
   * productBlueprintCategory.kind。
   * 現状は alcohol の場合に検品結果カードで容量列を表示する。
   */
  categoryKind: string;

  /**
   * true の場合、InspectionResultCard 側で サイズ/カラー ではなく 容量 を表示する。
   */
  showVolumeColumn: boolean;

  // ✅ 共通 util 互換に合わせる：undefined も返しうる
  rgbIntToHex: (rgb: number | string | null | undefined) => string | undefined;
};

/**
 * InspectionBatch（+ modelMeta + productBlueprintPatch）から
 * InspectionResultCard 用の行データ・集計値・RGB変換関数を提供するフック。
 *
 * 責務:
 * - hook 側は不足 modelMeta の HTTP 補完と React 状態管理のみ担当
 * - rows / totalPassed / totalQuantity / title / categoryKind / showVolumeColumn の構築は
 *   buildInspectionResultCardData に移譲
 * - ModelVariationForMintDTO -> MintModelMetaEntryDTO の変換は
 *   modelVariationMapper に移譲
 */
export function useInspectionResultCard(
  params: UseInspectionResultCardParams,
): UseInspectionResultCardResult {
  const { batch } = params;

  // API から来ない/不足している modelMeta をここで補完する
  const [resolvedMeta, setResolvedMeta] = React.useState<
    Record<string, MintModelMetaEntryDTO>
  >({});

  // batch が変わったら補完キャッシュもリセット（production 切替時など）
  React.useEffect(() => {
    setResolvedMeta({});
  }, [batch?.productionId, (batch as any)?.id, (batch as any)?.inspectionId]);

  // inspections からユニークな modelId を抽出
  const modelIds = React.useMemo(() => {
    return getInspectionModelIds(batch);
  }, [batch]);

  // 既存 meta（API から来た分 + こちらで解決した分）をマージ
  const mergedModelMeta = React.useMemo<
    Record<string, MintModelMetaEntryDTO>
  >(() => {
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
        const next: Record<string, MintModelMetaEntryDTO> = {
          ...(prev ?? {}),
        };

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
    categoryKind: cardData.categoryKind,
    showVolumeColumn: cardData.showVolumeColumn,
    rgbIntToHex,
  };
}