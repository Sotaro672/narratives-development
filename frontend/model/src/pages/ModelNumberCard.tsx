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
import "../../../shared/ui/card.css";

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
   * モデルナンバーの変更コールバック（編集モード時）
   * 指定されている場合は、入力変更時に呼び出される
   */
  onChangeCode?: (sizeLabel: string, color: string, nextCode: string) => void;
};

const makeKey = (sizeLabel: string, color: string) => `${sizeLabel}__${color}`;

const ModelNumberCard: React.FC<ModelNumberCardProps> = ({
  sizes,
  colors,
  modelNumbers,
  className,
  mode = "edit",
  onChangeCode,
}) => {
  const isEdit = mode === "edit";

  // props から初期マップを作成し、編集モードではローカル state を編集可能にする
  const [codeMap, setCodeMap] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    const next: Record<string, string> = {};
    sizes.forEach((s) => {
      colors.forEach((c) => {
        const found =
          modelNumbers.find(
            (m) => m.size === s.sizeLabel && m.color === c,
          )?.code ?? "";
        const key = makeKey(s.sizeLabel, c);
        next[key] = found;
      });
    });
    setCodeMap(next);
  }, [sizes, colors, modelNumbers]);

  const findCodeFromProps = (sizeLabel: string, color: string) =>
    modelNumbers.find((m) => m.size === sizeLabel && m.color === color)?.code ??
    "";

  const getValue = (sizeLabel: string, color: string) => {
    const key = makeKey(sizeLabel, color);
    return isEdit ? codeMap[key] ?? "" : findCodeFromProps(sizeLabel, color);
  };

  const handleChange =
    (sizeLabel: string, color: string) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) return;

      const nextCode = e.target.value;
      const key = makeKey(sizeLabel, color);

      setCodeMap((prev) => ({
        ...prev,
        [key]: nextCode,
      }));

      if (onChangeCode) {
        onChangeCode(sizeLabel, color, nextCode);
      }
    };

  const readonlyProps = { variant: "readonly" as const, readOnly: true };

  return (
    <Card
      className={`mnc ${className ?? ""} ${
        mode === "view" ? "view-mode" : ""
      }`}
    >
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">
          モデルナンバー
          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">
              （閲覧）
            </span>
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
                {colors.map((c) => (
                  <TableCell key={c}>
                    <Input
                      {...(!isEdit ? readonlyProps : {})}
                      value={getValue(s.sizeLabel, c)}
                      onChange={handleChange(s.sizeLabel, c)}
                      placeholder="例: LM-SB-S-WHT"
                      aria-label={`${s.sizeLabel} / ${c} のモデルナンバー`}
                    />
                  </TableCell>
                ))}
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
