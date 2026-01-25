//frontend\console\production\src\application\detail\notifyPrintLogCompleted.ts
import type { ProductionStatus } from "../../../../shell/src/shared/types/production";

import { getCurrentUser, getIdTokenOrThrow } from "../../infrastructure/auth/firebaseAuth";
import { updateProduction } from "../../infrastructure/http/productionClient";

/* ---------------------------------------------------------
 * 印刷完了シグナル受信（usecase）
 *   - Production を printed に更新
 *   - ProductBlueprint の printed 更新は printService 側に委譲
 * --------------------------------------------------------- */
export async function notifyPrintLogCompleted(params: {
  productionId: string;
  logCount: number;
  totalQrCount: number;
  reusedExistingLogs?: boolean;
}): Promise<void> {
  const { productionId } = params;

  const id = productionId.trim();
  if (!id) return;

  const user = getCurrentUser();
  if (!user) return;

  const printedBy = user.uid;

  // ✅ payload は ISO 文字列で送る（API契約として妥当）
  const printedAt = new Date().toISOString();

  try {
    const token = await getIdTokenOrThrow();

    const payload: any = {
      status: "printed" as ProductionStatus,
      printedAt,
      printedBy,
    };

    await updateProduction({
      productionId: id,
      token,
      payload,
      // notifyはエラーを握る仕様のため、ここではthrowさせないために catch で吸収する
      swallowError: true,
      logContext: {
        tag: "[notifyPrintLogCompleted]",
        productionId: id,
      },
    });
  } catch (e) {
    console.error(
      "[notifyPrintLogCompleted] unexpected error while updating production printed status",
      { productionId: id, error: e },
    );
  }
}
