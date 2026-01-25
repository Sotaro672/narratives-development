//frontend\console\production\src\application\detail\updateProductionDetail.ts
import type { ProductionDetail, ProductionQuantityRow } from "./types";
import { getIdTokenOrThrow } from "../../infrastructure/auth/firebaseAuth";
import { updateProduction } from "../../infrastructure/http/productionClient";
import { loadProductionDetail } from "./loadProductionDetail";

/* ---------------------------------------------------------
 * Production 更新リクエスト（usecase）
 * --------------------------------------------------------- */
export async function updateProductionDetail(params: {
  productionId: string;
  rows: ProductionQuantityRow[];
  assigneeId?: string | null;
}): Promise<ProductionDetail | null> {
  const { productionId, rows, assigneeId } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  const token = await getIdTokenOrThrow();

  const modelsPayload = rows.map((r) => ({
    modelId: r.id,
    quantity: Number.isFinite(Number(r.quantity))
      ? Math.max(0, Math.floor(Number(r.quantity)))
      : 0,
  }));

  const payload: any = {
    assigneeId: assigneeId ?? null,
    models: modelsPayload,
  };

  await updateProduction({
    productionId: id,
    token,
    payload,
  });

  return loadProductionDetail(id);
}
