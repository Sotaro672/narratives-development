// frontend/console/production/src/application/detail/loadProductionDetail.ts

import type { Production } from "../../../../shell/src/shared/types/production";

import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../../infrastructure/query/productionQuery";

import type { ProductionDetail, ProductionQuantityRow } from "./types";

/* ---------------------------------------------------------
 * Production 詳細取得（usecase）
 * - 正: backend が返す DTO のキー（ID, ProductBlueprintID, Models[].ModelID ...）
 * - 名揺れ吸収/正規化は行わない
 * - 一覧補完（/productions）を復活させ、brandName / assigneeName を補完する
 * --------------------------------------------------------- */
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  const pid = String(productionId ?? "").trim();
  if (!pid) return null;

  const repo = new ProductionRepositoryHTTP();
  const raw = (await repo.getById(pid)) as any;
  if (!raw) return null;

  // ✅ backend response (single source of truth)
  const id: string = String(raw.ID);
  const productBlueprintId: string = String(raw.ProductBlueprintID);

  // detail 側で欠けることがあるため、まずは raw の値を入れる
  let brandName: string = String(raw.brandName ?? "");
  let assigneeName: string = String(raw.assigneeName ?? "");

  const assigneeId: string = String(raw.AssigneeID);
  const printed: boolean = Boolean(raw.Printed);

  const printedAt: Date | null = raw.PrintedAt
    ? new Date(String(raw.PrintedAt))
    : null;
  const createdAt: Date | null = raw.CreatedAt
    ? new Date(String(raw.CreatedAt))
    : null;
  const updatedAt: Date | null = raw.UpdatedAt
    ? new Date(String(raw.UpdatedAt))
    : null;

  // models: [{ ModelID, Quantity }]
  const rawModels = Array.isArray(raw.Models) ? raw.Models : [];

  const models: ProductionQuantityRow[] = rawModels.map((m: any) => {
    const modelId = String(m.ModelID);
    const qNum = Number(m.Quantity);
    const quantity = Number.isFinite(qNum) ? Math.max(0, Math.floor(qNum)) : 0;

    return {
      modelId,
      quantity,

      // 表示メタは detail 側で補完（modelIndex と join する前提）
      modelNumber: "",
      size: "",
      color: "",
      rgb: null,
      displayOrder: undefined,
    };
  });

  const totalQuantity: number =
    typeof raw.totalQuantity === "number" ? raw.totalQuantity : 0;

  // ---------------------------------------------------------
  // ✅ 一覧から name 解決ロジック（補完）復活
  // - /productions は brandName / assigneeName を確実に含む
  // - detail 側が id のまま等で崩れるケースをここで上書きする
  // ---------------------------------------------------------
  try {
    const list = await listProductionsHTTP();
    const items = Array.isArray(list) ? list : [];

    const match = items.find((it: any) => String(it.ID) === id);
    if (match) {
      const bn = String(match.brandName ?? "");
      const an = String(match.assigneeName ?? "");
      if (bn) brandName = bn;
      if (an) assigneeName = an;
    }
  } catch {
    // noop（補完失敗でも detail は返す）
  }

  const detail: ProductionDetail = {
    ...(raw as Production),

    id,
    productBlueprintId,

    // brandId は DTO に無い前提なので空（必要なら backend に追加）
    brandId: "",
    brandName,

    assigneeId,
    assigneeName,

    printed,

    models,
    totalQuantity,

    printedAt,

    createdById: raw.CreatedBy ? String(raw.CreatedBy) : null,
    createdByName: raw.CreatedBy ? String(raw.CreatedBy) : "-",
    createdAt,

    updatedById: raw.UpdatedBy ? String(raw.UpdatedBy) : null,
    updatedByName: raw.UpdatedBy ? String(raw.UpdatedBy) : "-",
    updatedAt,
  };

  return detail;
}