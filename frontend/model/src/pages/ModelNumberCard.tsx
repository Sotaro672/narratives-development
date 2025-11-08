// frontend/model/src/pages/ModelNumberCard.tsx
import * as React from "react";
import { Tags } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui";
import { Input } from "../../../shared/ui/input";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from "../../../shared/ui/table";
import "./ModelNumberCard.css";

export type ModelNumber = {
  size: string;  // 例: "S" | "M" | "L"
  color: string; // 例: "ホワイト" | "ブラック" | ...
  code: string;  // 例: "LM-SB-S-WHT"
};

type SizeLike = { id: string; sizeLabel: string };

type ModelNumberCardProps = {
  sizes: SizeLike[];
  colors: string[];
  modelNumbers: ModelNumber[];
  className?: string;
  /** "edit" | "view"（既定: "edit"） */
  mode?: "edit" | "view";
  /**
   * モデルナンバーの変更コールバック（編集モード時のみ有効）
   * 未指定の場合、編集モードでも入力は読み取り専用のまま
   */
  onChangeCode?: (sizeLabel: string, color: string, nextCode: string) => void;
};

const ModelNumberCard: React.FC<ModelNumberCardProps> = ({
  sizes,
  colors,
  modelNumbers,
  className,
  mode = "edit",
  onChangeCode,
}) => {
  const isEdit = mode === "edit";
  const canEdit = isEdit && typeof onChangeCode === "function";

  const findCode = (sizeLabel: string, color: string) =>
    modelNumbers.find((m) => m.size === sizeLabel && m.color === color)?.code ?? "";

  const handleChange =
    (sizeLabel: string, color: string) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!canEdit) return;
      onChangeCode!(sizeLabel, color, e.target.value);
    };

  const readonlyProps = { variant: "readonly" as const, readOnly: true };

  return (
    <Card className={`mnc ${className ?? ""}`}>
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">
          モデルナンバー
          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">（閲覧）</span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <Table className="mnc__table">
          <TableHeader>
            <TableRow>
              <TableHead>サイズ / カラー</TableHead>
              {colors.map((color) => (
                <TableHead key={color}>{color}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {sizes.map((s) => (
              <TableRow key={s.id}>
                <TableCell className="mnc__size">{s.sizeLabel}</TableCell>
                {colors.map((c) => {
                  const value = findCode(s.sizeLabel, c);
                  return (
                    <TableCell key={c}>
                      <Input
                        {...(canEdit ? {} : readonlyProps)}
                        value={value}
                        onChange={handleChange(s.sizeLabel, c)}
                        placeholder="例: LM-SB-S-WHT"
                        aria-label={`${s.sizeLabel} / ${c} のモデルナンバー`}
                      />
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}

            {sizes.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={Math.max(1, colors.length + 1)}
                  className="mnc__empty"
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

export default ModelNumberCard;
