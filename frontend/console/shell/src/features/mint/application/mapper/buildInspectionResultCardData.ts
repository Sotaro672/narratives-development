// frontend/console/shell/src/features/mint/application/mapper/buildInspectionResultCardData.ts

import type { InspectionBatch, MintModelMeta } from "../../../../shared/types/inspections";
import type { MintProductBlueprintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

type InspectionResultRow = {
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  volume?: string | number | null;
  volumeUnit?: string | null;
  volumeLabel?: string;
  passedQuantity: number;
  quantity: number;
};

export type BuildInspectionResultCardDataInput = {
  inspection: InspectionBatch | null | undefined;
  productName: string;
  modelMeta: Record<string, MintModelMeta>;
  productBlueprint:
    | Pick<MintProductBlueprintDTO, "modelRefs" | "productBlueprintCategory">
    | null
    | undefined;
};

export type InspectionResultCardData = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;
  categoryKind: string;
  showVolumeColumn: boolean;
};

/**
 * Backendが返したvolume / volumeUnitだけを使用して表示文字列を生成する。
 * volumeUnitのFrontend側補完は行わない。
 */
function buildVolumeLabel(
  volume: number | null | undefined,
  volumeUnit: string | null | undefined,
): string {
  if (volume === null || volume === undefined) return "";
  return `${volume}${volumeUnit ?? ""}`;
}

/**
 * Backend BFF responseから検品結果カード用ViewModelを生成する。
 *
 * Backendを正とする値:
 * - productName
 * - modelMeta
 * - inspection.totalPassed
 * - inspection.quantity
 * - productBlueprint.productBlueprintCategory.kind
 * - productBlueprint.modelRefs.displayOrder
 *
 * Frontendではモデル単位のpassed / total集計と表示用volumeLabel生成のみを行う。
 */
export function buildInspectionResultCardData(
  input: BuildInspectionResultCardDataInput,
): InspectionResultCardData {
  const inspection = input.inspection;

  if (!inspection) {
    return {
      title: "モデル別検査結果",
      rows: [],
      totalPassed: 0,
      totalQuantity: 0,
      categoryKind: "",
      showVolumeColumn: false,
    };
  }

  const categoryKind = input.productBlueprint?.productBlueprintCategory?.kind ?? "";
  const isAlcohol = categoryKind === "alcohol";
  const aggregation = new Map<string, { passed: number; total: number }>();

  for (const item of inspection.inspections) {
    if (!item.modelId) continue;

    const aggregated = aggregation.get(item.modelId) ?? { passed: 0, total: 0 };
    aggregated.total += 1;

    if (item.inspectionResult === "passed") {
      aggregated.passed += 1;
    }

    aggregation.set(item.modelId, aggregated);
  }

  const rowsByModelId = new Map<string, InspectionResultRow>();

  for (const [modelId, aggregated] of aggregation.entries()) {
    const meta = input.modelMeta[modelId];

    if (!meta) {
      throw new Error(`modelMeta が存在しません: modelId=${modelId}`);
    }

    const volume = meta.volume ?? null;
    const volumeUnit = meta.volumeUnit ?? null;

    rowsByModelId.set(modelId, {
      modelNumber: meta.modelNumber ?? "",
      size: meta.size ?? "",
      color: meta.colorName ?? "",
      rgb: meta.rgb ?? null,
      volume,
      volumeUnit,
      volumeLabel: isAlcohol ? buildVolumeLabel(volume, volumeUnit) : "",
      passedQuantity: aggregated.passed,
      quantity: aggregated.total,
    });
  }

  let rows: InspectionResultRow[];

  if (input.productBlueprint) {
    const modelRefs = input.productBlueprint.modelRefs;

    if (!modelRefs) {
      throw new Error("productBlueprint.modelRefs が存在しません");
    }

    rows = [...modelRefs]
      .sort((a, b) => a.displayOrder - b.displayOrder)
      .flatMap((modelRef) => {
        const row = rowsByModelId.get(modelRef.modelId);
        return row ? [row] : [];
      });
  } else {
    rows = [...rowsByModelId.values()];
  }

  return {
    title: `検査結果：${input.productName}`,
    rows,
    totalPassed: inspection.totalPassed,
    totalQuantity: inspection.quantity,
    categoryKind,
    showVolumeColumn: isAlcohol,
  };
}