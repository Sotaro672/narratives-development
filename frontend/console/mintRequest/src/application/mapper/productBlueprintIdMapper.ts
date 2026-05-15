// frontend/console/mintRequest/src/application/mapper/productBlueprintIdMapper.ts

import { asNonEmptyString } from "./modelInspectionMapper";

export function extractProductBlueprintIdFromBatch(batch: unknown): string {
  if (!batch || typeof batch !== "object") return "";

  const b = batch as any;
  const value = b.productBlueprintId ?? b.productBlueprint?.id ?? "";

  return asNonEmptyString(value);
}