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
 * Production 側から productionId / quantityRows などを渡して利用します。
 *
 * - Product 作成
 * - print_log 作成
 * - print_log 一覧（QR ペイロード付き）取得
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
   *
   * 戻り値として print_log 一覧（PrintLogForPrint[]）を返すので、
   * 呼び出し元（productionDetail.tsx）から PDF 印刷処理などを呼び出せる。
   */
  const onPrint = React.useCallback(async (): Promise<PrintLogForPrint[]> => {
    if (!productionId || !hasProduction) return [];

    try {
      setPrinting(true);
      setError(null);

      // PrintRow[] へマッピング（printService.tsx の型に合わせる）
      const rowsForPrint: PrintRow[] = rows.map((row) => ({
        modelId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      // Product 作成 + print_log 作成 + print_log 一覧取得
      const logs = await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      setPrintLogs(logs);
      return logs;
    } catch (e) {
      console.error(e);
      setError("印刷用のデータ作成に失敗しました");
      alert("印刷用のデータ作成に失敗しました");
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows]);

  return { onPrint, printLogs, printing, error };
}
