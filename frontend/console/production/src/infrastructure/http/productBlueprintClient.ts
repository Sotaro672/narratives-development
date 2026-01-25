//frontend\console\production\src\infrastructure\http\productBlueprintClient.ts
import { BACKEND_API_BASE } from "./apiBase";

export async function fetchProductBlueprintById(params: {
  productBlueprintId: string;
  token: string;
}): Promise<any> {
  const { productBlueprintId, token } = params;

  const safeId = encodeURIComponent(productBlueprintId);
  const url = `${BACKEND_API_BASE}/product-blueprints/${safeId}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `ProductBlueprint API error: ${res.status} ${res.statusText}${body ? ` - ${body}` : ""}`,
    );
  }

  return res.json();
}
