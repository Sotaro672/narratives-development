// frontend/console/production/src/presentation/viewModels/normalizeProductionModels.ts

export type NormalizedProductionModel = {
  modelId: string;
  quantity: number;
  modelNumber?: string;
  size?: string;
  color?: string;
  rgb?: number | string | null;
  displayOrder?: number;
};

/**
 * backend の揺れ（PascalCase / camelCase 混在）を吸収して
 * ViewModel Builder に渡すための “正規化済みモデル行” を生成する。
 */
export function normalizeProductionModels(raw: any[]): NormalizedProductionModel[] {
  const safe = Array.isArray(raw) ? raw : [];

  return safe.map((m: any, index: number) => {
    const modelIdRaw =
      m?.modelId ?? m?.ModelID ?? m?.ModelId ?? m?.modelID ?? "";
    const quantityRaw = m?.quantity ?? m?.Quantity ?? 0;

    const modelId = String(modelIdRaw ?? "").trim() || String(index);

    const quantity = Number.isFinite(Number(quantityRaw))
      ? Math.max(0, Math.floor(Number(quantityRaw)))
      : 0;

    const modelNumberCandidate = m?.modelNumber ?? m?.ModelNumber;
    const modelNumber =
      typeof modelNumberCandidate === "string"
        ? modelNumberCandidate.trim()
        : undefined;

    const sizeCandidate = m?.size ?? m?.Size;
    const size =
      typeof sizeCandidate === "string" ? sizeCandidate.trim() : undefined;

    const colorCandidate = m?.color ?? m?.Color;
    const color =
      typeof colorCandidate === "string" ? colorCandidate.trim() : undefined;

    const rgbCandidate = m?.rgb ?? m?.RGB;
    const rgb =
      typeof rgbCandidate === "number" || typeof rgbCandidate === "string"
        ? rgbCandidate
        : null;

    const displayOrderCandidate = m?.displayOrder ?? m?.DisplayOrder;
    const displayOrder =
      typeof displayOrderCandidate === "number"
        ? displayOrderCandidate
        : Number.isFinite(Number(displayOrderCandidate))
          ? Number(displayOrderCandidate)
          : undefined;

    return {
      modelId,
      quantity,
      modelNumber,
      size,
      color,
      rgb,
      displayOrder,
    };
  });
}
