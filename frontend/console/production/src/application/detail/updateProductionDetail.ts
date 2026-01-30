// frontend/console/production/src/application/detail/updateProductionDetail.ts

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

  // ✅ modelId を正として送る（ProductionQuantityRow のキーは modelId）
  const modelsPayload = (Array.isArray(rows) ? rows : []).map((r) => ({
    modelId: String((r as any).modelId ?? "").trim(),
    quantity: Number.isFinite(Number((r as any).quantity))
      ? Math.max(0, Math.floor(Number((r as any).quantity)))
      : 0,
  }));

  // guard: modelId 欠損行を落とす（空文字は送らない）
  const safeModelsPayload = modelsPayload.filter((m) => m.modelId !== "");

  const payload: any = {
    assigneeId: assigneeId ?? null,
    models: safeModelsPayload,
  };

  await updateProduction({
    productionId: id,
    token,
    payload,
  });

  return loadProductionDetail(id);
}
