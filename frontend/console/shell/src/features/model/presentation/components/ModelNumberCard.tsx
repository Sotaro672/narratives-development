// frontend/console/model/src/presentation/components/ModelNumberCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderLeft,
  CardInput,
  CardReadonly,
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

/**
 * サイズ行の見た目用の最小情報
 */
type SizeLike = {
  id: string;
  sizeLabel: string;
};

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
    (event: React.ChangeEvent<HTMLInputElement>) => {
      if (!isEdit) {
        return;
      }

      const nextCode = event.target.value;
      onChangeModelNumber?.(sizeLabel, color, nextCode);
    };

  return (
    <Card
      className={`mnc ${mode === "view" ? "view-mode" : ""} ${
        className ?? ""
      }`}
    >
      <CardHeader>
        <CardHeaderLeft>
          <CardTitle strong>
            モデルナンバー
          </CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
        <Table className="mnc__table">
          <TableHeader>
            <TableRow>
              <TableHead>
                サイズ / カラー
              </TableHead>

              {colors.map((color) => (
                <TableHead key={color}>
                  {color}
                </TableHead>
              ))}
            </TableRow>
          </TableHeader>

          <TableBody>
            {sizes.map((size) => (
              <TableRow key={size.id}>
                <TableCell className="mnc__size">
                  {size.sizeLabel}
                </TableCell>

                {colors.map((color) => {
                  const code = getCode(size.sizeLabel, color);

                  return (
                    <TableCell key={color}>
                      {isEdit ? (
                        <CardInput
                          value={code}
                          onChange={handleChange(
                            size.sizeLabel,
                            color,
                          )}
                          placeholder="例: LM-SB-S-WHT"
                          aria-label={`${size.sizeLabel} / ${color} のモデルナンバー`}
                        />
                      ) : (
                        <CardReadonly inputLike>
                          {code || "-"}
                        </CardReadonly>
                      )}
                    </TableCell>
                  );
                })}
              </TableRow>
            ))}

            {sizes.length === 0 && (
              <TableRow>
                <TableCell
                  colSpan={Math.max(
                    1,
                    colors.length + 1,
                  )}
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