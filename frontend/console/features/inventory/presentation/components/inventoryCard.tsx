// frontend/console/inventory/src/presentation/components/inventoryCard.tsx

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

import type { InventoryRow } from "../../application/inventoryTypes";

import { rgbIntToHex } from "../../../../shell/src/shared/util/color";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

type InventoryCardProps = {
  title?: string;
  rows: InventoryRow[];

  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;

  className?: string;
  mode?: "view"; // 現状は閲覧専用
};

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: InventoryRow[];
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

function getVolumeValueLabel(row: InventoryRow): string {
  const value = row.volumeValue;

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function getVolumeUnitLabel(row: InventoryRow): string {
  return String(row.volumeUnit ?? "").trim();
}

const InventoryCard: React.FC<InventoryCardProps> = ({
  title = "モデル別在庫一覧",
  rows,
  productBlueprintCategory,
  className,
  mode = "view",
}) => {
  // displayOrder のみに従って昇順ソート
  // - displayOrder が無い行は最後
  // - displayOrder が同じ場合は「元の順番」を維持（副基準で並べ替えない）
  const sortedRows = React.useMemo(() => {
    const src = rows ?? [];

    const orderOf = (r: InventoryRow): number => {
      const n = Number(r.displayOrder);
      return Number.isFinite(n) && n > 0 ? n : Number.POSITIVE_INFINITY;
    };

    // 安定ソート（同一 displayOrder は入力順維持）
    return src
      .map((r, idx) => ({ r, idx }))
      .sort((a, b) => {
        const oa = orderOf(a.r);
        const ob = orderOf(b.r);
        if (oa !== ob) return oa - ob;
        return a.idx - b.idx;
      })
      .map((x) => x.r);
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

  const totalStock = React.useMemo(
    () => sortedRows.reduce((sum, r) => sum + (r.stock || 0), 0),
    [sortedRows],
  );

  const footerColSpan = 3;

  return (
    <Card className={`ivc ${className ?? ""}`}>
      <CardHeader className="ivc__header">
        <div className="ivc__header-inner">
          <Palette className="ivc__icon" size={18} />
          <CardTitle className="ivc__title">
            {title}
            {mode !== "view" && (
              <span className="ml-2 text-xs text-[hsl(var(--muted-foreground))]">
                （{mode}）
              </span>
            )}
          </CardTitle>
        </div>
      </CardHeader>

      <CardContent className="ivc__body">
        <div className="ivc__table-wrap">
          <Table className="ivc__table">
            <TableHeader>
              <TableRow>
                <TableHead className="ivc__th ivc__th--left">型番</TableHead>

                {isAlcoholCategory ? (
                  <>
                    <TableHead className="ivc__th">容量</TableHead>
                    <TableHead className="ivc__th">単位</TableHead>
                  </>
                ) : (
                  <>
                    <TableHead className="ivc__th">サイズ</TableHead>
                    <TableHead className="ivc__th">カラー</TableHead>
                  </>
                )}

                <TableHead className="ivc__th ivc__th--right">
                  在庫数
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {sortedRows.map((row, idx) => {
                const rgbHex = rgbIntToHex(row.rgb) ?? null;
                const bgColor = rgbHex ?? "#ffffff";

                return (
                  <TableRow
                    key={`${row.modelNumber}-${idx}`}
                    className="ivc__tr"
                  >
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
                              backgroundColor: bgColor,
                              boxShadow: "0 0 0 1px rgba(0,0,0,0.18)",
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
                      <span className="ivc__stock-number">{row.stock}</span>
                    </TableCell>
                  </TableRow>
                );
              })}

              {sortedRows.length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="ivc__empty">
                    表示できる在庫データがありません。
                  </TableCell>
                </TableRow>
              )}

              {sortedRows.length > 0 && (
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