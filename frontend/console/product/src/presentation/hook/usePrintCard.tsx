// frontend/console/product/src/presentation/hook/usePrintCard.tsx
import * as React from "react";
import {
  createProductsForPrint,
  type PrintRow,
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
 */
export function usePrintCard<T extends QuantityRowBase>({
  productionId,
  hasProduction,
  rows,
}: UsePrintCardParams<T>) {
  const onPrint = React.useCallback(async () => {
    if (!productionId || !hasProduction) return;

    try {
      // PrintRow[] へマッピング（printService.tsx の型に合わせる）
      const rowsForPrint: PrintRow[] = rows.map((row) => ({
        modelVariationId: row.modelVariationId,
        quantity: row.quantity ?? 0,
      }));

      await createProductsForPrint({
        productionId,
        rows: rowsForPrint,
      });
    } catch {
      alert("印刷用のデータ作成に失敗しました");
    }
  }, [productionId, hasProduction, rows]);

  return { onPrint };
}
