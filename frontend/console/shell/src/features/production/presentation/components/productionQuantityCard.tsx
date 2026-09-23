// frontend/console/shell/src/features/production/presentation/components/productionQuantityCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";

import { Badge } from "../../../../shared/ui/badge";
import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardTitle,
} from "../../../../shared/ui/card";
import { ColorValue } from "../../../../shared/ui/color";
import { Input } from "../../../../shared/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import type { ProductionQuantityRow } from "../../../../shared/types/production";
import { rgbIntToHex } from "../../../../shared/util/color";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRow[];

  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;
  className?: string;
  mode?: "view" | "edit";
  onChangeRows?: (rows: ProductionQuantityRow[]) => void;
};

function displayOrderRank(value: unknown): number {
  return Number.isFinite(value as number)
    ? (value as number)
    : Number.POSITIVE_INFINITY;
}

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: ProductionQuantityRow[];
}): ProductBlueprintCategoryKind {
  const category = String(args.productBlueprintCategory ?? "")
    .trim()
    .toLowerCase();

  if (category.startsWith("alcohol") || category.includes(".sake")) {
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

function getVolumeValueLabel(row: ProductionQuantityRow): string {
  const value = row.volumeValue;

  return typeof value === "number" && Number.isFinite(value)
    ? String(value)
    : "";
}

function getVolumeUnitLabel(row: ProductionQuantityRow): string {
  return String(row.volumeUnit ?? "").trim();
}

const ProductionQuantityCard: React.FC<ProductionQuantityCardProps> = ({
  title = "モデル別生産数一覧",
  rows,
  productBlueprintCategory,
  className,
  mode = "view",
  onChangeRows,
}) => {
  const isEditable = mode === "edit";

  const sortedRows = React.useMemo(() => {
    const safeRows = Array.isArray(rows) ? rows : [];

    return [...safeRows].sort(
      (a, b) =>
        displayOrderRank(a.displayOrder) -
        displayOrderRank(b.displayOrder),
    );
  }, [rows]);

  const categoryKind = React.useMemo(
    () =>
      resolveProductBlueprintCategoryKind({
        productBlueprintCategory,
        rows: sortedRows,
      }),
    [productBlueprintCategory, sortedRows],
  );

  const isAlcoholCategory = categoryKind === "alcohol";

  const totalQuantity = React.useMemo(
    () =>
      sortedRows.reduce(
        (sum, row) =>
          sum + (Number.isFinite(row.quantity) ? row.quantity : 0),
        0,
      ),
    [sortedRows],
  );

  const handleChangeQuantity = React.useCallback(
    (modelId: string, value: string) => {
      if (!onChangeRows) {
        return;
      }

      const quantityNumber = Number(value || "0");
      const quantity = Number.isFinite(quantityNumber)
        ? Math.max(0, Math.floor(quantityNumber))
        : 0;

      onChangeRows(
        sortedRows.map((row) =>
          row.modelId === modelId
            ? {
                ...row,
                quantity,
              }
            : row,
        ),
      );
    },
    [sortedRows, onChangeRows],
  );

  return (
    <Card className={className}>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <Palette className="card__header-icon-svg" />
          </CardHeaderIcon>

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

              <TableHead align="center">生産数</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {sortedRows.map((row) => {
              const rgbHex =
                typeof row.rgb === "number"
                  ? rgbIntToHex(row.rgb) ?? null
                  : null;

              const backgroundColor = rgbHex ?? "#ffffff";
              const modelNumber = row.modelNumber ?? "-";

              return (
                <TableRow key={row.modelId}>
                  <TableCell>{modelNumber}</TableCell>

                  {isAlcoholCategory ? (
                    <>
                      <TableCell>{getVolumeValueLabel(row) || "-"}</TableCell>
                      <TableCell>{getVolumeUnitLabel(row) || "-"}</TableCell>
                    </>
                  ) : (
                    <>
                      <TableCell>{row.size ?? "-"}</TableCell>

                      <TableCell>
                        <ColorValue
                          color={backgroundColor}
                          swatchTitle={rgbHex ?? undefined}
                        >
                          {row.color ?? "-"}
                        </ColorValue>
                      </TableCell>
                    </>
                  )}

                  <TableCell align="center">
                    {isEditable ? (
                      <Input
                        type="number"
                        min={0}
                        step={1}
                        value={row.quantity}
                        onChange={(event) =>
                          handleChangeQuantity(
                            row.modelId,
                            event.target.value,
                          )
                        }
                        aria-label={`${modelNumber} の生産数`}
                      />
                    ) : (
                      row.quantity
                    )}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>

          {sortedRows.length > 0 && (
            <TableFooter>
              <TableRow>
                <TableCell colSpan={3}>合計</TableCell>
                <TableCell align="center">
                  <Badge variant="active">{totalQuantity}</Badge>
                </TableCell>
              </TableRow>
            </TableFooter>
          )}
        </Table>
      </CardContent>
    </Card>
  );
};

export default ProductionQuantityCard;