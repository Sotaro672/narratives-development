// frontend/console/mintRequest/src/application/mapper/buildInspectionResultCardData.ts

import type { InspectionBatch } from "../../domain/entity/inspections";
import type {
  InspectionResultRow,
  MintModelMetaEntry,
} from "../../presentation/hook/useInspectionResultCard";

export type ProductBlueprintModelRefLike = {
  modelId?: string | null;
  displayOrder?: number | null;
};

export type InspectionBatchForCard = InspectionBatch & {
  productName?: string | null;
  modelMeta?: Record<string, MintModelMetaEntry> | null;
  productBlueprintPatch?: {
    modelRefs?: ProductBlueprintModelRefLike[] | null;
    [k: string]: any;
  } | null;
};

export type BuildInspectionResultCardDataInput = {
  batch: InspectionBatchForCard | null | undefined;
  resolvedMeta?: Record<string, MintModelMetaEntry> | null;
};

export type InspectionResultCardData = {
  title: string;
  rows: InspectionResultRow[];
  totalPassed: number;
  totalQuantity: number;
};

function buildDisplayOrderByModelId(
  modelRefs: ProductBlueprintModelRefLike[] | null | undefined,
): Record<string, number> {
  const out: Record<string, number> = {};

  for (const ref of modelRefs ?? []) {
    const modelId = String(ref?.modelId ?? "").trim();
    const displayOrder = ref?.displayOrder;

    if (!modelId) continue;
    if (typeof displayOrder !== "number") continue;
    if (!Number.isFinite(displayOrder)) continue;

    out[modelId] = displayOrder;
  }

  return out;
}

function buildMergedModelMeta(
  batchMeta: Record<string, MintModelMetaEntry> | null | undefined,
  resolvedMeta: Record<string, MintModelMetaEntry> | null | undefined,
): Record<string, MintModelMetaEntry> {
  return {
    ...(batchMeta ?? {}),
    ...(resolvedMeta ?? {}),
  };
}

export function getInspectionModelIds(
  batch: InspectionBatch | null | undefined,
): string[] {
  if (!batch?.inspections) return [];

  const set = new Set<string>();

  for (const inspection of batch.inspections ?? []) {
    const modelId = String((inspection as any)?.modelId ?? "").trim();
    if (modelId) set.add(modelId);
  }

  return Array.from(set);
}

export function getMissingModelIds(input: {
  modelIds: string[];
  modelMeta: Record<string, MintModelMetaEntry>;
}): string[] {
  const modelIds = input.modelIds ?? [];
  const modelMeta = input.modelMeta ?? {};

  return modelIds.filter((modelId) => !modelMeta[modelId]);
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
    };
  }

  const mergedModelMeta = buildMergedModelMeta(
    batch.modelMeta,
    input.resolvedMeta,
  );

  const displayOrderByModelId = buildDisplayOrderByModelId(
    batch.productBlueprintPatch?.modelRefs,
  );

  const aggregation = new Map<
    string,
    {
      modelNumber: string;
      passed: number;
      total: number;
    }
  >();

  for (const inspection of batch.inspections ?? []) {
    const modelId = String((inspection as any)?.modelId ?? "").trim();
    if (!modelId) continue;

    const modelNumberFromInspection = String(
      (inspection as any)?.modelNumber ?? "",
    ).trim();

    const entry =
      aggregation.get(modelId) ?? {
        modelNumber: modelNumberFromInspection,
        passed: 0,
        total: 0,
      };

    entry.total += 1;

    if ((inspection as any)?.inspectionResult === "passed") {
      entry.passed += 1;
    }

    if (!entry.modelNumber && modelNumberFromInspection) {
      entry.modelNumber = modelNumberFromInspection;
    }

    aggregation.set(modelId, entry);
  }

  const rowsWithOrder: Array<InspectionResultRow & { __order: number }> = [];
  const INF = Number.POSITIVE_INFINITY;

  for (const [modelId, agg] of aggregation.entries()) {
    const meta = mergedModelMeta[modelId];

    const displayModelNumber = meta?.modelNumber || agg.modelNumber || modelId;

    const order =
      typeof displayOrderByModelId[modelId] === "number" &&
      Number.isFinite(displayOrderByModelId[modelId])
        ? displayOrderByModelId[modelId]
        : INF;

    rowsWithOrder.push({
      __order: order,
      modelNumber: displayModelNumber,
      size: meta?.size ?? "",
      color: meta?.colorName ?? "",
      rgb: meta?.rgb ?? null,
      passedQuantity: agg.passed,
      quantity: agg.total,
    });
  }

  rowsWithOrder.sort((a, b) => a.__order - b.__order);

  const rows = rowsWithOrder.map(({ __order, ...row }) => row);

  const totalPassed = rows.reduce(
    (sum, row) => sum + (row.passedQuantity || 0),
    0,
  );

  const totalQuantity = rows.reduce(
    (sum, row) => sum + (row.quantity || 0),
    0,
  );

  const productName = String((batch as any)?.productName ?? "").trim();

  return {
    title: productName ? `検査結果：${productName}` : "モデル別検査結果",
    rows,
    totalPassed,
    totalQuantity,
  };
}