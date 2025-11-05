//frontend\model\src\pages\ColorVariationCard.tsx
import * as React from "react";
import { Palette, Plus, X } from "lucide-react";
import "./colorVariationCard.css";

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
    <section className="vc box">
      <header className="box__header">
        <Palette size={16} /> <h2 className="box__title">カラーバリエーション</h2>
      </header>

      <div className="box__body">
        {/* カラー chips */}
        <div className="chips vc__chips">
          {colors.map((c) => (
            <span key={c} className="chip vc__chip" title={c}>
              {c}
              <button
                className="vc__chip-close"
                onClick={() => onRemoveColor(c)}
                aria-label={`${c} を削除`}
              >
                <X size={14} />
              </button>
            </span>
          ))}
          {colors.length === 0 && (
            <span className="vc__empty">まだカラーがありません。右で追加してください。</span>
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
          <button className="btn btn--icon vc__add" onClick={onAddColor} aria-label="カラーを追加">
            <Plus size={18} />
          </button>
        </div>
      </div>
    </section>
  );
};

export default ColorVariationCard;
