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
 * 商品IDタグ用 Product 発行 + print_log 取得ロジックをまとめた Hook。
 * Production 側から productionId / quantityRows などを渡して利用します。
 *
 * - createProductsForPrint:
 *   1. Product を必要数作成
 *   2. print_log を作成
 *   3. print_log（QR ペイロード付き）一覧を返す
 *
 * → Hook では返却された print_log 一覧を保持し、呼び出し元へ渡します。
 */
export function usePrintCard<T extends QuantityRowBase>({
  productionId,
  hasProduction,
  rows,
}: UsePrintCardParams<T>) {
  const [printLogs, setPrintLogs] = React.useState<PrintLogForPrint[]>([]);
  const [printing, setPrinting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const onPrint = React.useCallback(async () => {
    if (!productionId || !hasProduction) return;

    try {
      setPrinting(true);
      setError(null);

      // PrintRow[] へマッピング（printService.tsx の型に合わせる）
      // modelVariationId → modelId として渡す
      const rowsForPrint: PrintRow[] = rows.map((row) => ({
        modelId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      const logs = await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      setPrintLogs(logs);
    } catch (e) {
      console.error(e);
      setError("印刷用のデータ作成または print_log の取得に失敗しました");
      alert("印刷用のデータ作成または print_log の取得に失敗しました");
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows]);

  return {
    onPrint,
    printLogs,
    printing,
    error,
  };
}
