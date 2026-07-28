// frontend/console/shell/src/features/model/infrastructure/codec/modelVariationCodec.ts

/* =========================================================
 * Response types
 * =======================================================*/

export type ModelVariationKind =
  | "apparel"
  | "alcohol";

export type Volume = {
  value: number;
  unit: string;
};

export type ModelVariationColor = {
  name: string;
  rgb: number;
};

type ModelVariationResponseBase = {
  id: string;
  productBlueprintId: string;
  kind: ModelVariationKind;
  modelNumber: string;
  createdAt?: string;
  createdBy?: string;
  updatedAt?: string;
  updatedBy?: string;
};

export type ApparelModelVariationResponse =
  ModelVariationResponseBase & {
    kind: "apparel";
    size: string;
    color: ModelVariationColor;
    measurements?: Record<string, number>;
  };

export type AlcoholModelVariationResponse =
  ModelVariationResponseBase & {
    kind: "alcohol";
    volume: Volume;
  };

export type ModelVariationResponse =
  | ApparelModelVariationResponse
  | AlcoholModelVariationResponse;

/* =========================================================
 * Type guards
 * =======================================================*/

export function isApparelModelVariationResponse(
  variation: ModelVariationResponse,
): variation is ApparelModelVariationResponse {
  return variation.kind === "apparel";
}

export function isAlcoholModelVariationResponse(
  variation: ModelVariationResponse,
): variation is AlcoholModelVariationResponse {
  return variation.kind === "alcohol";
}

/* =========================================================
 * Generic helpers
 * =======================================================*/

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    value !== null &&
    typeof value === "object" &&
    !Array.isArray(value)
  );
}

function requireNonEmptyString(
  value: unknown,
  fieldName: string,
): string {
  if (
    typeof value !== "string" ||
    value.length === 0
  ) {
    throw new Error(
      `modelVariationCodec: ${fieldName}が空または文字列ではありません`,
    );
  }

  return value;
}

function optionalString(
  value: unknown,
  fieldName: string,
): string | undefined {
  if (
    value === undefined ||
    value === null
  ) {
    return undefined;
  }

  if (typeof value !== "string") {
    throw new Error(
      `modelVariationCodec: ${fieldName}が文字列ではありません`,
    );
  }

  return value;
}

function requireInteger(
  value: unknown,
  fieldName: string,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value) ||
    !Number.isInteger(value)
  ) {
    throw new Error(
      `modelVariationCodec: ${fieldName}が整数ではありません`,
    );
  }

  return value;
}

/* =========================================================
 * Response envelope helpers
 * =======================================================*/

function unwrapSingleResponse(
  value: unknown,
): unknown {
  if (!isRecord(value)) {
    return value;
  }

  if (isRecord(value.modelVariation)) {
    return value.modelVariation;
  }

  if (isRecord(value.item)) {
    return value.item;
  }

  if (isRecord(value.data)) {
    if (
      isRecord(
        value.data.modelVariation,
      )
    ) {
      return value.data.modelVariation;
    }

    if (isRecord(value.data.item)) {
      return value.data.item;
    }

    if (
      typeof value.data.id === "string" ||
      typeof value.data.kind === "string"
    ) {
      return value.data;
    }
  }

  return value;
}

function unwrapListResponse(
  value: unknown,
): unknown[] {
  if (Array.isArray(value)) {
    return value;
  }

  if (!isRecord(value)) {
    throw new Error(
      "modelVariationCodec: Model variation一覧レスポンスが配列ではありません",
    );
  }

  const directCandidates: unknown[] = [
    value.variations,
    value.modelVariations,
    value.items,
  ];

  for (const candidate of directCandidates) {
    if (Array.isArray(candidate)) {
      return candidate;
    }
  }

  if (Array.isArray(value.data)) {
    return value.data;
  }

  if (isRecord(value.data)) {
    const dataCandidates: unknown[] = [
      value.data.variations,
      value.data.modelVariations,
      value.data.items,
    ];

    for (const candidate of dataCandidates) {
      if (Array.isArray(candidate)) {
        return candidate;
      }
    }
  }

  throw new Error(
    "modelVariationCodec: Model variation一覧レスポンスに配列がありません",
  );
}

/* =========================================================
 * Field parsers
 * =======================================================*/

function parseMeasurements(
  value: unknown,
): Record<string, number> | undefined {
  if (
    value === undefined ||
    value === null
  ) {
    return undefined;
  }

  if (!isRecord(value)) {
    throw new Error(
      "modelVariationCodec: measurementsがobjectではありません",
    );
  }

  const measurements: Record<string, number> = {};

  for (
    const [key, rawValue]
    of Object.entries(value)
  ) {
    if (!key) {
      throw new Error(
        "modelVariationCodec: measurementsの項目名が空です",
      );
    }

    const measurementValue =
      requireInteger(
        rawValue,
        `measurements.${key}`,
      );

    if (measurementValue < 0) {
      throw new Error(
        `modelVariationCodec: measurements.${key}が0未満です`,
      );
    }

    measurements[key] =
      measurementValue;
  }

  return Object.keys(measurements).length > 0
    ? measurements
    : undefined;
}

function parseColor(
  value: Record<string, unknown>,
): ModelVariationColor {
  if (isRecord(value.color)) {
    const name =
      requireNonEmptyString(
        value.color.name,
        "color.name",
      );

    const rgb =
      requireInteger(
        value.color.rgb,
        "color.rgb",
      );

    if (
      rgb < 0 ||
      rgb > 0xffffff
    ) {
      throw new Error(
        "modelVariationCodec: color.rgbが0から16777215の範囲外です",
      );
    }

    return {
      name,
      rgb,
    };
  }

  /*
   * 旧レスポンス形式:
   * {
   *   color: "ホワイト",
   *   rgb: 16777215
   * }
   */
  if (typeof value.color === "string") {
    const name =
      requireNonEmptyString(
        value.color,
        "color",
      );

    const rgb =
      requireInteger(
        value.rgb,
        "rgb",
      );

    if (
      rgb < 0 ||
      rgb > 0xffffff
    ) {
      throw new Error(
        "modelVariationCodec: rgbが0から16777215の範囲外です",
      );
    }

    return {
      name,
      rgb,
    };
  }

  throw new Error(
    "modelVariationCodec: apparel colorがありません",
  );
}

function parseVolume(
  value: unknown,
): Volume {
  if (!isRecord(value)) {
    throw new Error(
      "modelVariationCodec: volumeがobjectではありません",
    );
  }

  const volumeValue =
    requireInteger(
      value.value,
      "volume.value",
    );

  if (volumeValue <= 0) {
    throw new Error(
      "modelVariationCodec: volume.valueが1未満です",
    );
  }

  const unit =
    requireNonEmptyString(
      value.unit,
      "volume.unit",
    );

  if (
    unit !== "ml" &&
    unit !== "L"
  ) {
    throw new Error(
      `modelVariationCodec: 未対応のvolume.unitです: ${unit}`,
    );
  }

  return {
    value: volumeValue,
    unit,
  };
}

/* =========================================================
 * Public parsers
 * =======================================================*/

export function parseModelVariationResponse(
  value: unknown,
  fallbackProductBlueprintId?: string,
): ModelVariationResponse {
  const unwrapped =
    unwrapSingleResponse(value);

  if (!isRecord(unwrapped)) {
    throw new Error(
      "modelVariationCodec: Model variationレスポンスがobjectではありません",
    );
  }

  const id =
    requireNonEmptyString(
      unwrapped.id,
      "id",
    );

  const productBlueprintId =
    typeof unwrapped.productBlueprintId === "string" &&
    unwrapped.productBlueprintId.length > 0
      ? unwrapped.productBlueprintId
      : requireNonEmptyString(
          fallbackProductBlueprintId,
          "productBlueprintId",
        );

  const kind =
    requireNonEmptyString(
      unwrapped.kind,
      "kind",
    );

  const modelNumber =
    requireNonEmptyString(
      unwrapped.modelNumber,
      "modelNumber",
    );

  const base = {
    id,
    productBlueprintId,
    modelNumber,

    createdAt:
      optionalString(
        unwrapped.createdAt,
        "createdAt",
      ),

    createdBy:
      optionalString(
        unwrapped.createdBy,
        "createdBy",
      ),

    updatedAt:
      optionalString(
        unwrapped.updatedAt,
        "updatedAt",
      ),

    updatedBy:
      optionalString(
        unwrapped.updatedBy,
        "updatedBy",
      ),
  };

  if (kind === "apparel") {
    const size =
      requireNonEmptyString(
        unwrapped.size,
        "size",
      );

    return {
      ...base,
      kind: "apparel",
      size,
      color:
        parseColor(unwrapped),
      measurements:
        parseMeasurements(
          unwrapped.measurements,
        ),
    };
  }

  if (kind === "alcohol") {
    return {
      ...base,
      kind: "alcohol",
      volume:
        parseVolume(
          unwrapped.volume,
        ),
    };
  }

  throw new Error(
    `modelVariationCodec: 未対応のkindです: ${kind}`,
  );
}

export function parseModelVariationListResponse(
  value: unknown,
  fallbackProductBlueprintId?: string,
): ModelVariationResponse[] {
  return unwrapListResponse(value).map(
    (variation) =>
      parseModelVariationResponse(
        variation,
        fallbackProductBlueprintId,
      ),
  );
}