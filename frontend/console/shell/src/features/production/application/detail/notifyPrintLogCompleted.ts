// frontend/console/shell/src/features/production/application/detail/notifyPrintLogCompleted.ts

import {
  auth,
} from "../../../../auth/infrastructure/config/firebaseClient";

import {
  ProductionRepositoryHTTP,
} from "../../infrastructure/http/productionRepositoryHTTP";

/* ---------------------------------------------------------
 * 印刷完了シグナル受信（usecase）
 *   - Productionをprintedに更新（初回のみ）
 *   - 2回目以降（既存ログ再利用）は更新しない
 *   - ProductBlueprintのprinted更新はprintService側に委譲
 * --------------------------------------------------------- */
export async function notifyPrintLogCompleted(
  params: {
    productionId: string;
    logCount: number;
    totalQrCount: number;
    reusedExistingLogs?: boolean;
  },
): Promise<void> {
  const {
    productionId,
    reusedExistingLogs,
  } = params;

  const id =
    productionId.trim();

  if (!id) {
    return;
  }

  // 2回目以降（既存ログ再利用）の場合は
  // Productionを更新しない。
  if (reusedExistingLogs) {
    return;
  }

  const user =
    auth.currentUser;

  if (!user) {
    return;
  }

  const printedBy =
    user.uid;

  const printedAt =
    new Date().toISOString();

  const repository =
    new ProductionRepositoryHTTP();

  try {
    await repository.update(
      id,
      {
        printed: true,
        printedAt,
        printedBy,
      },
    );
  } catch {
    // 印刷ログ処理自体を失敗させないため、
    // Production更新エラーはここで吸収する。
  }
}