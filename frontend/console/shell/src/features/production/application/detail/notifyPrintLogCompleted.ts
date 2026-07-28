// frontend/console/shell/src/features/production/application/detail/notifyPrintLogCompleted.ts

import {
  getCurrentUser,
  getIdTokenOrThrow,
} from "../../infrastructure/auth/firebaseAuth";
import { updateProduction } from "../../infrastructure/http/productionClient";

/* ---------------------------------------------------------
 * 印刷完了シグナル受信（usecase）
 *   - Productionをprintedに更新（初回のみ）
 *   - 2回目以降（既存ログ再利用）は更新しない
 *   - ProductBlueprintのprinted更新はprintService側に委譲
 * --------------------------------------------------------- */
export async function notifyPrintLogCompleted(params: {
  productionId: string;
  logCount: number;
  totalQrCount: number;
  reusedExistingLogs?: boolean;
}): Promise<void> {
  const { productionId, reusedExistingLogs } = params;

  const id = productionId.trim();
  if (!id) return;

  // 2回目以降（既存ログ再利用）の場合はProductionを更新しない
  if (reusedExistingLogs) return;

  const user = getCurrentUser();
  if (!user) return;

  const printedBy = user.uid;
  const printedAt = new Date().toISOString();

  try {
    const token = await getIdTokenOrThrow();

    const payload = {
      printed: true,
      printedAt,
      printedBy,
    };

    await updateProduction({
      productionId: id,
      token,
      payload,
      // notifyはエラーを握る仕様のため、updateProduction側で吸収する
      swallowError: true,
      logContext: {
        tag: "[notifyPrintLogCompleted]",
        productionId: id,
      },
    });
  } catch (error) {
    console.error(
      "[notifyPrintLogCompleted] unexpected error while updating production printed status",
      {
        productionId: id,
        error,
      },
    );
  }
}