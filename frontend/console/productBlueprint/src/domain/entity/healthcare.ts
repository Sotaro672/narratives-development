// frontend/console/productBlueprint/src/domain/entity/healthcare.ts

export type HealthcareCategoryCode =
  | "healthcare.medical_device"
  | "healthcare.supplement"
  | "healthcare.wellness";

export type HealthcareCategoryOption = {
  value: HealthcareCategoryCode;
  label: string;
};

export const HEALTHCARE_CATEGORY_OPTIONS: HealthcareCategoryOption[] = [
  { value: "healthcare.medical_device", label: "医療・衛生用品" },
  { value: "healthcare.supplement", label: "サプリメント" },
  { value: "healthcare.wellness", label: "ウェルネス用品" },
];

export type HealthcareCategoryFieldKey = never;

export type HealthcareCategoryFields = Record<string, never>;

export const HEALTHCARE_CATEGORY_FIELD_KEYS: Record<
  HealthcareCategoryCode,
  HealthcareCategoryFieldKey[]
> = {
  "healthcare.medical_device": [],
  "healthcare.supplement": [],
  "healthcare.wellness": [],
};

export function isHealthcareCategoryCode(
  value: string,
): value is HealthcareCategoryCode {
  return (
    value === "healthcare.medical_device" ||
    value === "healthcare.supplement" ||
    value === "healthcare.wellness"
  );
}

export function getHealthcareCategoryFieldKeys(
  categoryCode: string,
): HealthcareCategoryFieldKey[] {
  if (!isHealthcareCategoryCode(categoryCode)) {
    return [];
  }

  return HEALTHCARE_CATEGORY_FIELD_KEYS[categoryCode] ?? [];
}