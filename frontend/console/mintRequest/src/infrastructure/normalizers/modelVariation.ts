// frontend/console/mintRequest/src/infrastructure/normalizers/modelVariation.ts

import type { ModelVariationForMintDTO } from "../dto/mintRequestLocal.dto";

export function normalizeModelVariationForMintDTO(
  v: any,
): ModelVariationForMintDTO | null {
  if (!v) return null;

  const id = String(v?.id ?? v?.ID ?? "").trim();
  if (!id) return null;

  const modelNumber = String(v?.modelNumber ?? v?.ModelNumber ?? "").trim() || null;
  const size = String(v?.size ?? v?.Size ?? "").trim() || null;

  const colorObj = v?.color ?? v?.Color ?? null;

  const colorName =
    String(
      v?.colorName ?? v?.ColorName ?? colorObj?.name ?? colorObj?.Name ?? "",
    ).trim() || null;

  const rgbRaw = v?.rgb ?? v?.RGB ?? colorObj?.rgb ?? colorObj?.RGB ?? null;

  const rgb =
    typeof rgbRaw === "number"
      ? rgbRaw
      : Number.isFinite(Number(rgbRaw))
        ? Number(rgbRaw)
        : null;

  return { id, modelNumber, size, colorName, rgb };
}
