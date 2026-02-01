// frontend/console/inventory/src/presentation/hook/listCreate/usePriceRows.ts
import * as React from "react";

import { usePriceCard } from "../../../../../list/src/presentation/hook/usePriceCard";
import type { PriceRow } from "../../../application/listCreate/priceCard.types";

export function usePriceRows(): {
  priceRows: PriceRow[];
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  onChangePrice: (index: number, price: number | null) => void;
  priceCard: ReturnType<typeof usePriceCard>;
} {
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);
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
    rows: priceRows,
    mode: "edit",
    currencySymbol: "¥",
    onChangePrice: (index, price, row) => onChangePrice(index, price),
  });

  return { priceRows, setPriceRows, initializedPriceRowsRef, onChangePrice, priceCard };
}
