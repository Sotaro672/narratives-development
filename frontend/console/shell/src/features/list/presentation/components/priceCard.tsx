// frontend/console/shell/src/features/list/presentation/components/priceCard.tsx

import * as React from "react";
import { Tag } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderIcon,
  CardHeaderLeft,
  CardInput,
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
import Text from "../../../../shared/ui/text";

import { usePriceCard } from "../hook/usePriceCard";

import type {
  PriceCardProps,
  PriceRowVM,
} from "../../../inventory/application/listCreateService";

import "../../../../styles/list.css";

type ProductBlueprintCategoryKind = "apparel" | "alcohol" | "unknown";

function resolveProductBlueprintCategoryKind(args: {
  productBlueprintCategory?: string;
  rows: PriceRowVM[];
}): ProductBlueprintCategoryKind {
  const category = String(args.productBlueprintCategory ?? "").trim().toLowerCase();

  if (category.startsWith("alcohol")) return "alcohol";
  if (category.startsWith("apparel")) return "apparel";

  const hasAlcoholRow = args.rows.some((row) => row.kind === "alcohol");
  if (hasAlcoholRow) return "alcohol";

  const hasApparelRow = args.rows.some((row) => row.kind === "apparel");
  if (hasApparelRow) return "apparel";

  return "unknown";
}

function getVolumeValueLabel(row: PriceRowVM): string {
  const value = row.volumeValue;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

function getVolumeUnitLabel(row: PriceRowVM): string {
  return String(row.volumeUnit ?? "").trim();
}

const PriceCard: React.FC<PriceCardProps> = (props) => {
  const { className, productBlueprintCategory } = props;

  const {
    title,
    mode,
    isEdit,
    showModeBadge,
    currencySymbol,
    rowsVM,
    isEmpty,
  } = usePriceCard(props);

  const categoryKind = React.useMemo(
    () =>
      resolveProductBlueprintCategoryKind({
        productBlueprintCategory,
        rows: rowsVM,
      }),
    [productBlueprintCategory, rowsVM],
  );

  const isAlcoholCategory = categoryKind === "alcohol";

  return (
    <Card className={className}>
      <CardHeader>
        <CardHeaderLeft>
          <CardHeaderIcon>
            <Tag className="card__header-icon-svg" />
          </CardHeaderIcon>

          <CardTitle strong>
            {title}
            {showModeBadge && (
              <Text size="xs" tone="muted" className="price-card__mode">
                （{mode}）
              </Text>
            )}
          </CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
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

              <TableHead align="right">在庫数</TableHead>
              <TableHead align="right">価格</TableHead>
            </TableRow>
          </TableHeader>

          <TableBody>
            {rowsVM.map((row) => (
              <TableRow key={row.modelId}>
                {isAlcoholCategory ? (
                  <>
                    <TableCell>{getVolumeValueLabel(row) || "-"}</TableCell>
                    <TableCell>{getVolumeUnitLabel(row) || "-"}</TableCell>
                  </>
                ) : (
                  <>
                    <TableCell>{row.size || "-"}</TableCell>
                    <TableCell className="text--wrap-nowrap">
                      <ColorValue
                        color={row.bgColor}
                        swatchTitle={row.rgbTitle || undefined}
                      >
                        {row.color || "-"}
                      </ColorValue>
                    </TableCell>
                  </>
                )}

                <TableCell>{row.stock}</TableCell>

                <TableCell>
                  {isEdit ? (
                    <div className="price-card__price-editor">
                      {currencySymbol ? (
                        <Text size="xs" tone="muted">
                          {currencySymbol}
                        </Text>
                      ) : null}

                      <CardInput
                        required
                        inputMode="numeric"
                        type="number"
                        min={0}
                        step={1}
                        sizeVariant="sm"
                        className="price-card__price-input"
                        value={row.priceInputValue}
                        placeholder="-"
                        onChange={row.onChangePriceInput}
                      />
                    </div>
                  ) : (
                    row.priceDisplayText
                  )}
                </TableCell>
              </TableRow>
            ))}

            {isEmpty && (
              <TableRow>
                <TableCell colSpan={4}>
                  表示できるデータがありません。
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default PriceCard;