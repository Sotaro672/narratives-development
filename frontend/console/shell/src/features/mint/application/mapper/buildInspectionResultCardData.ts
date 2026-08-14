// frontend/console/shell/src/features/mint/application/mapper/buildInspectionResultCardData.ts

import type { InspectionBatch, MintModelMeta } from "../../../../shared/types/inspections";
import type { MintProductBlueprintDTO } from "../../infrastructure/dto/mintRequestLocal.dto";

export type InspectionResultRow = {
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

export type InspectionBatchForCard = InspectionBatch & {
  productName?: string | null;

  /**
   * modelId -> model meta。
   * Backend responseのmodelMetaをモデル表示情報の唯一の正として使用する。
   */
  modelMeta?: Record<string, MintModelMeta> | null;

  /**
   * GET /mint/product_blueprints/{productBlueprintId} のBackend BFF response。
   * modelRefsはdisplayOrderの唯一のソースとし、
   * productBlueprintCategory.kindで表示を切り替える。
   */
  productBlueprint?: Pick<
    MintProductBlueprintDTO,
    "modelRefs" | "productBlueprintCategory"
  > | null;
};

export type BuildInspectionResultCardDataInput = {
  batch: InspectionBatchForCard | null | undefined;
};

export type InspectionResultCardData = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;
  categoryKind: string;
  showVolumeColumn: boolean;
};

function toText(value: unknown): string {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  return "";
}

/**
 * Backend BFFのProductBlueprint.modelRefsを正としてdisplayOrder対応表を生成する。
 */
function buildDisplayOrderByModelId(
  modelRefs: MintProductBlueprintDTO["modelRefs"],
): Record<string, number> {
  return Object.fromEntries(
    (modelRefs ?? []).map((ref) => [ref.modelId, ref.displayOrder]),
  );
}

function resolveCategoryKind(
  batch: InspectionBatchForCard | null | undefined,
): string {
  return toText(batch?.productBlueprint?.productBlueprintCategory?.kind);
}

/**
 * Backendが返したvolume / volumeUnitだけを使用する。
 * volumeUnitが無い場合に"ml"などをFrontend側で補完しない。
 */
function buildVolumeLabel(params: {
  volume: string | number | null | undefined;
  volumeUnit: string | null | undefined;
  isAlcohol: boolean;
}): string {
  const { volume, volumeUnit, isAlcohol } = params;
  if (!isAlcohol) return "";

  const volumeText = toText(volume);
  if (!volumeText) return "";

  const unitText = toText(volumeUnit);
  return unitText ? `${volumeText}${unitText}` : volumeText;
}

export function buildInspectionResultCardData(
  input: BuildInspectionResultCardDataInput,
): InspectionResultCardData {
  const batch = input.batch ?? null;

  if (!batch) {
    return {
      title: "モデル別検査結果",
      rows: [],
      totalPassed: 0,
      totalQuantity: 0,
      categoryKind: "",
      showVolumeColumn: false,
    };
  }

  const categoryKind = resolveCategoryKind(batch);
  const isAlcohol = categoryKind === "alcohol";

  /**
   * modelNumber / size / color / rgb / volume / volumeUnitは
   * Backend responseのmodelMetaだけを正とする。
   */
  const modelMeta = batch.modelMeta ?? {};
  const displayOrderByModelId = buildDisplayOrderByModelId(
    batch.productBlueprint?.modelRefs,
  );

  const aggregation = new Map<string, { passed: number; total: number }>();

  for (const inspection of batch.inspections) {
    const modelId = inspection.modelId;
    if (!modelId) continue;

    const entry = aggregation.get(modelId) ?? { passed: 0, total: 0 };
    entry.total += 1;

    if (inspection.inspectionResult === "passed") {
      entry.passed += 1;
    }

    aggregation.set(modelId, entry);
  }

  const rowsWithOrder: Array<
    InspectionResultRow & { __order: number }
  > = [];

  const fallbackOrder = Number.POSITIVE_INFINITY;

  for (const [modelId, aggregated] of aggregation.entries()) {
    const meta = modelMeta[modelId];
    const modelNumber = toText(meta?.modelNumber);
    const order = displayOrderByModelId[modelId] ?? fallbackOrder;
    const volume = meta?.volume ?? null;
    const volumeUnit = meta?.volumeUnit ?? null;
    const volumeLabel = buildVolumeLabel({
      volume,
      volumeUnit,
      isAlcohol,
    });

    rowsWithOrder.push({
      __order: order,
      modelNumber,
      size: meta?.size ?? "",
      color: meta?.colorName ?? "",
      rgb: meta?.rgb ?? null,
      volume,
      volumeUnit,
      volumeLabel,
      passedQuantity: aggregated.passed,
      quantity: aggregated.total,
    });
  }

  rowsWithOrder.sort((a, b) => {
    if (a.__order !== b.__order) return a.__order - b.__order;
    return a.modelNumber.localeCompare(b.modelNumber);
  });

  const rows = rowsWithOrder.map(({ __order, ...row }) => row);
  const totalPassed = rows.reduce(
    (sum, row) => sum + row.passedQuantity,
    0,
  );
  const totalQuantity = rows.reduce(
    (sum, row) => sum + row.quantity,
    0,
  );
  const productName = toText(batch.productName);

  return {
    title: productName
      ? `検査結果：${productName}`
      : "モデル別検査結果",
    rows,
    totalPassed,
    totalQuantity,
    categoryKind,
    showVolumeColumn: isAlcohol,
  };
}