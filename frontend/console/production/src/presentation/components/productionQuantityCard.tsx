// frontend/console/production/src/presentation/components/productionQuantityCard.tsx

import * as React from "react";
import { Palette } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../../shell/src/shared/ui/table";
import { Input } from "../../../../shell/src/shared/ui/input";

import "../styles/production.css";
import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

type ProductionQuantityCardProps = {
  title?: string;
  rows: ProductionQuantityRowVM[];

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
  onChangeRows?: (rows: ProductionQuantityRowVM[]) => void;
};

function displayOrderRank(v: unknown): number {
  return Number.isFinite(v as number)
    ? (v as number)
    : Number.POSITIVE_INFINITY;
}

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: ProductionQuantityRowVM[];
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

  const hasAlcoholRow = args.rows.some((row) => row.kind === "alcohol");
  if (hasAlcoholRow) {
    return "alcohol";
  }

  const hasApparelRow = args.rows.some((row) => row.kind === "apparel");
  if (hasApparelRow) {
    return "apparel";
  }

  return "unknown";
}

function getVolumeValueLabel(row: ProductionQuantityRowVM): string {
  const value = row.volumeValue;

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function getVolumeUnitLabel(row: ProductionQuantityRowVM): string {
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
    const safe = Array.isArray(rows) ? rows : [];
    const copied = [...safe];

    copied.sort((a, b) => {
      const da = displayOrderRank(a.displayOrder);
      const db = displayOrderRank(b.displayOrder);
      return da - db;
    });

    return copied;
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
  const isApparelCategory = categoryKind === "apparel";

  const totalQuantity = React.useMemo(
    () => sortedRows.reduce((sum, r) => sum + (r.quantity || 0), 0),
    [sortedRows],
  );

  const handleChangeQuantity = React.useCallback(
    (modelId: string, value: string) => {
      if (!onChangeRows) return;

      const n = Math.max(0, Math.floor(Number(value || "0")));
      const safe = Number.isFinite(n) ? n : 0;

      const next = sortedRows.map((row) =>
        row.modelId === modelId ? { ...row, quantity: safe } : row,
      );

      onChangeRows(next);
    },
    [sortedRows, onChangeRows],
  );

  const footerColSpan = isAlcoholCategory ? 3 : 3;

  return (
    <Card className={`mqc ${className ?? ""}`}>
      <CardHeader className="mqc__header">
        <div className="mqc__header-inner">
          <Palette size={18} />
          <CardTitle className="mqc__title">{title}</CardTitle>
        </div>
      </CardHeader>

      <CardContent className="mqc__body">
        <Table className="mqc__table">
          <TableHeader>
            <TableRow>
              <TableHead className="mqc__th mqc__th--left">型番</TableHead>

              {isAlcoholCategory ? (
                <>
                  <TableHead className="mqc__th">容量</TableHead>
                  <TableHead className="mqc__th">単位</TableHead>
                </>
              ) : (
                <>
                  <TableHead className="mqc__th">サイズ</TableHead>
                  <TableHead className="mqc__th">カラー</TableHead>
                </>
              )}

              <TableHead className="mqc__th mqc__cell">生産数</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {sortedRows.map((row) => {
              const rgbHex = rgbIntToHex(row.rgb) ?? null;
              const bgColor = rgbHex ?? "#ffffff";

              return (
                <TableRow key={row.modelId}>
                  <TableCell>{row.modelNumber}</TableCell>

                  {isAlcoholCategory ? (
                    <>
                      <TableCell className="mqc__size">
                        {getVolumeValueLabel(row) || "-"}
                      </TableCell>
                      <TableCell className="mqc__size">
                        {getVolumeUnitLabel(row) || "-"}
                      </TableCell>
                    </>
                  ) : (
                    <>
                      <TableCell className="mqc__size">{row.size}</TableCell>

                      <TableCell>
                        <span className="mqc__color">
                          <span
                            className="mqc__color-dot"
                            style={{ backgroundColor: bgColor }}
                            title={rgbHex ?? ""}
                          />
                          <span>{row.color}</span>
                        </span>
                      </TableCell>
                    </>
                  )}

                  <TableCell className="mqc__cell">
                    {isEditable ? (
                      <Input
                        type="number"
                        min={0}
                        step={1}
                        value={row.quantity ?? 0}
                        onChange={(e) =>
                          handleChangeQuantity(row.modelId, e.target.value)
                        }
                        className="mqc__input"
                        aria-label={`${row.modelNumber} の生産数`}
                      />
                    ) : (
                      <span>{row.quantity}</span>
                    )}
                  </TableCell>
                </TableRow>
              );
            })}

            {sortedRows.length > 0 && (
              <TableRow className="mqc__footer-row">
                <TableCell colSpan={footerColSpan} className="mqc__footer-label">
                  合計
                </TableCell>
                <TableCell className="mqc__footer-cell">
                  <span className="mqc__pill mqc__pill--total">
                    {totalQuantity}
                  </span>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default ProductionQuantityCard;
