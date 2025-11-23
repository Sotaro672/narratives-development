// frontend/console/model/src/presentation/components/ModelNumberCard.tsx

import * as React from "react";
import { Tags } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
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

export type ModelNumber = {
  size: string;  // 例: "S" | "M" | "L"
  color: string; // 例: "ホワイト" | "ブラック"
  code: string;  // 例: "LM-SB-S-WHT"
};

type SizeLike = { id: string; sizeLabel: string };

type ModelNumberCardProps = {
  sizes: SizeLike[];
  colors: string[];
  modelNumbers: ModelNumber[];
  className?: string;
  mode?: "edit" | "view";

  /** hook に変更通知する */
  onChangeModelNumber?: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;
};

const makeKey = (sizeLabel: string, color: string) =>
  `${sizeLabel}__${color}`;

const ModelNumberCard: React.FC<ModelNumberCardProps> = ({
  sizes,
  colors,
  modelNumbers,
  className,
  mode = "edit",
  onChangeModelNumber,
}) => {
  const isEdit = mode === "edit";

  // 内部編集用の state
  const [codeMap, setCodeMap] = React.useState<Record<string, string>>({});

  // props → state 反映
  React.useEffect(() => {
    const next: Record<string, string> = {};

    sizes.forEach((s) => {
      colors.forEach((c) => {
        const found =
          modelNumbers.find(
            (m) => m.size === s.sizeLabel && m.color === c,
          )?.code ?? "";
        next[makeKey(s.sizeLabel, c)] = found;
      });
    });

    setCodeMap(next);
  }, [sizes, colors, modelNumbers]);

  const getValue = (sizeLabel: string, color: string) => {
    const key = makeKey(sizeLabel, color);
    return codeMap[key] ?? "";
  };

  const handleChange =
    (sizeLabel: string, color: string) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) return;

      const nextCode = e.target.value;
      const key = makeKey(sizeLabel, color);

      // ローカル state 更新
      setCodeMap((prev) => ({
        ...prev,
        [key]: nextCode,
      }));

      // hook に伝える
      if (onChangeModelNumber) {
        onChangeModelNumber(sizeLabel, color, nextCode);
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
              {colors.map((c) => (
                <TableHead key={c}>{c}</TableHead>
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
