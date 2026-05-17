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
    Object.prototype.hasOwnProperty.call(fields, "region") ||
    Object.prototype.hasOwnProperty.call(fields, "material")
  );
}

export function buildScanVolumeLabel(input: {
  volumeValue?: unknown;
  volumeUnit?: unknown;
}): string {
  const { volumeValue, volumeUnit } = input;

  const hasVolumeValue =
    typeof volumeValue === "number" && Number.isFinite(volumeValue);

  const unit = typeof volumeUnit === "string" ? volumeUnit.trim() : "";

  if (!hasVolumeValue || !unit) {
    return "";
  }

  return `${volumeValue}${unit}`;
}

export function createScanAlcoholInfo(input: {
  categoryFields: unknown;
  volumeValue?: unknown;
  volumeUnit?: unknown;
}): ScanAlcoholInfo | null {
  const fields = getScanCategoryFields(input.categoryFields);

  if (!isAlcoholCategoryFields(fields)) {
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
    }),
  };
}