// frontend/model/src/pages/ColorVariationCard.tsx
import * as React from "react";
import { Palette, Plus, X } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui";
import { Badge } from "../../../shared/ui/badge";
import { Button } from "../../../shared/ui/button";
import "./ColorVariationCard.css";

type ColorVariationCardProps = {
  /** 現在のカラー一覧 */
  colors: string[];
  /** 入力中のカラー名 */
  colorInput: string;
  /** 入力変更 */
  onChangeColorInput: (v: string) => void;
  /** 追加ボタン or Enter で呼ばれる */
  onAddColor: () => void;
  /** チップの×で呼ばれる */
  onRemoveColor: (color: string) => void;
};

const ColorVariationCard: React.FC<ColorVariationCardProps> = ({
  colors,
  colorInput,
  onChangeColorInput,
  onAddColor,
  onRemoveColor,
}) => {
  return (
    <Card className="vc box">
      <CardHeader className="box__header">
        <Palette size={16} />
        <CardTitle className="box__title">カラーバリエーション</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {/* カラー chips */}
        <div className="chips vc__chips">
          {colors.map((c) => (
            <span key={c} title={c}>
              <Badge
                className="vc__chip inline-flex items-center gap-1.5 px-2 py-1"
                variant="secondary"
              >
                {c}
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
              </Badge>
            </span>
          ))}

          {colors.length === 0 && (
            <span className="vc__empty">
              まだカラーがありません。右で追加してください。
            </span>
          )}
        </div>

        {/* 入力 + 追加 */}
        <div className="vc__row">
          <input
            className="input vc__input"
            placeholder="カラーを入力（例：ホワイト、ブラック など）"
            value={colorInput}
            onChange={(e) => onChangeColorInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") onAddColor();
            }}
          />
          <Button
            variant="secondary"
            size="icon"
            onClick={onAddColor}
            aria-label="カラーを追加"
            className="vc__add"
          >
            <Plus size={18} />
          </Button>
        </div>
      </CardContent>
    </Card>
  );
};

export default ColorVariationCard;
