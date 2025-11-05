import * as React from "react";
import { Tags } from "lucide-react";
import "./ModelNumberCard.css";

export type ModelNumber = {
  size: string;   // 例: "S" | "M" | "L"
  color: string;  // 例: "ホワイト" | "ブラック" | ...
  code: string;   // 例: "LM-SB-S-WHT"
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
    modelNumbers.find((m) => m.size === sizeLabel && m.color === color)?.code ?? "";

  return (
    <section className={`mnc box ${className ?? ""}`}>
      <header className="box__header">
        <Tags size={16} /> <h2 className="box__title">モデルナンバー</h2>
      </header>

      <div className="box__body">
        <table className="mnc__table">
          <thead>
            <tr>
              <th>サイズ / カラー</th>
              {colors.map((color) => (
                <th key={color}>{color}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {sizes.map((s) => (
              <tr key={s.id}>
                <td className="mnc__size">{s.sizeLabel}</td>
                {colors.map((c) => (
                  <td key={c}>
                    <input className="readonly" value={findCode(s.sizeLabel, c)} readOnly />
                  </td>
                ))}
              </tr>
            ))}

            {sizes.length === 0 && (
              <tr>
                <td colSpan={Math.max(1, colors.length + 1)} className="mnc__empty">
                  登録されているサイズはありません。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default ModelNumberCard;
