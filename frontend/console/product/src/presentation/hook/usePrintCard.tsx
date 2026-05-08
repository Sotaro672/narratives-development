// frontend/console/product/src/presentation/hook/usePrintCard.tsx

import * as React from "react";
import {
  createProductsForPrint,
  type PrintRow,
  type PrintLogForPrint,
} from "../../application/printService";

type QuantityRowBase = {
  modelId: string;
  quantity?: number | null;
};

type UsePrintCardParams<T extends QuantityRowBase> = {
  productionId: string | null;
  hasProduction: boolean;
  rows: T[];

  // ✅ 追加: 印刷完了（printLogs取得）後に呼ぶ
  onCompleted?: (logs: PrintLogForPrint[]) => void;
};

/**
 * 商品IDタグ用 Product 発行ロジックをまとめた Hook。
 */
export function usePrintCard<T extends QuantityRowBase>({
  productionId,
  hasProduction,
  rows,
  onCompleted,
}: UsePrintCardParams<T>) {
  const [printLogs, setPrintLogs] = React.useState<PrintLogForPrint[]>([]);
  const [printing, setPrinting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  /**
   * 印刷処理本体。
   */
  const onPrint = React.useCallback(async (): Promise<PrintLogForPrint[]> => {
    if (!productionId || !hasProduction) {
      return [];
    }

    try {
      setPrinting(true);
      setError(null);

      if (!Array.isArray(rows)) {
        alert("印刷用データが不正です（rows が配列ではありません）");
        return [];
      }

      const rowsForPrint: PrintRow[] = rows.map((row) => ({
        modelId: row.modelId,
        quantity: row.quantity ?? 0,
      }));

      const logs = await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      const safeLogs = Array.isArray(logs) ? logs : [];

      setPrintLogs(safeLogs);

      // ✅ 遷移や次処理は呼び出し側に委譲
      try {
        onCompleted?.(safeLogs);
      } catch {
        // noop（遷移失敗などで印刷自体を失敗扱いにしない）
      }

      return safeLogs;
    } catch (_) {
      setError("印刷用のデータ作成に失敗しました");
      alert("印刷用のデータ作成に失敗しました");
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows, onCompleted]);

  return { onPrint, printLogs, printing, error };
}