//frontend\console\list\src\infrastructure\http\list\extractors.ts
export function extractItemsArrayFromAny(json: any): any[] {
  if (Array.isArray(json)) return json;
  if (json && typeof json === "object") {
    if (Array.isArray((json as any).items)) return (json as any).items;
    if (Array.isArray((json as any).Items)) return (json as any).Items;
    if (Array.isArray((json as any).data)) return (json as any).data;
  }
  return [];
}

export function extractFirstItemFromAny(json: any): any | null {
  if (!json) return null;
  if (Array.isArray(json)) return json[0] ?? null;

  if (json && typeof json === "object") {
    if ((json as any).id) return json;

    const items = extractItemsArrayFromAny(json);
    return items[0] ?? null;
  }

  return null;
}
