// frontend/console/mintRequest/src/infrastructure/repository/http/modelVariations.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { ModelVariationForMintDTO } from "../../dto/mintRequestLocal.dto";

type VolumeRaw = {
  Value?: unknown;
  Unit?: unknown;
};

type ModelVariationForMintRaw = {
  ID?: unknown;
  ProductBlueprintID?: unknown;
  ModelNumber?: unknown;
  Size?: unknown;
  ColorName?: unknown;
  RGB?: unknown;
  Volume?: VolumeRaw | null;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === "object" && !Array.isArray(value);
}

function toText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function toNullableText(value: unknown): string | null {
  const text = toText(value);
  return text || null;
}

function toNumberOrNull(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const text = value.trim();
    if (!text) return null;

    const parsed = Number(text);
    return Number.isFinite(parsed) ? parsed : null;
  }

  return null;
}

function toVolume(value: unknown): string | number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const text = value.trim();
    return text || null;
  }

  return null;
}

function toModelVariationForMintDTO(
  value: unknown,
): ModelVariationForMintDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw = value as ModelVariationForMintRaw;

  const id = toText(raw.ID);
  if (!id) return null;

  return {
    id,
    modelNumber: toNullableText(raw.ModelNumber),
    size: toNullableText(raw.Size),
    colorName: toNullableText(raw.ColorName),
    rgb: toNumberOrNull(raw.RGB),
    volume: toVolume(raw.Volume?.Value),
    volumeUnit: toNullableText(raw.Volume?.Unit),
  };
}

export async function fetchModelVariationByIdForMintHTTP(
  variationId: string,
): Promise<ModelVariationForMintDTO | null> {
  const vid = String(variationId ?? "").trim();
  if (!vid) throw new Error("variationId が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/models/${encodeURIComponent(vid)}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");

    throw new Error(
      `Failed to fetch model variation for mint: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as unknown;

  return toModelVariationForMintDTO(json);
}