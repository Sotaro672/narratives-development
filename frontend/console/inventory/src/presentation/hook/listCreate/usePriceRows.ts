// frontend/console/inventory/src/presentation/hook/listCreate/usePriceRows.ts

import * as React from "react";

import { usePriceCard } from "../../../../../list/presentation/hook/usePriceCard";
import type { PriceRow } from "../../../application/listCreate/listCreate.types";

type UsePriceRowsResult = {
  priceRows: PriceRow[];
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  productBlueprintCategory: string | undefined;
  setProductBlueprintCategory: React.Dispatch<
    React.SetStateAction<string | undefined>
  >;
  onChangePrice: (index: number, price: number | null) => void;
  priceCard: ReturnType<typeof usePriceCard>;
};

export function usePriceRows(): UsePriceRowsResult {
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);
  const [productBlueprintCategory, setProductBlueprintCategory] =
    React.useState<string | undefined>(undefined);

  const initializedPriceRowsRef = React.useRef(false);

  const onChangePrice = React.useCallback(
    (index: number, price: number | null) => {
      setPriceRows((prev) => {
        const next = [...prev];
        if (!next[index]) return prev;

        next[index] = {
          ...next[index],
          price,
        };

        return next;
      });
    },
    [],
  );

  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows,
    mode: "edit",
    currencySymbol: "¥",
    productBlueprintCategory,
    onChangePrice,
  });

  return {
    priceRows,
    setPriceRows,
    initializedPriceRowsRef,
    productBlueprintCategory,
    setProductBlueprintCategory,
    onChangePrice,
    priceCard,
  };
}