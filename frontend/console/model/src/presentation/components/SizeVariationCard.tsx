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

export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest?: number;
  waist?: number;
  length?: number;
  shoulder?: number;
};

type SizePatch = Partial<Omit<SizeRow, "id">>;

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

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({
  sizes,
  onRemove,
  onChangeSize,
  mode = "edit",
  measurementOptions,
  onAddSize,
}) => {
  const isEdit = mode === "edit";

  const handleChange =
    (id: string, key: keyof Omit<SizeRow, "id">) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit || !onChangeSize) return;

      const v = e.target.value;

      if (key === "sizeLabel") {
        onChangeSize(id, { sizeLabel: v });
      } else {
        onChangeSize(id, {
          [key]: v === "" ? undefined : Number(v),
        } as SizePatch);
      }
    };

  // 閲覧モードのみ readOnly スタイルを適用
  const readonlyProps =
    !isEdit
      ? ({ variant: "readonly" as const, readOnly: true } as const)
      : ({} as const);

  // ★ カタログの measurement に応じてヘッダ名を切り替える
  //   - 現在の実装は 4 列分の数値カラムを持っているため、最大 4 つまで使用
  const measurementHeaders = React.useMemo(() => {
    if (!measurementOptions || measurementOptions.length === 0) {
      // フォールバック（従来のヘッダー）
      return ["胸囲", "ウエスト", "着丈", "肩幅"];
    }
    return measurementOptions.map((m) => m.label).slice(0, 4);
  }, [measurementOptions]);

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
              {measurementHeaders.map((label) => (
                <TableHead key={label}>{label}(cm)</TableHead>
              ))}
              {isEdit && <TableHead />} {/* 削除列（編集時のみ） */}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sizes.map((row) => (
              <TableRow key={row.id}>
                <TableCell>
                  <Input
                    {...readonlyProps}
                    value={row.sizeLabel}
                    onChange={handleChange(row.id, "sizeLabel")}
                    aria-label={`${row.sizeLabel} サイズ名`}
                  />
                </TableCell>

                {/* 1 列目（旧: chest） */}
                {measurementHeaders[0] && (
                  <TableCell>
                    <Input
                      {...readonlyProps}
                      type="number"
                      inputMode="decimal"
                      value={row.chest ?? ""}
                      onChange={handleChange(row.id, "chest")}
                      aria-label={`${row.sizeLabel} ${measurementHeaders[0]}`}
                    />
                  </TableCell>
                )}

                {/* 2 列目（旧: waist） */}
                {measurementHeaders[1] && (
                  <TableCell>
                    <Input
                      {...readonlyProps}
                      type="number"
                      inputMode="decimal"
                      value={row.waist ?? ""}
                      onChange={handleChange(row.id, "waist")}
                      aria-label={`${row.sizeLabel} ${measurementHeaders[1]}`}
                    />
                  </TableCell>
                )}

                {/* 3 列目（旧: length） */}
                {measurementHeaders[2] && (
                  <TableCell>
                    <Input
                      {...readonlyProps}
                      type="number"
                      inputMode="decimal"
                      value={row.length ?? ""}
                      onChange={handleChange(row.id, "length")}
                      aria-label={`${row.sizeLabel} ${measurementHeaders[2]}`}
                    />
                  </TableCell>
                )}

                {/* 4 列目（旧: shoulder） */}
                {measurementHeaders[3] && (
                  <TableCell>
                    <Input
                      {...readonlyProps}
                      type="number"
                      inputMode="decimal"
                      value={row.shoulder ?? ""}
                      onChange={handleChange(row.id, "shoulder")}
                      aria-label={`${row.sizeLabel} ${measurementHeaders[3]}`}
                    />
                  </TableCell>
                )}

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
                    1 + measurementHeaders.length + (isEdit ? 1 : 0)
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
