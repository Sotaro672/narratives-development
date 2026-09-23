// frontend/console/shell/src/features/inventory/presentation/components/inventoryCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import { ColorValue } from "../../../../shared/ui/color";
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
  const category = String(args.productBlueprintCategory ?? "").trim().toLowerCase();

  if (category.startsWith("alcohol")) return "alcohol";
  if (category.startsWith("apparel")) return "apparel";
  if (args.rows.some((row) => row.kind === "alcohol")) return "alcohol";
  if (args.rows.some((row) => row.kind === "apparel")) return "apparel";

  return "unknown";
}

function getVolumeValueLabel(row: InventoryDetailRowDTO): string {
  const value = row.volumeValue;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
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
          <CardTitle strong>{title}</CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>型番</TableHead>

              {isAlcoholCategory ? (
                <>
                  <TableHead>容量</TableHead>
                  <TableHead>単位</TableHead>
                </>
              ) : (
                <>
                  <TableHead>サイズ</TableHead>
                  <TableHead>カラー</TableHead>
                </>
              )}

              <TableHead>在庫数</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {rows.map((row) => {
              const rgbHex = rgbIntToHex(row.rgb) ?? null;
              const backgroundColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={row.modelId}>
                  <TableCell>{row.modelNumber}</TableCell>

                  {isAlcoholCategory ? (
                    <>
                      <TableCell>{getVolumeValueLabel(row) || "-"}</TableCell>
                      <TableCell>{getVolumeUnitLabel(row) || "-"}</TableCell>
                    </>
                  ) : (
                    <>
                      <TableCell>{row.size || "-"}</TableCell>
                      <TableCell>
                        <ColorValue
                          color={backgroundColor}
                          swatchTitle={rgbHex ?? undefined}
                        >
                          {row.color || "-"}
                        </ColorValue>
                      </TableCell>
                    </>
                  )}

                  <TableCell>{row.stock}</TableCell>
                </TableRow>
              );
            })}

            {rows.length === 0 && (
              <TableRow>
                <TableCell colSpan={4}>
                  表示できる在庫データがありません。
                </TableCell>
              </TableRow>
            )}

            {rows.length > 0 && (
              <TableRow>
                <TableCell colSpan={footerColSpan}>合計</TableCell>
                <TableCell>
                  <strong>{totalStock}</strong>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default InventoryCard;