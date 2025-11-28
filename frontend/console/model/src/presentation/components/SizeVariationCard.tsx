// frontend/console/model/src/presentation/components/SizeVariationCard.tsx

import * as React from "react";
import { Tags, Trash2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../../shell/src/shared/ui/table";
import "../styles/model.css";
import "../../../../shell/src/shared/ui/card.css";

// ★ productBlueprint 側の catalog を import
import type {
  ItemType,
  MeasurementOption,
} from "../../../../productBlueprint/src/domain/entity/catalog";
import {
  ITEM_TYPE_MEASUREMENT_OPTIONS,
} from "../../../../productBlueprint/src/domain/entity/catalog";

// ロジックは hook 側に集約
import {
  useSizeVariationCard,
  type SizeRow,
  type SizePatch,
} from "../hook/useModelCard";

/** props */
export type SizeVariationCardProps = {
  sizes: SizeRow[];
  onRemove: (id: string) => void;
  onChangeSize?: (id: string, patch: SizePatch) => void;
  mode?: "edit" | "view";
  measurementOptions?: MeasurementOption[];
  onAddSize?: () => void;
};

// Measurement のラベル → SizeRow のどのフィールドか、の対応表
type SizeFieldKey = keyof Omit<SizeRow, "id" | "sizeLabel">;

/**
 * measurement label → SizeRow フィールド対応表
 */
function mapLabelToField(label: string): SizeFieldKey {
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
  }

  throw new Error(`Unknown measurement label: ${label}`);
}

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({
  sizes,
  onRemove,
  onChangeSize,
  mode = "edit",
  measurementOptions,
  onAddSize,
}) => {
  // hook ロジック
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

  // label → field 紐づけ
  const measurementCols = React.useMemo(
    () =>
      (measurementHeaders ?? []).map((label) => ({
        label,
        field: mapLabelToField(label) as SizeFieldKey,
      })),
    [measurementHeaders],
  );

  return (
    <Card className={`svc ${mode === "view" ? "view-mode" : ""}`}>
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

        {/* ★ edit モードのときだけボタンを表示（onAddSize は任意） */}
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
              <TableHead>サイズ</TableHead>

              {measurementCols.map((col) => (
                <TableHead key={col.label}>{col.label}(cm)</TableHead>
              ))}

              {isEdit && <TableHead />} {/* 削除列（view モードでは非表示） */}
            </TableRow>
          </TableHeader>

          <TableBody>
            {sizes.map((row) => (
              <TableRow key={row.id}>
                {/* サイズラベル */}
                <TableCell>
                  {isEdit ? (
                    <Input
                      {...readonlyInputProps}
                      value={row.sizeLabel}
                      onChange={handleChange(row.id, "sizeLabel")}
                      aria-label={`${row.sizeLabel} サイズ名`}
                    />
                  ) : (
                    <span>{row.sizeLabel}</span>
                  )}
                </TableCell>

                {/* 採寸列 */}
                {measurementCols.map((col) => (
                  <TableCell key={col.field}>
                    {isEdit ? (
                      <Input
                        {...readonlyInputProps}
                        type="number"
                        inputMode="decimal"
                        value={row[col.field] ?? ""}
                        onChange={handleChange(
                          row.id,
                          col.field as keyof Omit<SizeRow, "id">,
                        )}
                        aria-label={`${row.sizeLabel} ${col.label}`}
                      />
                    ) : (
                      <span>
                        {row[col.field] !== undefined && row[col.field] !== null
                          ? String(row[col.field])
                          : ""}
                      </span>
                    )}
                  </TableCell>
                ))}

                {/* 削除ボタン列（view モードでは非表示） */}
                {isEdit && (
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => onRemove(row.id)}
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
                  colSpan={1 + measurementCols.length + (isEdit ? 1 : 0)}
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

// default export
export default SizeVariationCard;
