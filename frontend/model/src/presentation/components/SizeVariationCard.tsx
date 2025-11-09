import * as React from "react";
import { Tags, Trash2 } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../../shared/ui";
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
import "../styles/model.css";
import "../../../shared/ui/card.css";

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
};

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({
  sizes,
  onRemove,
  onChangeSize,
  mode = "edit",
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

  return (
    <Card className={`svc ${mode === "view" ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">
          サイズバリエーション
          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
              （閲覧）
            </span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <Table className="svc__table">
          <TableHeader>
            <TableRow>
              <TableHead>サイズ</TableHead>
              <TableHead>胸囲(cm)</TableHead>
              <TableHead>ウエスト(cm)</TableHead>
              <TableHead>着丈(cm)</TableHead>
              <TableHead>肩幅(cm)</TableHead>
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
                <TableCell>
                  <Input
                    {...readonlyProps}
                    type="number"
                    inputMode="decimal"
                    value={row.chest ?? ""}
                    onChange={handleChange(row.id, "chest")}
                    aria-label={`${row.sizeLabel} 胸囲`}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    {...readonlyProps}
                    type="number"
                    inputMode="decimal"
                    value={row.waist ?? ""}
                    onChange={handleChange(row.id, "waist")}
                    aria-label={`${row.sizeLabel} ウエスト`}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    {...readonlyProps}
                    type="number"
                    inputMode="decimal"
                    value={row.length ?? ""}
                    onChange={handleChange(row.id, "length")}
                    aria-label={`${row.sizeLabel} 着丈`}
                  />
                </TableCell>
                <TableCell>
                  <Input
                    {...readonlyProps}
                    type="number"
                    inputMode="decimal"
                    value={row.shoulder ?? ""}
                    onChange={handleChange(row.id, "shoulder")}
                    aria-label={`${row.sizeLabel} 肩幅`}
                  />
                </TableCell>

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
                <TableCell colSpan={isEdit ? 6 : 5} className="svc__empty">
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
