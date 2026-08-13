// frontend/console/mintRequest/src/application/mapper/buildInspectionResultCardData.ts 
 
import type { 
  InspectionBatch, 
  MintModelMeta, 
} from "../../../../shared/types/inspections"; 
 
import type { 
  ProductBlueprintPatchDTO, 
} from "../../infrastructure/dto/mintRequestLocal.dto"; 
 
export type InspectionResultRow = { 
  modelNumber: string; 
  size: string; 
  color: string; 
  rgb?: number | string | null; 
 
  /** 
   * alcohol 対応: 
   * showVolumeColumn=true の場合、InspectionResultCard 側で volumeLabel を表示する。 
   */ 
  volume?: string | number | null; 
  volumeUnit?: string | null; 
  volumeLabel?: string; 
 
  passedQuantity: number; 
  quantity: number; 
}; 
 
export type InspectionBatchForCard = InspectionBatch & { 
  productName?: string | null; 
 
  /** 
   * modelId -> model meta 
   * 
   * Backend response の modelMeta を正として使用する。 
   * frontend から Model Variation の個別補完は行わない。 
   */ 
  modelMeta?: Record<string, MintModelMeta> | null; 
 
  /** 
   * ProductBlueprintPatch。 
   * 
   * - modelRefs は displayOrder の唯一のソース 
   * - productBlueprintCategory.kind で alcohol などの表示切替を行う 
   */ 
  productBlueprintPatch?: Pick< 
    ProductBlueprintPatchDTO, 
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
 
  /** 
   * productBlueprintCategory.kind。 
   * 現状は alcohol の場合に検品結果カードで容量列を表示する。 
   */ 
  categoryKind: string; 
 
  /** 
   * true の場合、InspectionResultCard 側で サイズ/カラー ではなく 容量 を表示する。 
   */ 
  showVolumeColumn: boolean; 
}; 
 
function toText(value: unknown): string { 
  if (value === null || value === undefined) { 
    return ""; 
  } 
 
  if (typeof value === "string") { 
    return value.trim(); 
  } 
 
  if ( 
    typeof value === "number" || 
    typeof value === "boolean" 
  ) { 
    return String(value); 
  } 
 
  return ""; 
} 
 
/** 
 * modelRefsはInfrastructure層で正規化済みのため、 
 * Application層では再検証せず、そのまま対応表へ変換する。 
 */ 
function buildDisplayOrderByModelId( 
  modelRefs: ProductBlueprintPatchDTO["modelRefs"], 
): Record<string, number> { 
  return Object.fromEntries( 
    (modelRefs ?? []).map((ref) => [ 
      ref.modelId, 
      ref.displayOrder, 
    ]), 
  ); 
} 
 
function resolveCategoryKind( 
  batch: InspectionBatchForCard | null | undefined, 
): string { 
  return toText( 
    batch?.productBlueprintPatch 
      ?.productBlueprintCategory?.kind, 
  ); 
} 
 
function buildVolumeLabel(params: { 
  volume: string | number | null | undefined; 
  volumeUnit: string | null | undefined; 
  isAlcohol: boolean; 
}): string { 
  const { 
    volume, 
    volumeUnit, 
    isAlcohol, 
  } = params; 
 
  if (!isAlcohol) { 
    return ""; 
  } 
 
  const volumeText = toText(volume); 
 
  if (!volumeText) { 
    return ""; 
  } 
 
  const unitText = 
    toText(volumeUnit) || "ml"; 
 
  return `${volumeText}${unitText}`; 
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
 
  const categoryKind = 
    resolveCategoryKind(batch); 
 
  const isAlcohol = 
    categoryKind === "alcohol"; 
 
  /** 
   * Model情報はBackend responseのmodelMetaを正とする。 
   * frontend側で個別Model Variation APIによる補完は行わない。 
   */ 
  const modelMeta = 
    batch.modelMeta ?? {}; 
 
  const displayOrderByModelId = 
    buildDisplayOrderByModelId( 
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
 
  for (const inspection of batch.inspections) { 
    const modelId = inspection.modelId; 
 
    if (!modelId) { 
      continue; 
    } 
 
    const modelNumberFromInspection = 
      toText(inspection.modelNumber); 
 
    const entry = 
      aggregation.get(modelId) ?? { 
        modelNumber: modelNumberFromInspection, 
        passed: 0, 
        total: 0, 
      }; 
 
    entry.total += 1; 
 
    if ( 
      inspection.inspectionResult === "passed" 
    ) { 
      entry.passed += 1; 
    } 
 
    if ( 
      !entry.modelNumber && 
      modelNumberFromInspection 
    ) { 
      entry.modelNumber = 
        modelNumberFromInspection; 
    } 
 
    aggregation.set(modelId, entry); 
  } 
 
  const rowsWithOrder: Array< 
    InspectionResultRow & { 
      __order: number; 
    } 
  > = []; 
 
  const fallbackOrder = 
    Number.POSITIVE_INFINITY; 
 
  for ( 
    const [modelId, aggregated] of 
    aggregation.entries() 
  ) { 
    const meta = 
      modelMeta[modelId]; 
 
    const displayModelNumber = 
      meta?.modelNumber || 
      aggregated.modelNumber || 
      modelId; 
 
    const order = 
      displayOrderByModelId[modelId] ?? 
      fallbackOrder; 
 
    const volume = 
      meta?.volume ?? null; 
 
    const volumeUnit = 
      meta?.volumeUnit ?? null; 
 
    const volumeLabel = 
      buildVolumeLabel({ 
        volume, 
        volumeUnit, 
        isAlcohol, 
      }); 
 
    rowsWithOrder.push({ 
      __order: order, 
      modelNumber: displayModelNumber, 
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
    if (a.__order !== b.__order) { 
      return a.__order - b.__order; 
    } 
 
    return a.modelNumber.localeCompare( 
      b.modelNumber, 
    ); 
  }); 
 
  const rows = rowsWithOrder.map( 
    ({ __order, ...row }) => row, 
  ); 
 
  const totalPassed = rows.reduce( 
    (sum, row) => 
      sum + row.passedQuantity, 
    0, 
  ); 
 
  const totalQuantity = rows.reduce( 
    (sum, row) => 
      sum + row.quantity, 
    0, 
  ); 
 
  const productName = 
    toText(batch.productName); 
 
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