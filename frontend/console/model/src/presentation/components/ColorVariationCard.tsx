// frontend/model/src/pages/ColorVariationCard.tsx
import * as React from "react";
import { Palette, Plus, X } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import { Button } from "../../../../shell/src/shared/ui/button";
import "../styles/model.css";
import "../../../../shell/src/shared/ui/card.css";

import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "../../../../shell/src/shared/ui/table";

import { SketchPicker } from "react-color";

type ColorVariationCardProps = {
  colors: string[];
  colorInput: string;
  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (color: string) => void;
  /** 表示モード（既定: "edit"） */
  mode?: "edit" | "view";

  /** color 名 → hex(RGB) のマップ（例: { "グリーン": "#00ff00" }） */
  colorRgbMap?: Record<string, string>;
  /**
   * 色名ごとの RGB 変更通知
   * - カラー追加ボタン押下時に `(colorInput, pickerColor)` で呼ばれる想定
   */
  onChangeColorRgb?: (color: string, rgbHex: string) => void;
};

const ColorVariationCard: React.FC<ColorVariationCardProps> = ({
  colors,
  colorInput,
  onChangeColorInput,
  onAddColor,
  onRemoveColor,
  mode = "edit",
  colorRgbMap,
  onChangeColorRgb,
}) => {
  const isEdit = mode === "edit";

  const [pickerColor, setPickerColor] = React.useState<string>(
    colorInput || "#ffffff",
  );

  // pickerColor の変化をログ出力（デバッグ用）
  React.useEffect(() => {
    console.log("[ColorVariationCard] pickerColor changed:", pickerColor);
  }, [pickerColor]);

  // 「カラーを追加」ボタン押下時のラッパー
  // - 先に color 名 → pickerColor を通知
  // - その後本来の onAddColor を呼び出す
  const handleAddColor = React.useCallback(() => {
    const name = colorInput.trim();

    console.log("[ColorVariationCard] handleAddColor called", {
      colorInput: name,
      pickerColor,
    });

    if (name) {
      onChangeColorRgb?.(name, pickerColor);
    }
    onAddColor();
  }, [colorInput, pickerColor, onAddColor, onChangeColorRgb]);

  return (
    <Card className="vc">
      <CardHeader className="box__header">
        <Palette size={16} />
        <CardTitle className="box__title">
          カラーバリエーション
          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)] align-middle">
              （閲覧）
            </span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {/* カード内を左右 2 カラムに分割 */}
        <div
          className="vc__layout"
          style={{
            display: "grid",
            gridTemplateColumns: isEdit
              ? "minmax(0, 1.2fr) minmax(0, 2fr)"
              : "minmax(0, 1fr)",
            gap: 16,
          }}
        >
          {/* 左カラム: カラーピッカー + 色名入力欄 + 追加ボタン */}
          <div className="vc__left">
            {isEdit && (
              <div className="flex flex-col gap-4">
                <div className="vc__picker">
                  <SketchPicker
                    color={pickerColor}
                    onChange={(color: any) => {
                      const hex = color?.hex ?? "#ffffff";
                      console.log(
                        "[ColorVariationCard] SketchPicker onChange",
                        {
                          raw: color,
                          hex,
                        },
                      );
                      setPickerColor(hex);
                    }}
                  />
                </div>

                <div className="vc__input-wrap flex items-center gap-2">
                  <input
                    className="input vc__input"
                    placeholder="例：White, Black, Navy..."
                    value={colorInput}
                    onChange={(e) => {
                      onChangeColorInput(e.target.value);
                    }}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") {
                        handleAddColor();
                      }
                    }}
                  />
                  <Button
                    variant="secondary"
                    size="icon"
                    onClick={handleAddColor}
                    aria-label="カラーを追加"
                    className="vc__add"
                  >
                    <Plus size={18} />
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* 右カラム: RGB / 色名 テーブル */}
          <div className="vc__right">
            <div className="vc__chips">
              {colors.length > 0 ? (
                <Table className="vc__table">
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[160px]">RGB(HEX)</TableHead>
                      <TableHead>色名</TableHead>
                      {isEdit && <TableHead className="w-[40px]" />}
                    </TableRow>
                  </TableHeader>

                  <TableBody>
                    {colors.map((c) => {
                      // 1行ごとの表示用 HEX
                      const hex = colorRgbMap?.[c] ?? pickerColor;

                      return (
                        <TableRow key={c}>
                          {/* RGB列（色プレビュー + HEX値） */}
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span
                                className="inline-block w-4 h-4 rounded border"
                                style={{ backgroundColor: hex }}
                              />
                              {hex}
                            </div>
                          </TableCell>

                          {/* 色名列 */}
                          <TableCell>
                            <Badge
                              className="vc__chip inline-flex items-center gap-1.5 px-2 py-1"
                              variant="secondary"
                            >
                              {c}
                            </Badge>
                          </TableCell>

                          {/* 削除ボタン（edit のみ） */}
                          {isEdit && (
                            <TableCell className="text-right">
                              <button
                                className="vc__chip-close"
                                onClick={() => onRemoveColor(c)}
                                aria-label={`${c} を削除`}
                                style={{
                                  background: "transparent",
                                  border: "none",
                                  cursor: "pointer",
                                  display: "inline-flex",
                                  alignItems: "center",
                                  padding: 0,
                                }}
                              >
                                <X size={12} />
                              </button>
                            </TableCell>
                          )}
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              ) : (
                <span className="vc__empty">
                  まだカラーがありません。
                  {isEdit ? " 左で色を選んで追加してください。" : "（データなし）"}
                </span>
              )}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
};

export default ColorVariationCard;
