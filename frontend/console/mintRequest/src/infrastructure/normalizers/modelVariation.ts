// frontend/console/mintRequest/src/infrastructure/normalizers/modelVariation.ts

import type { ModelVariationForMintDTO } from "../dto/mintRequestLocal.dto";

export function normalizeModelVariationForMintDTO(
  v: any,
): ModelVariationForMintDTO | null {
  if (!v) return null;

  const id = String(v.id ?? "").trim();
  if (!id) return null;

  const modelNumber = String(v.modelNumber ?? "").trim() || null;
  const size = String(v.size ?? "").trim() || null;

  const colorObj = v.color ?? null;

  const colorName = String(colorObj?.name ?? "").trim() || null;

  const rgbRaw = colorObj?.rgb ?? null;
  const rgb =
    typeof rgbRaw === "number"
      ? rgbRaw
      : Number.isFinite(Number(rgbRaw))
        ? Number(rgbRaw)
        : null;

  return { id, modelNumber, size, colorName, rgb };
}
