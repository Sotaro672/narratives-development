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

// ★ 商品設計側のカタログから採寸ラベルを受け取る
import type { MeasurementOption } from "../../../../productBlueprint/src/domain/entity/catalog";

// ロジックは hook 側に集約
import {
  useSizeVariationCard,
  type SizeRow,
  type SizePatch,
} from "../hook/useModelCard";

type SizeVariationCardProps = {
  sizes: SizeRow[];
  onRemove: (id: string) => void;
  onChangeSize?: (id: string, patch: SizePatch) => void;
  mode?: "edit" | "view";

  /** 商品設計側から渡される採寸定義（itemType に連動） */
  measurementOptions?: MeasurementOption[];

  /** ヘッダー右端の「サイズを追加」ボタン押下時のハンドラ */
  onAddSize?: () => void;
};

// Measurement のラベル → SizeRow のどのフィールドか、の対応表
type SizeFieldKey = keyof Omit<
  SizeRow,
  "id" | "sizeLabel"
>;

function mapLabelToField(label: string): SizeFieldKey {
  switch (label) {
    // トップス
    case "着丈":
      return "length";
    case "身幅":
      // 仕様的には身幅＝chest とみなす
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

    // フォールバック（旧ヘッダーなど）
    case "胸囲":
      return "chest";

    default:
      // 万一未知ラベルの場合は chest にフォールバック
      return "chest";
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
  // ★ ビューロジックは hook に委譲
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

  // カラム定義: label と SizeRow のフィールド名を紐づけ
  const measurementCols = React.useMemo(
    () =>
      (measurementHeaders ?? []).map((label) => ({
        label,
        field: mapLabelToField(label),
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

        {isEdit && onAddSize && (
          <Button
            type="button"
            size="sm"
            variant="outline"
            className="ml-auto"
            onClick={onAddSize}
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
              {/* ★ 渡された measurement に応じて列名・列数を制御 */}
              {measurementCols.map((col) => (
                <TableHead key={col.label}>{col.label}(cm)</TableHead>
              ))}
              {isEdit && <TableHead />} {/* 削除列（編集時のみ） */}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sizes.map((row) => (
              <TableRow key={row.id}>
                <TableCell>
                  <Input
                    {...readonlyInputProps}
                    value={row.sizeLabel}
                    onChange={handleChange(row.id, "sizeLabel")}
                    aria-label={`${row.sizeLabel} サイズ名`}
                  />
                </TableCell>

                {/* 採寸列はラベル→フィールドの対応表に基づいて動的に描画 */}
                {measurementCols.map((col) => (
                  <TableCell key={col.field}>
                    <Input
                      {...readonlyInputProps}
                      type="number"
                      inputMode="decimal"
                      value={row[col.field] ?? ""}
                      // handleChange の key は keyof Omit<SizeRow, "id"> なので
                      // col.field をそのまま渡して OK
                      onChange={handleChange(
                        row.id,
                        col.field as keyof Omit<SizeRow, "id">,
                      )}
                      aria-label={`${row.sizeLabel} ${col.label}`}
                    />
                  </TableCell>
                ))}

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
                  colSpan={
                    // サイズ列 + measurement 列数 + 削除列（編集時のみ）
                    1 + measurementCols.length + (isEdit ? 1 : 0)
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
