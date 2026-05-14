// frontend/console/model/src/presentation/components/ColorVariationCard.tsx

import * as React from "react";
import { Palette, Plus, X } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
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
  mode?: "edit" | "view";
  /** color 名 -> #rrggbb */
  colorRgbMap?: Record<string, string>;
  /** カラーごとの RGB(hex) 更新用 */
  onChangeColorRgb?: (color: string, rgbHex: string) => void;
};

function normalizeColorName(value: unknown): string {
  return String(value ?? "").trim();
}

function normalizeHex(value: unknown): string {
  const raw = String(value ?? "").trim();

  if (!raw) {
    return "#ffffff";
  }

  return raw.startsWith("#") ? raw : `#${raw}`;
}

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

  const safeColors = React.useMemo(
    () =>
      Array.isArray(colors)
        ? colors.map(normalizeColorName).filter(Boolean)
        : [],
    [colors],
  );

  const safeColorRgbMap = colorRgbMap ?? {};

  const [pickerColor, setPickerColor] = React.useState<string>("#ffffff");

  const handleAddColor = React.useCallback(() => {
    const name = normalizeColorName(colorInput);

    if (!name) {
      return;
    }

    if (safeColors.includes(name)) {
      return;
    }

    const hex = normalizeHex(pickerColor);

    /**
     * 先に色名 -> hex を保存してから、親の colors 配列へ追加する。
     * #000000 は falsy 判定に巻き込まないため、文字列としてそのまま扱う。
     */
    onChangeColorRgb?.(name, hex);
    onAddColor();
  }, [
    colorInput,
    pickerColor,
    safeColors,
    onAddColor,
    onChangeColorRgb,
  ]);

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
          {/* 左カラム（edit のみ） */}
          <div className="vc__left">
            {isEdit && (
              <div className="flex flex-col gap-4">
                <div className="vc__picker">
                  <SketchPicker
                    color={pickerColor}
                    onChange={(color: any) => {
                      const hex = normalizeHex(color?.hex);
                      setPickerColor(hex);
                    }}
                  />
                </div>

                <div className="vc__input-wrap flex items-center gap-2">
                  <input
                    className="input vc__input"
                    placeholder="例：White, Black, Navy..."
                    value={colorInput}
                    onChange={(e) => onChangeColorInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key !== "Enter") {
                        return;
                      }

                      e.preventDefault();
                      handleAddColor();
                    }}
                  />

                  <Button
                    type="button"
                    variant="secondary"
                    size="icon"
                    onClick={handleAddColor}
                    aria-label="カラーを追加"
                    className="vc__add"
                    disabled={!normalizeColorName(colorInput)}
                  >
                    <Plus size={18} />
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* 右カラム: 色一覧テーブル */}
          <div className="vc__right">
            <div className="vc__chips">
              {safeColors.length > 0 ? (
                <Table className="vc__table">
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[160px]">RGB(HEX)</TableHead>
                      <TableHead>色名</TableHead>
                      {isEdit && <TableHead className="w-[40px]" />}
                    </TableRow>
                  </TableHeader>

                  <TableBody>
                    {safeColors.map((colorName) => {
                      const hexFromMap = safeColorRgbMap[colorName];
                      const hex = normalizeHex(hexFromMap);

                      return (
                        <TableRow key={colorName}>
                          <TableCell>
                            <div className="flex items-center gap-2">
                              <span
                                className="inline-block w-4 h-4 rounded border"
                                style={{ backgroundColor: hex }}
                              />
                              {hexFromMap ? hex : "-"}
                            </div>
                          </TableCell>

                          <TableCell>{colorName}</TableCell>

                          {isEdit && (
                            <TableCell className="text-right">
                              <button
                                type="button"
                                className="vc__chip-close"
                                onClick={() => onRemoveColor(colorName)}
                                aria-label={`${colorName} を削除`}
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