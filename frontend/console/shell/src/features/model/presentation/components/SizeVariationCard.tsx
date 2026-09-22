// frontend/console/shell/src/features/model/presentation/components/SizeVariationCard.tsx

import * as React from "react";
import { Trash2 } from "lucide-react";

import {
  Card,
  CardButton,
  CardContent,
  CardHeader,
  CardHeaderLeft,
  CardInput,
  CardTitle,
} from "../../../../shared/ui";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import type { MeasurementOption } from "../../../../shared/types/apparel";

import {
  useSizeVariationCard,
  type SizePatch,
  type SizeRow,
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

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({
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
      (measurementHeaders ?? []).map((label) => ({
        label,
        field: mapLabelToField(label),
      })),
    [measurementHeaders],
  );

  return (
    <Card
      className={
        mode === "view"
          ? "view-mode"
          : undefined
      }
    >
      <CardHeader>
        <CardHeaderLeft>
          <CardTitle strong>
            サイズバリエーション
          </CardTitle>
        </CardHeaderLeft>

        {isEdit && (
          <CardButton
            type="button"
            variant="default"
            size="sm"
            onClick={() => onAddSize?.()}
          >
            サイズを追加
          </CardButton>
        )}
      </CardHeader>

      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>
                サイズ
              </TableHead>

              {measurementCols.map((col) => (
                <TableHead key={col.label}>
                  {col.label}(mm)
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
                    <CardInput
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
                      <CardInput
                        {...readonlyInputProps}
                        type="number"
                        min={0}
                        step={1}
                        inputMode="numeric"
                        value={row[col.field] ?? ""}
                        onChange={handleChange(
                          row.id,
                          col.field,
                        )}
                        aria-label={`${row.sizeLabel} ${col.label} mm`}
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
                    <CardButton
                      type="button"
                      variant="default"
                      size="icon"
                      onClick={() => onRemove(row.id)}
                      aria-label={`${row.sizeLabel} を削除`}
                    >
                      <Trash2 size={16} />
                    </CardButton>
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