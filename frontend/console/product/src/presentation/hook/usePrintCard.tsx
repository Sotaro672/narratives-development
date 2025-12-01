// frontend/console/product/src/presentation/hook/usePrintCard.tsx
import * as React from "react";
import {
  createProductsForPrint,
  type PrintRow,
  type PrintLogForPrint,
} from "../../application/printService";

type QuantityRowBase = {
  modelVariationId: string;
  quantity?: number | null;
};

type UsePrintCardParams<T extends QuantityRowBase> = {
  productionId: string | null;
  hasProduction: boolean;
  rows: T[];
};

/**
 * 商品IDタグ用 Product 発行ロジックをまとめた Hook。
 */
export function usePrintCard<T extends QuantityRowBase>({
  productionId,
  hasProduction,
  rows,
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
        modelId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      const logs = await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      setPrintLogs(logs);
      return logs ?? [];
    } catch (_) {
      setError("印刷用のデータ作成に失敗しました");
      alert("印刷用のデータ作成に失敗しました");
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows]);

  return { onPrint, printLogs, printing, error };
}
