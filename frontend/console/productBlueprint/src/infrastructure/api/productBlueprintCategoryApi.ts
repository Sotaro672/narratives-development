//frontend\console\productBlueprint\src\infrastructure\api\productBlueprintCategoryApi.ts
import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";
import type { ProductBlueprintCategory } from "../../domain/entity/productBlueprintCategory";

export async function listProductBlueprintCategoriesApi(): Promise<ProductBlueprintCategory[]> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(`${API_BASE}/product-blueprint-categories`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `商品カテゴリ一覧の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  const json = (await res.json()) as ProductBlueprintCategory[];

  return [...json].sort((a, b) => {
    const ao = Number(a.displayOrder ?? 0);
    const bo = Number(b.displayOrder ?? 0);
    if (ao !== bo) return ao - bo;
    return String(a.code ?? "").localeCompare(String(b.code ?? ""));
  });
}