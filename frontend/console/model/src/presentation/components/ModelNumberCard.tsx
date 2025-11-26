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

/**
 * サイズ行の見た目用の最小情報
 */
type SizeLike = { id: string; sizeLabel: string };

type ModelNumberCardProps = {
  /** 行方向：サイズ一覧 */
  sizes: SizeLike[];

  /** 列方向：カラー名一覧 */
  colors: string[];

  /** 表示用：サイズ×カラーのコード値を取得する関数（ロジックは hook 側） */
  getCode: (sizeLabel: string, color: string) => string;

  className?: string;
  mode?: "edit" | "view";

  /** 変更通知（ロジックは hook 側に委譲） */
  onChangeModelNumber?: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;
};

const ModelNumberCard: React.FC<ModelNumberCardProps> = ({
  sizes,
  colors,
  getCode,
  className,
  mode = "edit",
  onChangeModelNumber,
}) => {
  const isEdit = mode === "edit";

  const handleChange =
    (sizeLabel: string, color: string) =>
    (e: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) return;
      const nextCode = e.target.value;
      if (onChangeModelNumber) {
        onChangeModelNumber(sizeLabel, color, nextCode);
      }
    };

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

                {colors.map((c) => {
                  const code = getCode(s.sizeLabel, c);
                  return (
                    <TableCell key={c}>
                      {isEdit ? (
                        <Input
                          value={code}
                          onChange={handleChange(s.sizeLabel, c)}
                          placeholder="例: LM-SB-S-WHT"
                          aria-label={`${s.sizeLabel} / ${c} のモデルナンバー`}
                        />
                      ) : (
                        <span>{code}</span>
                      )}
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
