// frontend/console/product/src/presentation/hook/usePrintCard.tsx

import * as React from "react";
import {
  printOrCreateProductsForPrint,
  type PrintLogForPrint,
} from "../../application/printService";

type UsePrintCardParams = {
  productionId: string | null;
  hasProduction: boolean;
  onCompleted?: (logs: PrintLogForPrint[]) => void;
};

/**
 * 商品IDタグ用 Product 発行ロジックをまとめた Hook。
 */
export function usePrintCard({
  productionId,
  hasProduction,
  onCompleted,
}: UsePrintCardParams) {
  const [printLogs, setPrintLogs] = React.useState<PrintLogForPrint[]>([]);
  const [printing, setPrinting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  /**
   * 印刷処理本体。
   *
   * 既存 print_log がある場合は GET のみで再印刷する。
   * 既存 print_log が無い場合だけ、初回作成として POST 処理に進む。
   */
  const onPrint = React.useCallback(async (): Promise<PrintLogForPrint[]> => {
    if (!productionId || !hasProduction) return [];

    try {
      setPrinting(true);
      setError(null);

      const logs = await printOrCreateProductsForPrint({ productionId });
      setPrintLogs(logs);
      onCompleted?.(logs);

      return logs;
    } catch {
      const message = "印刷用のデータ作成に失敗しました";
      setError(message);
      alert(message);
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, onCompleted]);

  return { onPrint, printLogs, printing, error };
}