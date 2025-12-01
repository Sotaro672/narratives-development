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
   */
  const onPrint = React.useCallback(async (): Promise<PrintLogForPrint[]> => {
    if (!productionId || !hasProduction) {
      console.warn("[usePrintCard] print skipped: no production or invalid state", {
        productionId,
        hasProduction,
      });
      return [];
    }

    try {
      setPrinting(true);
      setError(null);

      // -------------------------
      // ★ 渡された rows をログ表示
      // -------------------------
      console.log("[usePrintCard] onPrint invoked with params:", {
        productionId,
        hasProduction,
        rows,
      });

      // rows が null/undefined の可能性対策
      if (!Array.isArray(rows)) {
        console.error("[usePrintCard] rows is NOT array", rows);
        alert("印刷用データが不正です（rows が配列ではありません）");
        return [];
      }

      // -------------------------
      // ★ PrintRow[] へマッピング
      // -------------------------
      const rowsForPrint: PrintRow[] = rows.map((row) => ({
        modelId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      console.log("[usePrintCard] rowsForPrint (payload for printService):", rowsForPrint);

      // -------------------------
      // ★ Product 作成 + print_log 作成
      // -------------------------
      const logs = await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });

      console.log("[usePrintCard] response logs from printService:", logs);

      setPrintLogs(logs);
      return logs ?? [];
    } catch (e) {
      console.error("[usePrintCard] print error:", e);
      setError("印刷用のデータ作成に失敗しました");
      alert("印刷用のデータ作成に失敗しました");
      return [];
    } finally {
      setPrinting(false);
    }
  }, [productionId, hasProduction, rows]);

  return { onPrint, printLogs, printing, error };
}
