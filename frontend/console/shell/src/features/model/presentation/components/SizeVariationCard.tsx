// frontend/console/shell/src/features/model/presentation/components/SizeVariationCard.tsx

import * as React from "react";
import { Tags, Trash2 } from "lucide-react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui";

import { Button } from "../../../../shared/ui/button";
import { Input } from "../../../../shared/ui/input";

import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../../shared/ui/table";

import type {
  MeasurementOption,
} from "../../../../shared/types/apparel";

// ロジックはhook側に集約
import {
  useSizeVariationCard,
  type SizeRow,
  type SizePatch,
} from "../hook/useModelCard";

export type SizeVariationCardProps = {
  sizes: SizeRow[];
  onRemove: (id: string) => void;
  onChangeSize?: (
    id: string,
    patch: SizePatch,
  ) => void;
  mode?: "edit" | "view";
  measurementOptions?: MeasurementOption[];
  onAddSize?: () => void;
};

// MeasurementのラベルからSizeRowのfieldを取得する。
type SizeFieldKey = keyof Omit<
  SizeRow,
  "id" | "sizeLabel"
>;

/**
 * measurement labelからSizeRowのfieldへの対応表。
 */
function mapLabelToField(
  label: string,
): SizeFieldKey {
  switch (label) {
    // トップス
    case "着丈":
      return "length";

    case "身幅":
      return "width";

    case "胸囲":
      return "chest";

    case "肩幅":
      return "shoulder";

    case "袖丈":
      return "sleeveLength";

    // ボトムス
    case "ウエスト":
      return "waist";

    case "ヒップ":
      return "hip";

    case "股上":
      return "rise";

    case "股下":
      return "inseam";

    case "わたり幅":
      return "thigh";

    case "裾幅":
      return "hemWidth";

    default:
      throw new Error(
        `Unknown measurement label: ${label}`,
      );
  }
}

const SizeVariationCard: React.FC<
  SizeVariationCardProps
> = ({
  sizes,
  onRemove,
  onChangeSize,
  mode = "edit",
  measurementOptions,
  onAddSize,
}) => {
  const {
    isEdit,
    readonlyInputProps,
    measurementHeaders,
    handleChange,
  } = useSizeVariationCard({
    sizes,
    mode,
    measurementOptions,
    onChangeSize,
  });

  const measurementCols = React.useMemo(
    () =>
      (measurementHeaders ?? []).map(
        (label) => ({
          label,
          field: mapLabelToField(label),
        }),
      ),
    [measurementHeaders],
  );

  return (
    <Card
      className={`svc ${
        mode === "view" ? "view-mode" : ""
      }`}
    >
      <CardHeader className="box__header">
        <div className="flex items-center gap-2">
          <Tags size={16} />

          <CardTitle className="box__title">
            サイズバリエーション

            {mode === "view" && (
              <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
                （閲覧）
              </span>
            )}
          </CardTitle>
        </div>

        {isEdit && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="ml-auto"
            onClick={() => onAddSize?.()}
          >
            サイズを追加
          </Button>
        )}
      </CardHeader>

      <CardContent className="box__body">
        <Table className="svc__table">
          <TableHeader>
            <TableRow>
              <TableHead>
                サイズ
              </TableHead>

              {measurementCols.map((col) => (
                <TableHead key={col.label}>
                  {col.label}(cm)
                </TableHead>
              ))}

              {isEdit && (
                <TableHead />
              )}
            </TableRow>
          </TableHeader>

          <TableBody>
            {sizes.map((row) => (
              <TableRow key={row.id}>
                <TableCell>
                  {isEdit ? (
                    <Input
                      {...readonlyInputProps}
                      value={row.sizeLabel}
                      onChange={handleChange(
                        row.id,
                        "sizeLabel",
                      )}
                      aria-label={`${row.sizeLabel} サイズ名`}
                    />
                  ) : (
                    <span>
                      {row.sizeLabel}
                    </span>
                  )}
                </TableCell>

                {measurementCols.map((col) => (
                  <TableCell key={col.field}>
                    {isEdit ? (
                      <Input
                        {...readonlyInputProps}
                        type="number"
                        inputMode="decimal"
                        value={
                          row[col.field] ?? ""
                        }
                        onChange={handleChange(
                          row.id,
                          col.field,
                        )}
                        aria-label={`${row.sizeLabel} ${col.label}`}
                      />
                    ) : (
                      <span>
                        {row[col.field] !== undefined &&
                        row[col.field] !== null
                          ? String(row[col.field])
                          : ""}
                      </span>
                    )}
                  </TableCell>
                ))}

                {isEdit && (
                  <TableCell>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() =>
                        onRemove(row.id)
                      }
                      aria-label={`${row.sizeLabel} を削除`}
                      className="svc__remove"
                    >
                      <Trash2 size={16} />
                    </Button>
                  </TableCell>
                )}
              </TableRow>
            ))}

            {sizes.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={
                    1 +
                    measurementCols.length +
                    (isEdit ? 1 : 0)
                  }
                  className="svc__empty"
                >
                  登録されているサイズはありません。
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
};

export default SizeVariationCard;