// frontend/console/mintRequest/src/infrastructure/normalizers/production.ts

function asTrimmedString(v: any): string {
  return String(v ?? "").trim();
}

export function normalizeProductionIdFromProductionListItem(v: any): string {
  return asTrimmedString(
    v?.productionId ??
      v?.id ??
      v?.ID ??
      v?.ProductionId ??
      // nested production（念のため）
      v?.production?.productionId ??
      v?.production?.id ??
      v?.production?.ID ??
      "",
  );
}

export function normalizeProductBlueprintIdFromProductionListItem(v: any): string {
  return asTrimmedString(
    v?.productBlueprintId ??
      v?.productBlueprintID ??
      v?.ProductBlueprintId ??
      v?.ProductBlueprintID ??
      // nested production
      v?.production?.productBlueprintId ??
      v?.production?.productBlueprintID ??
      v?.production?.ProductBlueprintId ??
      v?.production?.ProductBlueprintID ??
      // nested productBlueprint
      v?.productBlueprint?.id ??
      v?.productBlueprint?.ID ??
      "",
  );
}

export function normalizeProductionsPayload(json: any): any[] {
  if (Array.isArray(json)) return json;

  const items = json?.items ?? json?.Items ?? json?.productions ?? json?.Productions ?? null;
  return Array.isArray(items) ? items : [];
}
