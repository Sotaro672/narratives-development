// frontend/console/model/src/presentation/components/ColorVariationCard.tsx

import * as React from "react";
import { Plus, X } from "lucide-react";
import { SketchPicker } from "react-color";

import {
  Card,
  CardContent,
  CardHeader,
  CardHeaderLeft,
  CardInput,
  CardTitle,
} from "../../../../shared/ui";
import { Button } from "../../../../shared/ui/button";
import { ColorSwatch } from "../../../../shared/ui/color";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";
import {
  coerceRgbInt,
  rgbIntToHex,
} from "../../../../shared/util/color";

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
  return rgbIntToHex(coerceRgbInt(value)) ?? "#FFFFFF";
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
  const [pickerColor, setPickerColor] = React.useState<string>("#FFFFFF");

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
      <CardHeader>
        <CardHeaderLeft>
          <CardTitle strong>カラーバリエーション</CardTitle>
        </CardHeaderLeft>
      </CardHeader>

      <CardContent>
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
                  <CardInput
                    className="vc__input"
                    placeholder="例：White, Black, Navy..."
                    value={colorInput}
                    onChange={(event) => onChangeColorInput(event.target.value)}
                    onKeyDown={(event) => {
                      if (event.key !== "Enter") {
                        return;
                      }

                      event.preventDefault();
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
                              <ColorSwatch
                                color={hex}
                                size="md"
                                shape="square"
                              />
                              {hexFromMap ? hex : "-"}
                            </div>
                          </TableCell>

                          <TableCell>{colorName}</TableCell>

                          {isEdit && (
                            <TableCell className="text-right">
                              <Button
                                type="button"
                                variant="ghost"
                                size="icon"
                                className="vc__chip-close"
                                onClick={() => onRemoveColor(colorName)}
                                aria-label={`${colorName} を削除`}
                              >
                                <X size={12} />
                              </Button>
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
                  {isEdit
                    ? " 左で色を選んで追加してください。"
                    : "（データなし）"}
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