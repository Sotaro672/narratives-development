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
  rgbIntToHex as rgbIntToHexShared,
} from "../../../../shared/util/color";

export type UseInspectionResultCardParams = {
  batch:
    | InspectionBatchForCard
    | null
    | undefined;
};

export type UseInspectionResultCardResult = {
  title: string;

  rows:
    InspectionResultRow[];

  totalPassed: number;
  totalQuantity: number;

  categoryKind: string;

  showVolumeColumn: boolean;

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
 * model情報はBackend responseのmodelMetaを正とする。
 * frontendからGET /models/{modelId}による個別補完は行わない。
 */
export function useInspectionResultCard({
  batch,
}: UseInspectionResultCardParams): UseInspectionResultCardResult {
  const cardData =
    React.useMemo(() => {
      return buildInspectionResultCardData({
        batch,
      });
    }, [
      batch,
    ]);

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