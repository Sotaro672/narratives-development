//frontend\console\production\src\application\detail\loadProductionDetail.ts
import type { Production, ProductionStatus } from "../../../../shell/src/shared/types/production";

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../../infrastructure/query/productionQuery";

import type { ProductionDetail } from "./types";
import { asNonEmptyString, asString, toDate } from "./normalizers";

/* ---------------------------------------------------------
 * Production 詳細取得（usecase）
 * --------------------------------------------------------- */
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();
  const raw = (await repo.getById(productionId)) as any;
  if (!raw) return null;

  const rawModels = Array.isArray(raw.models)
    ? raw.models
    : Array.isArray(raw.Models)
      ? raw.Models
      : [];

  const totalQuantity = rawModels.reduce(
    (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
    0,
  );

  const blueprintId = asNonEmptyString(
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "",
  );

  const resolvedProductName =
    asNonEmptyString(raw.productName ?? raw.ProductName) || blueprintId;

  let detail: ProductionDetail = {
    ...(raw as Production),

    id: asNonEmptyString(raw.id ?? raw.ID ?? ""),
    productBlueprintId: blueprintId,

    productName: resolvedProductName,
    brandName: asString(
      raw.brandName ?? raw.BrandName ?? raw.brand ?? raw.Brand ?? "",
    ),

    assigneeId: asString(raw.assigneeId ?? raw.AssigneeID ?? ""),
    assigneeName: asString(raw.assigneeName ?? raw.AssigneeName ?? ""),

    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

    // ✅ time として保持（Date）
    printedAt: toDate(raw.printedAt ?? raw.PrintedAt ?? null),
    createdAt: toDate(raw.createdAt ?? raw.CreatedAt ?? null),
    updatedAt: toDate(raw.updatedAt ?? raw.UpdatedAt ?? null),

    models: rawModels,
    totalQuantity,
  };

  /* ---------------------------------------------------------
   * 一覧の name 解決ロジック（ユースケース内の合成）
   * --------------------------------------------------------- */
  try {
    const listItems = await listProductionsHTTP();

    const match = (listItems as any[]).find((item) => {
      const itemId = item.id ?? item.ID ?? "";
      const itemBlueprintId =
        item.productBlueprintId ?? item.ProductBlueprintID ?? "";
      return (
        itemId === detail.id ||
        (itemBlueprintId && itemBlueprintId === detail.productBlueprintId)
      );
    });

    if (match) {
      const matchProductName =
        match.productName ?? match.ProductName ?? detail.productBlueprintId;

      detail = {
        ...detail,
        productName:
          detail.productName && detail.productName !== detail.productBlueprintId
            ? detail.productName
            : matchProductName,

        brandName: detail.brandName || match.brandName || match.BrandName || "",

        assigneeName:
          detail.assigneeName ||
          match.assigneeName ||
          match.AssigneeName ||
          "",
      };
    }
  } catch (_) {
    // noop
  }

  return detail;
}
