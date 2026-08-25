// frontend/console/shell/src/features/list/presentation/hook/usePriceCard.tsx

import * as React from "react";

import { rgbIntToHex } from "../../../../shared/util/color";

import type {
  PriceCardProps,
  PriceRow,
  PriceRowVM,
  UsePriceCardResult,
} from "../../../inventory/application/listCreateService";

// ----------------------------------------------------------
// Types
// ----------------------------------------------------------

type IndexedPriceRow = {
  row: PriceRow;
  originalIdx: number;
};

// ----------------------------------------------------------
// Helpers
// ----------------------------------------------------------

function parsePriceInput(
  value: string,
): number | undefined {
  if (value === "") {
    return undefined;
  }

  const parsed = Number(value);

  if (!Number.isFinite(parsed)) {
    return undefined;
  }

  return Math.max(
    0,
    Math.floor(parsed),
  );
}

function getBgColor(
  rgb: PriceRow["rgb"],
): string {
  const rgbHex =
    rgbIntToHex(rgb) ?? null;

  if (
    typeof rgb === "string" &&
    rgb.startsWith("#")
  ) {
    return rgb;
  }

  return rgbHex ?? "#ffffff";
}

function getRgbTitle(
  rgb: PriceRow["rgb"],
): string {
  const rgbHex =
    rgbIntToHex(rgb) ?? null;

  if (rgbHex) {
    return rgbHex;
  }

  if (typeof rgb === "string") {
    return rgb;
  }

  return "";
}

function getPriceInputValue(
  price: number | undefined,
): string {
  if (price === undefined) {
    return "";
  }

  return String(price);
}

function getPriceDisplayText(
  args: {
    price: number | undefined;
    currencySymbol: string;
  },
): string {
  if (args.price === undefined) {
    return "-";
  }

  return `${args.currencySymbol}${args.price}`;
}

// ----------------------------------------------------------
// Hook
// ----------------------------------------------------------

export function usePriceCard(
  props: PriceCardProps,
): UsePriceCardResult {
  const {
    title = "価格設定",
    rows,
    mode = "view",
    onChangePrice,
    currencySymbol = "¥",
  } = props;

  const isEdit =
    mode === "edit";

  const showModeBadge =
    mode !== "view";

  const rowsVM =
    React.useMemo<
      PriceRowVM[]
    >(() => {
      const sortedRows:
        IndexedPriceRow[] =
        rows
          .map(
            (
              row,
              originalIdx,
            ) => ({
              row,
              originalIdx,
            }),
          )
          .sort(
            (
              first,
              second,
            ) => {
              const firstOrder =
                first.row
                  .displayOrder ??
                Number.POSITIVE_INFINITY;

              const secondOrder =
                second.row
                  .displayOrder ??
                Number.POSITIVE_INFINITY;

              if (
                firstOrder !==
                secondOrder
              ) {
                return (
                  firstOrder -
                  secondOrder
                );
              }

              return (
                first.originalIdx -
                second.originalIdx
              );
            },
          );

      return sortedRows.map(
        ({
          row,
          originalIdx,
        }) => {
          const priceInputValue =
            getPriceInputValue(
              row.price,
            );

          const priceDisplayText =
            getPriceDisplayText({
              price:
                row.price,

              currencySymbol,
            });

          const onChangePriceInput = (
            event:
              React.ChangeEvent<HTMLInputElement>,
          ) => {
            const nextPrice =
              parsePriceInput(
                event.target.value,
              );

            onChangePrice?.(
              originalIdx,
              nextPrice,
              row,
            );
          };

          return {
            modelId:row.modelId,
            kind:row.kind ?? null,
            displayOrder:row.displayOrder ?? null,
            size:row.size ?? null,
            color:row.color ?? null,
            volumeValue:row.volumeValue ?? null,
            volumeUnit:row.volumeUnit ?? null,
            stock:row.stock,
            bgColor:getBgColor(row.rgb,),
            rgbTitle:getRgbTitle(row.rgb,),
            priceInputValue,
            priceDisplayText,
            onChangePriceInput,
          };
        },
      );
    }, [
      rows,
      onChangePrice,
      currencySymbol,
    ]);

  return {
    title,
    mode,
    isEdit,
    showModeBadge,
    currencySymbol,
    rowsVM,
    isEmpty:
      rows.length === 0,
  };
}