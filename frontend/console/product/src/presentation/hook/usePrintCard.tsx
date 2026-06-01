// frontend/console/product/src/presentation/hook/usePrintCard.tsx

import * as React from "react";
import {
  printOrCreateProductsForPrint,
  type PrintRow,
  type PrintLogForPrint,
} from "../../application/printService";

type QuantityRowBase = {
  modelId: string;
  quantity?: number | null;

  /**
   * QR 下ラベル用。
   * 印刷時点で画面側が保持している modelNumber を printService まで渡す。
   */
  modelNumber?: string;
};

type UsePrintCardParams<T extends QuantityRowBase> = {
  productionId: string | null;
  hasProduction: boolean;
  rows: T[];

  // 印刷完了後に呼ぶ
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
   *
   * 既存 print_log がある場合は GET のみで再印刷する。
   * 既存 print_log が無い場合だけ、初回作成として POST 系処理に進む。
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
        modelId: String(row.modelId ?? "").trim(),
        quantity: row.quantity ?? 0,
        modelNumber: String(row.modelNumber ?? "").trim(),
      }));

      const logs = await printOrCreateProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      const safeLogs = Array.isArray(logs) ? logs : [];

      setPrintLogs(safeLogs);

      try {
        onCompleted?.(safeLogs);
      } catch {
        // noop
      }

      return safeLogs;
    } catch {
      setError("印刷用のデータ作成に失敗しました");
      alert("印刷用のデータ作成に失敗しました");
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows, onCompleted]);

  return { onPrint, printLogs, printing, error };
}