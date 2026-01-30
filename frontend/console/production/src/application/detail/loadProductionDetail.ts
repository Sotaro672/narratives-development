// frontend/console/production/src/application/detail/loadProductionDetail.ts

import type {
  Production,
  ProductionStatus,
} from "../../../../shell/src/shared/types/production";

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../../infrastructure/query/productionQuery";

import type { ProductionDetail, ProductionQuantityRow } from "./types";
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

  const rawModelsSrc = Array.isArray(raw.models)
    ? raw.models
    : Array.isArray(raw.Models)
      ? raw.Models
      : [];

  // ProductionDetail が要求する shape に正規化
  const rawModels: ProductionQuantityRow[] = (rawModelsSrc as any[]).map(
    (m: any, index: number) => {
      const modelId = asNonEmptyString(m?.modelId ?? m?.ModelID ?? m?.id ?? m?.ID ?? "") || String(index);

      const quantityRaw = m?.quantity ?? m?.Quantity ?? 0;
      const qNum = Number(quantityRaw);
      const quantity = Number.isFinite(qNum) ? Math.max(0, Math.floor(qNum)) : 0;

      const displayOrderRaw = m?.displayOrder ?? m?.DisplayOrder;
      const displayOrderNum = Number(displayOrderRaw);
      const displayOrder = Number.isFinite(displayOrderNum) ? displayOrderNum : undefined;

      return {
        modelId,
        modelNumber: asString(m?.modelNumber ?? m?.ModelNumber ?? ""),
        size: asString(m?.size ?? m?.Size ?? ""),
        color: asString(m?.color ?? m?.Color ?? ""),
        rgb: m?.rgb ?? m?.RGB ?? null,
        displayOrder,
        quantity,
      };
    },
  );

  const totalQuantity = rawModels.reduce((sum: number, m) => sum + (m.quantity ?? 0), 0);

  const blueprintId = asNonEmptyString(
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "",
  );

  // 必須: brandId / createdByName / updatedByName を必ず埋める
  const brandIdFromRaw = asString(
    raw.brandId ?? raw.BrandID ?? raw.BrandId ?? raw.brandID ?? "",
  );

  const createdByIdFromRaw = asString(
    raw.createdById ??
      raw.CreatedByID ??
      raw.createdBy ??
      raw.CreatedBy ??
      raw.createdByID ??
      "",
  );
  const updatedByIdFromRaw = asString(
    raw.updatedById ??
      raw.UpdatedByID ??
      raw.updatedBy ??
      raw.UpdatedBy ??
      raw.updatedByID ??
      "",
  );

  const createdByNameFromRaw = asString(raw.createdByName ?? raw.CreatedByName ?? "");
  const updatedByNameFromRaw = asString(raw.updatedByName ?? raw.UpdatedByName ?? "");

  let detail: ProductionDetail = {
    // 既存の raw を流用（ただし Date 型などは下で上書き）
    ...(raw as Production),

    id: asNonEmptyString(raw.id ?? raw.ID ?? ""),
    productBlueprintId: blueprintId,

    // ✅ 必須: Brand（NameResolver 済み想定だが、欠損は許容して埋める）
    brandId: brandIdFromRaw,
    brandName: asString(raw.brandName ?? raw.BrandName ?? raw.brand ?? raw.Brand ?? ""),

    // ✅ Assignee
    assigneeId: asString(raw.assigneeId ?? raw.AssigneeID ?? ""),
    assigneeName: asString(raw.assigneeName ?? raw.AssigneeName ?? ""),

    // ✅ Status
    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

    // ✅ timestamps（Date）
    printedAt: toDate(raw.printedAt ?? raw.PrintedAt ?? null),
    createdAt: toDate(raw.createdAt ?? raw.CreatedAt ?? null),
    updatedAt: toDate(raw.updatedAt ?? raw.UpdatedAt ?? null),

    // ✅ created/updated by（ID は optional、Name は required）
    createdById: createdByIdFromRaw ? createdByIdFromRaw : null,
    createdByName: (createdByNameFromRaw || createdByIdFromRaw || "-").trim(),

    updatedById: updatedByIdFromRaw ? updatedByIdFromRaw : null,
    updatedByName: (updatedByNameFromRaw || updatedByIdFromRaw || "-").trim(),

    // ✅ models
    models: rawModels,
    totalQuantity,
  };

  /* ---------------------------------------------------------
   * 一覧から name 解決ロジック（補完）
   * - productName は持たない
   * - brandId/brandName/assigneeName/createdByName/updatedByName を不足時に補完
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
      const matchBrandId = asString(match.brandId ?? match.BrandID ?? match.BrandId ?? "");
      const matchBrandName = asString(match.brandName ?? match.BrandName ?? "");
      const matchAssigneeName = asString(match.assigneeName ?? match.AssigneeName ?? "");

      const matchCreatedByName = asString(match.createdByName ?? match.CreatedByName ?? "");
      const matchUpdatedByName = asString(match.updatedByName ?? match.UpdatedByName ?? "");

      detail = {
        ...detail,

        brandId: detail.brandId || matchBrandId,
        brandName: detail.brandName || matchBrandName,

        assigneeName: detail.assigneeName || matchAssigneeName,

        createdByName:
          (detail.createdByName && detail.createdByName !== "-")
            ? detail.createdByName
            : (matchCreatedByName || detail.createdByName || "-"),

        updatedByName:
          (detail.updatedByName && detail.updatedByName !== "-")
            ? detail.updatedByName
            : (matchUpdatedByName || detail.updatedByName || "-"),
      };
    }
  } catch (_) {
    // noop
  }

  return detail;
}
