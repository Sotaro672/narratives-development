// frontend/amol/src/features/scan-result/application/scanAlcoholInfoFactory.ts

export type ScanCategoryFields = Record<string, unknown>;

export type ScanAlcoholInfo = {
  isAlcohol: boolean;
  vintage: string;
  region: string;
  material: string;
  alcoholContent: string;
  volumeLabel: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function getStringValue(fields: ScanCategoryFields, key: string): string {
  const value = fields[key];

  if (typeof value === "string") {
    return value.trim();
  }

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function getNumberValue(fields: ScanCategoryFields, key: string): string {
  const value = fields[key];

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  if (typeof value === "string" && value.trim()) {
    const parsed = Number(value);

    return Number.isFinite(parsed) ? String(parsed) : "";
  }

  return "";
}

function getStringFromRecord(value: unknown, key: string): string {
  if (!isRecord(value)) {
    return "";
  }

  const raw = value[key];

  if (typeof raw === "string") {
    return raw.trim();
  }

  return "";
}

function isAlcoholKind(value: unknown): boolean {
  return typeof value === "string" && value.trim().toLowerCase() === "alcohol";
}

function isAlcoholCode(value: unknown): boolean {
  if (typeof value !== "string") {
    return false;
  }

  const code = value.trim().toLowerCase();

  return code === "alcohol" || code.startsWith("alcohol.");
}

export function getScanCategoryFields(value: unknown): ScanCategoryFields {
  if (!isRecord(value)) {
    return {};
  }

  return value;
}

export function isAlcoholCategoryFields(fields: ScanCategoryFields): boolean {
  return (
    Object.prototype.hasOwnProperty.call(fields, "alcoholContent") ||
    Object.prototype.hasOwnProperty.call(fields, "vintage") ||
    Object.prototype.hasOwnProperty.call(fields, "region")
  );
}

function resolveIsAlcohol(input: {
  categoryFields: ScanCategoryFields;
  modelKind?: unknown;
  productBlueprintCategoryKind?: unknown;
  productBlueprintCategory?: unknown;
  categoryInputSchema?: unknown;
}): boolean {
  if (isAlcoholKind(input.modelKind)) {
    return true;
  }

  if (isAlcoholKind(input.productBlueprintCategoryKind)) {
    return true;
  }

  if (isAlcoholKind(getStringFromRecord(input.productBlueprintCategory, "Kind"))) {
    return true;
  }

  if (isAlcoholKind(getStringFromRecord(input.productBlueprintCategory, "kind"))) {
    return true;
  }

  if (isAlcoholCode(getStringFromRecord(input.productBlueprintCategory, "Code"))) {
    return true;
  }

  if (isAlcoholCode(getStringFromRecord(input.productBlueprintCategory, "code"))) {
    return true;
  }

  if (isAlcoholKind(getStringFromRecord(input.categoryInputSchema, "categoryKind"))) {
    return true;
  }

  if (isAlcoholCode(getStringFromRecord(input.categoryInputSchema, "categoryCode"))) {
    return true;
  }

  return isAlcoholCategoryFields(input.categoryFields);
}

export function buildScanVolumeLabel(input: {
  volumeValue?: unknown;
  volumeUnit?: unknown;
  modelLabel?: unknown;
}): string {
  const { volumeValue, volumeUnit, modelLabel } = input;

  const unit = typeof volumeUnit === "string" ? volumeUnit.trim() : "";

  if (typeof volumeValue === "number" && Number.isFinite(volumeValue)) {
    return unit ? `${volumeValue}${unit}` : String(volumeValue);
  }

  if (typeof volumeValue === "string" && volumeValue.trim()) {
    const normalizedVolumeValue = volumeValue.trim();

    return unit ? `${normalizedVolumeValue}${unit}` : normalizedVolumeValue;
  }

  if (typeof modelLabel === "string" && modelLabel.trim()) {
    return modelLabel.trim();
  }

  return "";
}

export function createScanAlcoholInfo(input: {
  categoryFields: unknown;
  volumeValue?: unknown;
  volumeUnit?: unknown;
  modelLabel?: unknown;
  modelKind?: unknown;
  productBlueprintCategoryKind?: unknown;
  productBlueprintCategory?: unknown;
  categoryInputSchema?: unknown;
}): ScanAlcoholInfo | null {
  const fields = getScanCategoryFields(input.categoryFields);

  const isAlcohol = resolveIsAlcohol({
    categoryFields: fields,
    modelKind: input.modelKind,
    productBlueprintCategoryKind: input.productBlueprintCategoryKind,
    productBlueprintCategory: input.productBlueprintCategory,
    categoryInputSchema: input.categoryInputSchema,
  });

  if (!isAlcohol) {
    return null;
  }

  return {
    isAlcohol: true,
    vintage: getNumberValue(fields, "vintage"),
    region: getStringValue(fields, "region"),
    material: getStringValue(fields, "material"),
    alcoholContent: getNumberValue(fields, "alcoholContent"),
    volumeLabel: buildScanVolumeLabel({
      volumeValue: input.volumeValue,
      volumeUnit: input.volumeUnit,
      modelLabel: input.modelLabel,
    }),
  };
}