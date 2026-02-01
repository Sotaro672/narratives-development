// frontend/console/inventory/src/presentation/hook/listCreate/usePriceRows.ts
import * as React from "react";

import { usePriceCard, type PriceRow } from "../../../../../list/src/presentation/hook/usePriceCard";
import type { PriceRowEx } from "../../../application/listCreate/listCreateService";

export function usePriceRows(): {
  priceRows: PriceRowEx[];
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRowEx[]>>;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  onChangePrice: (index: number, price: number | null) => void;
  priceCard: ReturnType<typeof usePriceCard>;
} {
  const [priceRows, setPriceRows] = React.useState<PriceRowEx[]>([]);
  const initializedPriceRowsRef = React.useRef(false);

  const onChangePrice = React.useCallback((index: number, price: number | null) => {
    setPriceRows((prev) => {
      const next = [...prev];
      if (!next[index]) return prev;
      next[index] = { ...next[index], price };
      return next;
    });
  }, []);

  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows as unknown as PriceRow[],
    mode: "edit",
    currencySymbol: "¥",
    onChangePrice: (index, price) => onChangePrice(index, price),
  });

  return { priceRows, setPriceRows, initializedPriceRowsRef, onChangePrice, priceCard };
}
