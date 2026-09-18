// frontend/console/shell/src/features/inventory/presentation/components/inventoryCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
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
import type { InventoryDetailRowDTO } from "../../../../shared/types/inventory";
import { rgbIntToHex } from "../../../../shared/util/color";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

type InventoryCardProps = {
  title?: string;
  rows: InventoryDetailRowDTO[];

  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;
  className?: string;
  mode?: "view";
};

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: InventoryDetailRowDTO[];
}): ProductBlueprintCategoryKind {
  const category = String(args.productBlueprintCategory ?? "")
    .trim()
    .toLowerCase();

  if (category.startsWith("alcohol")) {
    return "alcohol";
  }

  if (category.startsWith("apparel")) {
    return "apparel";
  }

  if (args.rows.some((row) => row.kind === "alcohol")) {
    return "alcohol";
  }

  if (args.rows.some((row) => row.kind === "apparel")) {
    return "apparel";
  }

  return "unknown";
}

function getVolumeValueLabel(row: InventoryDetailRowDTO): string {
  const value = row.volumeValue;

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function getVolumeUnitLabel(row: InventoryDetailRowDTO): string {
  return String(row.volumeUnit ?? "").trim();
}

const InventoryCard: React.FC<InventoryCardProps> = ({
  title = "モデル別在庫一覧",
  rows,
  productBlueprintCategory,
  className,
}) => {
  const categoryKind = React.useMemo(
    () =>
      resolveProductBlueprintCategoryKind({
        productBlueprintCategory,
        rows,
      }),
    [productBlueprintCategory, rows],
  );

  const isAlcoholCategory = categoryKind === "alcohol";

  const totalStock = React.useMemo(
    () => rows.reduce((sum, row) => sum + row.stock, 0),
    [rows],
  );

  const footerColSpan = 3;

  return (
    <Card className={className}>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <Palette className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            {title}
          </CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <div className="ivc__table-wrap">
          <Table className="ivc__table">
            <TableHeader>
              <TableRow>
                <TableHead className="ivc__th ivc__th--left">
                  型番
                </TableHead>

                {isAlcoholCategory ? (
                  <>
                    <TableHead className="ivc__th">
                      容量
                    </TableHead>
                    <TableHead className="ivc__th">
                      単位
                    </TableHead>
                  </>
                ) : (
                  <>
                    <TableHead className="ivc__th">
                      サイズ
                    </TableHead>
                    <TableHead className="ivc__th">
                      カラー
                    </TableHead>
                  </>
                )}

                <TableHead className="ivc__th ivc__th--right">
                  在庫数
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rows.map((row) => {
                const rgbHex = rgbIntToHex(row.rgb) ?? null;
                const backgroundColor = rgbHex ?? "#ffffff";

                return (
                  <TableRow key={row.modelId} className="ivc__tr">
                    <TableCell className="ivc__model">
                      {row.modelNumber}
                    </TableCell>

                    {isAlcoholCategory ? (
                      <>
                        <TableCell className="ivc__size">
                          {getVolumeValueLabel(row) || "-"}
                        </TableCell>
                        <TableCell className="ivc__size">
                          {getVolumeUnitLabel(row) || "-"}
                        </TableCell>
                      </>
                    ) : (
                      <>
                        <TableCell className="ivc__size">
                          {row.size || "-"}
                        </TableCell>
                        <TableCell className="ivc__color-cell">
                          <span
                            className="ivc__color-dot"
                            style={{
                              backgroundColor,
                              boxShadow: "0 0 0 1px rgba(0, 0, 0, 0.18)",
                            }}
                            title={rgbHex ?? ""}
                          />
                          <span className="ivc__color-label">
                            {row.color || "-"}
                          </span>
                        </TableCell>
                      </>
                    )}

                    <TableCell className="ivc__stock">
                      <span className="ivc__stock-number">
                        {row.stock}
                      </span>
                    </TableCell>
                  </TableRow>
                );
              })}

              {rows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる在庫データがありません。
                  </TableCell>
                </TableRow>
              )}

              {rows.length > 0 && (
                <TableRow className="ivc__total-row">
                  <TableCell
                    colSpan={footerColSpan}
                    className="ivc__total-label ivc__th--right"
                  >
                    合計
                  </TableCell>
                  <TableCell className="ivc__total-value">
                    <strong>{totalStock}</strong>
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