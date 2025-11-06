// frontend/model/src/pages/ModelNumberCard.tsx
import * as React from "react";
import { Tags } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui"; // ✅ Card系コンポーネント導入
import { Input } from "../../../shared/ui/input"; // ✅ Input導入
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
} from "../../../shared/ui/table"; // ✅ Table導入
import "./ModelNumberCard.css";

export type ModelNumber = {
  size: string; // 例: "S" | "M" | "L"
  color: string; // 例: "ホワイト" | "ブラック" | ...
  code: string; // 例: "LM-SB-S-WHT"
};

type SizeLike = { id: string; sizeLabel: string };

type ModelNumberCardProps = {
  sizes: SizeLike[];
  colors: string[];
  modelNumbers: ModelNumber[];
  className?: string;
};

const ModelNumberCard: React.FC<ModelNumberCardProps> = ({
  sizes,
  colors,
  modelNumbers,
  className,
}) => {
  const findCode = (sizeLabel: string, color: string) =>
    modelNumbers.find((m) => m.size === sizeLabel && m.color === color)?.code ??
    "";

  return (
    <Card className={`mnc ${className ?? ""}`}>
      <CardHeader className="box__header">
        <Tags size={16} />
        <CardTitle className="box__title">モデルナンバー</CardTitle>
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
                    <Input value={findCode(s.sizeLabel, c)} variant="readonly" />
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
