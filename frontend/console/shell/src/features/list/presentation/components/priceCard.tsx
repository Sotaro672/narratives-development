// frontend/console/shell/src/features/list/presentation/components/priceCard.tsx

import * as React from "react";
import { Tag } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import { Input } from "../../../../shared/ui/input";

import { usePriceCard } from "../hook/usePriceCard";

import type {
  PriceCardProps,
  PriceRowVM,
} from "../../../inventory/application/listCreate/listCreateService";

type ProductBlueprintCategoryKind =
  | "apparel"
  | "alcohol"
  | "unknown";

function resolveProductBlueprintCategoryKind(
  args: {
    productBlueprintCategory?: string;
    rows: PriceRowVM[];
  },
): ProductBlueprintCategoryKind {
  const category =
    String(
      args.productBlueprintCategory ?? "",
    )
      .trim()
      .toLowerCase();

  if (
    category.startsWith(
      "alcohol",
    )
  ) {
    return "alcohol";
  }

  if (
    category.startsWith(
      "apparel",
    )
  ) {
    return "apparel";
  }

  const hasAlcoholRow =
    args.rows.some(
      (row) =>
        row.kind ===
        "alcohol",
    );

  if (hasAlcoholRow) {
    return "alcohol";
  }

  const hasApparelRow =
    args.rows.some(
      (row) =>
        row.kind ===
        "apparel",
    );

  if (hasApparelRow) {
    return "apparel";
  }

  return "unknown";
}

function getVolumeValueLabel(
  row: PriceRowVM,
): string {
  const value =
    row.volumeValue;

  if (
    typeof value ===
      "number" &&
    Number.isFinite(value)
  ) {
    return String(value);
  }

  return "";
}

function getVolumeUnitLabel(
  row: PriceRowVM,
): string {
  return String(
    row.volumeUnit ?? "",
  ).trim();
}

const PriceCard:
  React.FC<PriceCardProps> = (
    props,
  ) => {
    const {
      className,
      productBlueprintCategory,
    } = props;

    const {
      title,
      mode,
      isEdit,
      showModeBadge,
      currencySymbol,
      rowsVM,
      isEmpty,
    } =
      usePriceCard(
        props,
      );

    const categoryKind =
      React.useMemo(
        () =>
          resolveProductBlueprintCategoryKind({
            productBlueprintCategory,
            rows: rowsVM,
          }),
        [
          productBlueprintCategory,
          rowsVM,
        ],
      );

    const isAlcoholCategory =
      categoryKind ===
      "alcohol";

    return (
      <Card
        className={`prc ${className ?? ""}`}
      >
        <CardHeader className="prc__header">
          <div className="prc__header-inner flex items-center gap-2">
            <Tag size={18} />

            <CardTitle className="prc__title">
              {title}

              {showModeBadge && (
                <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">
                  （{mode}）
                </span>
              )}
            </CardTitle>
          </div>
        </CardHeader>

        <CardContent className="prc__body">
          <div className="prc__table-wrap">
            <Table className="prc__table">
              <TableHeader>
                <TableRow>
                  {isAlcoholCategory ? (
                    <>
                      <TableHead className="prc__th">
                        容量
                      </TableHead>

                      <TableHead className="prc__th">
                        単位
                      </TableHead>
                    </>
                  ) : (
                    <>
                      <TableHead className="prc__th">
                        サイズ
                      </TableHead>

                      <TableHead className="prc__th">
                        カラー
                      </TableHead>
                    </>
                  )}

                  <TableHead className="prc__th prc__th--right">
                    在庫数
                  </TableHead>

                  <TableHead className="prc__th prc__th--right">
                    価格
                  </TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {rowsVM.map(
                  (row) => {
                    return (
                      <TableRow
                        key={
                          row.modelId
                        }
                        className="prc__tr"
                      >
                        {isAlcoholCategory ? (
                          <>
                            <TableCell className="prc__size">
                              {getVolumeValueLabel(
                                row,
                              ) ||
                                "-"}
                            </TableCell>

                            <TableCell className="prc__size">
                              {getVolumeUnitLabel(
                                row,
                              ) ||
                                "-"}
                            </TableCell>
                          </>
                        ) : (
                          <>
                            <TableCell className="prc__size">
                              {row.size ||
                                "-"}
                            </TableCell>

                            <TableCell className="prc__color-cell">
                              <span
                                className="prc__color-dot"
                                style={{
                                  backgroundColor:
                                    row.bgColor,
                                }}
                                title={
                                  row.rgbTitle
                                }
                              />

                              <span className="prc__color-label">
                                {row.color ||
                                  "-"}
                              </span>
                            </TableCell>
                          </>
                        )}

                        <TableCell className="prc__stock text-right">
                          <span className="prc__stock-number">
                            {
                              row.stock
                            }
                          </span>
                        </TableCell>

                        <TableCell className="prc__price text-right">
                          {isEdit ? (
                            <div className="flex items-center justify-end gap-2">
                              {currencySymbol ? (
                                <span className="text-xs text-[hsl(var(--muted-foreground))]">
                                  {
                                    currencySymbol
                                  }
                                </span>
                              ) : null}

                              <Input
                                required
                                inputMode="numeric"
                                type="number"
                                min={0}
                                step={1}
                                className="h-8 w-32 text-right"
                                value={
                                  row.priceInputValue
                                }
                                placeholder="-"
                                onChange={
                                  row.onChangePriceInput
                                }
                              />
                            </div>
                          ) : (
                            <span className="prc__price-value">
                              {
                                row.priceDisplayText
                              }
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    );
                  },
                )}

                {isEmpty && (
                  <TableRow>
                    <TableCell
                      colSpan={4}
                      className="prc__empty"
                    >
                      表示できるデータがありません。
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    );
  };

export default PriceCard;