// frontend/model/src/pages/ColorVariationCard.tsx
import * as React from "react";
import { Palette, Plus, X } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "../../../../shared/ui";
import { Badge } from "../../../../shared/ui/badge";
import { Button } from "../../../../shared/ui/button";
import "../styles/model.css";
import "../../../shared/ui/card.css";

type ColorVariationCardProps = {
  /** 現在のカラー一覧 */
  colors: string[];
  /** 入力中のカラー名（編集モード時のみ使用） */
  colorInput: string;
  /** 入力変更（編集モード時のみ使用） */
  onChangeColorInput: (v: string) => void;
  /** 追加ボタン or Enter で呼ばれる（編集モード時のみ使用） */
  onAddColor: () => void;
  /** チップの×で呼ばれる（編集モード時のみ使用） */
  onRemoveColor: (color: string) => void;
  /** 表示モード: "edit" | "view"（既定: "edit"） */
  mode?: "edit" | "view";
};

const ColorVariationCard: React.FC<ColorVariationCardProps> = ({
  colors,
  colorInput,
  onChangeColorInput,
  onAddColor,
  onRemoveColor,
  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  return (
    <Card className="vc">
      <CardHeader className="box__header">
        <Palette size={16} />
        <CardTitle className="box__title">
          カラーバリエーション
          {mode === "view" && (
            <span
              className="ml-2 text-xs text-[var(--pbp-text-soft)] align-middle"
              aria-label="閲覧モード"
            >
              （閲覧）
            </span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {/* カラー chips */}
        <div className="chips vc__chips" role="list">
          {colors.map((c) => (
            <span key={c} title={c} role="listitem">
              <Badge
                className="vc__chip inline-flex items-center gap-1.5 px-2 py-1"
                variant="secondary"
              >
                {c}
                {isEdit && (
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
                )}
              </Badge>
            </span>
          ))}

          {colors.length === 0 && (
            <span className="vc__empty">
              まだカラーがありません。
              {isEdit ? " 右で追加してください。" : "（データがありません）"}
            </span>
          )}
        </div>

        {/* 入力 + 追加（編集モードのみ表示） */}
        {isEdit && (
          <div className="vc__row">
            <input
              className="input vc__input"
              placeholder="カラーを入力（例：ホワイト、ブラック など）"
              value={colorInput}
              onChange={(e) => onChangeColorInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") onAddColor();
              }}
              aria-label="カラー名を入力"
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
        )}
      </CardContent>
    </Card>
  );
};

export default ColorVariationCard;
