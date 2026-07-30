// frontend/console/shell/src/features/mintRequest/presentation/hook/useInspectionResultCard.tsx

import * as React from "react";

import {
  buildInspectionResultCardData,
} from "../../application/mapper/buildInspectionResultCardData";

import type {
  InspectionBatchForCard,
  InspectionResultRow,
} from "../../application/mapper/buildInspectionResultCardData";

import {
  resolveInspectionModelMeta,
} from "../../application/usecase/resolveInspectionModelMeta";

import type {
  InspectionModelMetaRepository,
  ResolveInspectionModelMetaResult,
} from "../../application/usecase/resolveInspectionModelMeta";

import {
  rgbIntToHex as rgbIntToHexShared,
} from "../../../../shared/util/color";

export type UseInspectionResultCardParams = {
  /**
   * MintInspectionView相当。
   *
   * InspectionBatch、modelMeta、
   * productBlueprintPatchを含む。
   */
  batch:
    | InspectionBatchForCard
    | null
    | undefined;

  /**
   * 不足しているModel Variationを取得するための
   * Application層のRepository契約。
   *
   * Presentation層からHTTP関数を直接呼び出さない。
   */
  modelMetaRepository:
    InspectionModelMetaRepository;
};

export type UseInspectionResultCardResult = {
  title: string;

  rows:
    InspectionResultRow[];

  totalPassed: number;
  totalQuantity: number;

  /**
   * productBlueprintCategory.kind。
   *
   * alcoholの場合は検品結果カードで
   * 容量列を表示する。
   */
  categoryKind: string;

  /**
   * trueの場合、サイズ・カラーではなく
   * 容量を表示する。
   */
  showVolumeColumn: boolean;

  /**
   * RGB整数値を#RRGGBB形式へ変換する。
   */
  rgbIntToHex: (
    rgb:
      | number
      | string
      | null
      | undefined,
  ) => string | undefined;
};

/**
 * 検品結果カードで使用するデータを提供する。
 *
 * Presentation層の責務:
 * - React stateの管理
 * - Application UseCaseの実行
 * - カード表示データのメモ化
 * - 表示用カラー変換関数の提供
 *
 * Application層の責務:
 * - 不足しているmodelIdの判定
 * - Model Variationの取得
 * - Model VariationからmodelMetaへの変換
 * - 行データと集計値の構築
 */
export function useInspectionResultCard({
  batch,
  modelMetaRepository,
}: UseInspectionResultCardParams): UseInspectionResultCardResult {
  /**
   * Application UseCaseによって補完された
   * modelIdごとのモデル情報。
   */
  const [
    resolvedMeta,
    setResolvedMeta,
  ] =
    React.useState<ResolveInspectionModelMetaResult>(
      {},
    );

  /**
   * batchが変更された場合は補完結果をリセットし、
   * 不足しているモデル情報をApplication UseCaseで取得する。
   *
   * productionIdを正とし、
   * idやinspectionIdへのフォールバックは行わない。
   */
  React.useEffect(() => {
    setResolvedMeta({});

    if (!batch) {
      return;
    }

    let cancelled = false;

    const run = async () => {
      try {
        const nextResolvedMeta =
          await resolveInspectionModelMeta(
            modelMetaRepository,
            {
              batch,
              resolvedMeta: null,
            },
          );

        if (cancelled) {
          return;
        }

        setResolvedMeta(
          nextResolvedMeta,
        );
      } catch {
        if (!cancelled) {
          /**
           * Model Variationの補完失敗では
           * 画面全体をエラーにしない。
           *
           * Backendから取得済みのbatch.modelMetaだけで
           * カードを構築する。
           */
          setResolvedMeta({});
        }
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [
    batch,
    modelMetaRepository,
  ]);

  /**
   * 行データ・集計値・カテゴリ表示条件の構築は
   * Application Mapperへ委譲する。
   */
  const cardData =
    React.useMemo(() => {
      return buildInspectionResultCardData({
        batch,
        resolvedMeta,
      });
    }, [
      batch,
      resolvedMeta,
    ]);

  /**
   * RGBからHEXへの変換は表示処理なので、
   * Presentation層の共通Utilityを使用する。
   */
  const rgbIntToHex =
    React.useCallback(
      (
        rgb:
          | number
          | string
          | null
          | undefined,
      ): string | undefined => {
        return rgbIntToHexShared(
          rgb,
        );
      },
      [],
    );

  return {
    title:
      cardData.title,

    rows:
      cardData.rows,

    totalPassed:
      cardData.totalPassed,

    totalQuantity:
      cardData.totalQuantity,

    categoryKind:
      cardData.categoryKind,

    showVolumeColumn:
      cardData.showVolumeColumn,

    rgbIntToHex,
  };
}