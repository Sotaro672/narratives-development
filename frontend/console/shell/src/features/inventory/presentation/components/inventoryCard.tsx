// frontend/console/shell/src/features/inventory/presentation/components/inventoryCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui/card";

import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../../shared/ui/table";

import type {
  InventoryRow,
} from "../../../../shared/types/inventory";

import {
  rgbIntToHex,
} from "../../../../shared/util/color";

type ProductBlueprintCategoryKind =
  | "apparel"
  | "alcohol"
  | "unknown";

type InventoryCardProps = {
  title?: string;

  rows:
    InventoryRow[];

  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?:
    string;

  className?:
    string;

  mode?:
    "view";
};

function resolveProductBlueprintCategoryKind(
  args: {
    productBlueprintCategory?:
      string;

    rows:
      InventoryRow[];
  },
): ProductBlueprintCategoryKind {
  const category =
    String(
      args.productBlueprintCategory ??
        "",
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
  row:
    InventoryRow,
): string {
  const value =
    row.volumeValue;

  if (
    typeof value ===
      "number" &&
    Number.isFinite(
      value,
    )
  ) {
    return String(
      value,
    );
  }

  return "";
}

function getVolumeUnitLabel(
  row:
    InventoryRow,
): string {
  return String(
    row.volumeUnit ??
      "",
  ).trim();
}

const InventoryCard:
  React.FC<InventoryCardProps> = ({
    title =
      "モデル別在庫一覧",

    rows,

    productBlueprintCategory,

    className,

    mode =
      "view",
  }) => {
    const categoryKind =
      React.useMemo(
        () =>
          resolveProductBlueprintCategoryKind({
            productBlueprintCategory,
            rows,
          }),
        [
          productBlueprintCategory,
          rows,
        ],
      );

    const isAlcoholCategory =
      categoryKind ===
      "alcohol";

    const totalStock =
      React.useMemo(
        () =>
          rows.reduce(
            (
              sum,
              row,
            ) =>
              sum +
              row.stock,
            0,
          ),
        [
          rows,
        ],
      );

    const footerColSpan =
      3;

    return (
      <Card
        className={`ivc ${className ?? ""}`}
      >
        <CardHeader
          className="ivc__header"
        >
          <div
            className="ivc__header-inner"
          >
            <Palette
              className="ivc__icon"
              size={18}
            />

            <CardTitle
              className="ivc__title"
            >
              {title}

              {mode !== "view" && (
                <span
                  className="ml-2 text-xs text-[hsl(var(--muted-foreground))]"
                >
                  （{mode}）
                </span>
              )}
            </CardTitle>
          </div>
        </CardHeader>

        <CardContent
          className="ivc__body"
        >
          <div
            className="ivc__table-wrap"
          >
            <Table
              className="ivc__table"
            >
              <TableHeader>
                <TableRow>
                  <TableHead
                    className="ivc__th ivc__th--left"
                  >
                    型番
                  </TableHead>

                  {isAlcoholCategory
                    ? (
                      <>
                        <TableHead
                          className="ivc__th"
                        >
                          容量
                        </TableHead>

                        <TableHead
                          className="ivc__th"
                        >
                          単位
                        </TableHead>
                      </>
                    )
                    : (
                      <>
                        <TableHead
                          className="ivc__th"
                        >
                          サイズ
                        </TableHead>

                        <TableHead
                          className="ivc__th"
                        >
                          カラー
                        </TableHead>
                      </>
                    )}

                  <TableHead
                    className="ivc__th ivc__th--right"
                  >
                    在庫数
                  </TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {rows.map(
                  (
                    row,
                  ) => {
                    const rgbHex =
                      rgbIntToHex(
                        row.rgb,
                      ) ??
                      null;

                    const bgColor =
                      rgbHex ??
                      "#ffffff";

                    return (
                      <TableRow
                        key={
                          row.modelId
                        }
                        className="ivc__tr"
                      >
                        <TableCell
                          className="ivc__model"
                        >
                          {
                            row.modelNumber
                          }
                        </TableCell>

                        {isAlcoholCategory
                          ? (
                            <>
                              <TableCell
                                className="ivc__size"
                              >
                                {
                                  getVolumeValueLabel(
                                    row,
                                  ) ||
                                  "-"
                                }
                              </TableCell>

                              <TableCell
                                className="ivc__size"
                              >
                                {
                                  getVolumeUnitLabel(
                                    row,
                                  ) ||
                                  "-"
                                }
                              </TableCell>
                            </>
                          )
                          : (
                            <>
                              <TableCell
                                className="ivc__size"
                              >
                                {
                                  row.size ||
                                  "-"
                                }
                              </TableCell>

                              <TableCell
                                className="ivc__color-cell"
                              >
                                <span
                                  className="ivc__color-dot"
                                  style={{
                                    backgroundColor:
                                      bgColor,

                                    boxShadow:
                                      "0 0 0 1px rgba(0,0,0,0.18)",
                                  }}
                                  title={
                                    rgbHex ??
                                    ""
                                  }
                                />

                                <span
                                  className="ivc__color-label"
                                >
                                  {
                                    row.color ||
                                    "-"
                                  }
                                </span>
                              </TableCell>
                            </>
                          )}

                        <TableCell
                          className="ivc__stock"
                        >
                          <span
                            className="ivc__stock-number"
                          >
                            {
                              row.stock
                            }
                          </span>
                        </TableCell>
                      </TableRow>
                    );
                  },
                )}

                {rows.length ===
                  0 && (
                  <TableRow>
                    <TableCell
                      colSpan={4}
                      className="ivc__empty"
                    >
                      表示できる在庫データがありません。
                    </TableCell>
                  </TableRow>
                )}

                {rows.length >
                  0 && (
                  <TableRow
                    className="ivc__total-row"
                  >
                    <TableCell
                      colSpan={
                        footerColSpan
                      }
                      className="ivc__total-label ivc__th--right"
                    >
                      合計
                    </TableCell>

                    <TableCell
                      className="ivc__total-value"
                    >
                      <strong>
                        {
                          totalStock
                        }
                      </strong>
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

export default InventoryCard;