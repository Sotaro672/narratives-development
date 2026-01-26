// frontend/console/mintRequest/src/infrastructure/normalizers/production.ts

/**
 * productions -> productionIds（mint/inspections 用）
 */
export function normalizeProductionIdFromProductionListItem(v: any): string {
  return String(
    v?.productionId ??
      v?.ProductionId ??
      v?.id ??
      v?.ID ??
      v?.production?.id ??
      v?.production?.ID ??
      v?.production?.productionId ??
      "",
  ).trim();
}

export function normalizeProductBlueprintIdFromProductionListItem(v: any): string {
  return String(
    v?.productBlueprintId ??
      v?.productBlueprintID ??
      v?.ProductBlueprintId ??
      v?.ProductBlueprintID ??
      v?.production?.productBlueprintId ??
      v?.production?.productBlueprintID ??
      v?.production?.ProductBlueprintId ??
      v?.production?.ProductBlueprintID ??
      v?.productBlueprint?.id ??
      v?.productBlueprint?.ID ??
      "",
  ).trim();
}

export function normalizeProductionsPayload(json: any): any[] {
  if (Array.isArray(json)) return json;
  const items =
    json?.items ??
    json?.Items ??
    json?.productions ?? 
    json?.Productions ?? 
    null;
  return Array.isArray(items) ? items : [];
}
