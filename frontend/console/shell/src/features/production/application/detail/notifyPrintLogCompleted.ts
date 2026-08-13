// frontend/console/shell/src/features/production/application/detail/notifyPrintLogCompleted.ts

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

/**
 * 印刷完了後、初回印刷時のみ Production の printed 情報を更新する。
 * 既存 print_log を利用した再印刷では更新しない。
 */
export async function notifyPrintLogCompleted(params: {
  productionId: string;
  reusedExistingLogs?: boolean;
}): Promise<void> {
  const { productionId, reusedExistingLogs } = params;
  const id = productionId.trim();

  if (!id || reusedExistingLogs) {
    return;
  }

  const user = auth.currentUser;

  if (!user) {
    return;
  }

  const repository = new ProductionRepositoryHTTP();

  try {
    await repository.update(id, {
      printed: true,
      printedAt: new Date().toISOString(),
      printedBy: user.uid,
    });
  } catch {
    // 印刷処理自体は成功しているため、Production更新エラーはここでは伝播させない。
  }
}