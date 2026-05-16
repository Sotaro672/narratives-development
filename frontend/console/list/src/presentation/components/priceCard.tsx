// frontend/console/list/src/presentation/components/priceCard.tsx

import * as React from "react";
import { Tag } from "lucide-react";
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

// ロジックは hook に寄せる
import { usePriceCard } from "../hook/usePriceCard";

// 型は inventory/application を正とする
import type { PriceCardProps } from "../../../../inventory/src/application/listCreate/listCreate.types";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

type PriceCardRowVMWithCategory = {
  modelId: string;

  kind?: string | null;

  size?: string | null;
  color?: string | null;
  bgColor?: string;
  rgbTitle?: string;

  volumeValue?: number | null;
  volumeUnit?: string | null;

  stock: number;

  priceInputValue: string;
  priceDisplayText: string;
  onChangePriceInput: React.ChangeEventHandler<HTMLInputElement>;
};

type PriceCardPropsWithCategory = PriceCardProps & {
  /**
   * ProductBlueprintCategory.code を渡す想定。
   *
   * 例:
   * - "apparel.tops"
   * - "alcohol.sake"
   */
  productBlueprintCategory?: string;
};

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: PriceCardRowVMWithCategory[];
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

function getVolumeValueLabel(row: PriceCardRowVMWithCategory): string {
  const value = row.volumeValue;

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function getVolumeUnitLabel(row: PriceCardRowVMWithCategory): string {
  return String(row.volumeUnit ?? "").trim();
}

const PriceCard: React.FC<PriceCardProps> = (props) => {
  const { className } = props;

  const {
    title,
    mode,
    isEdit,
    showModeBadge,
    currencySymbol,
    rowsVM,
    isEmpty,
  } = usePriceCard(props);

  const rowsWithCategory = React.useMemo(
    () => rowsVM as PriceCardRowVMWithCategory[],
    [rowsVM],
  );

  const productBlueprintCategory = (props as PriceCardPropsWithCategory)
    .productBlueprintCategory;

  const categoryKind = React.useMemo(
    () =>
      resolveProductBlueprintCategoryKind({
        productBlueprintCategory,
        rows: rowsWithCategory,
      }),
    [productBlueprintCategory, rowsWithCategory],
  );

  const isAlcoholCategory = categoryKind === "alcohol";

  return (
    <Card className={`prc ${className ?? ""}`}>
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
                    <TableHead className="prc__th">容量</TableHead>
                    <TableHead className="prc__th">単位</TableHead>
                  </>
                ) : (
                  <>
                    <TableHead className="prc__th">サイズ</TableHead>
                    <TableHead className="prc__th">カラー</TableHead>
                  </>
                )}

                <TableHead className="prc__th prc__th--right">
                  在庫数
                </TableHead>
                <TableHead className="prc__th prc__th--right">価格</TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {rowsWithCategory.map((row) => {
                return (
                  <TableRow key={row.modelId} className="prc__tr">
                    {isAlcoholCategory ? (
                      <>
                        <TableCell className="prc__size">
                          {getVolumeValueLabel(row) || "-"}
                        </TableCell>

                        <TableCell className="prc__size">
                          {getVolumeUnitLabel(row) || "-"}
                        </TableCell>
                      </>
                    ) : (
                      <>
                        <TableCell className="prc__size">
                          {row.size || "-"}
                        </TableCell>

                        <TableCell className="prc__color-cell">
                          <span
                            className="prc__color-dot"
                            style={{
                              backgroundColor: row.bgColor ?? "#ffffff",
                            }}
                            title={row.rgbTitle ?? ""}
                          />
                          <span className="prc__color-label">
                            {row.color || "-"}
                          </span>
                        </TableCell>
                      </>
                    )}

                    <TableCell className="prc__stock text-right">
                      <span className="prc__stock-number">{row.stock}</span>
                    </TableCell>

                    <TableCell className="prc__price text-right">
                      {isEdit ? (
                        <div className="flex items-center gap-2 justify-end">
                          {currencySymbol ? (
                            <span className="text-xs text-[hsl(var(--muted-foreground))]">
                              {currencySymbol}
                            </span>
                          ) : null}
                          <Input
                            inputMode="numeric"
                            type="number"
                            min={0}
                            step={1}
                            className="h-8 w-32 text-right"
                            value={row.priceInputValue}
                            placeholder="-"
                            onChange={row.onChangePriceInput}
                          />
                        </div>
                      ) : (
                        <span className="prc__price-value">
                          {row.priceDisplayText}
                        </span>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}

              {isEmpty && (
                <TableRow>
                  <TableCell colSpan={4} className="prc__empty">
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