import * as React from "react";
import { ShieldCheck, X, Package2 } from "lucide-react";
import "./productBlueprintCard.css";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

type ProductBlueprintCardProps = {
  productName: string;
  brand: string;
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: string;
  onChangeProductName: (v: string) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterials: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeWashTags: (nextTags: string[]) => void;
  onChangeProductIdTag: (v: string) => void;
};

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productName,
  brand,
  fit,
  materials,
  weight,
  washTags,
  productIdTag,
  onChangeProductName,
  onChangeFit,
  onChangeMaterials,
  onChangeWeight,
  onChangeWashTags,
  onChangeProductIdTag,
}) => {
  return (
    <section className="pbc box">
      <header className="box__header">
        <Package2 size={16} /> <h2 className="box__title">基本情報</h2>
      </header>

      <div className="box__body">
        <div className="label">プロダクト名</div>
        <input
          className="input"
          value={productName}
          onChange={(e) => onChangeProductName(e.target.value)}
        />

        <div className="label">ブランド</div>
        <input className="readonly" value={brand} readOnly />

        <div className="label">フィット</div>
        <select
          className="select"
          value={fit}
          onChange={(e) => onChangeFit(e.target.value as Fit)}
        >
          <option>レギュラーフィット</option>
          <option>スリムフィット</option>
          <option>リラックスフィット</option>
          <option>オーバーサイズ</option>
        </select>

        <div className="label">素材</div>
        <input
          className="input"
          value={materials}
          onChange={(e) => onChangeMaterials(e.target.value)}
        />

        <div className="label">重さ</div>
        <div className="flex gap-8">
          <input
            className="input"
            type="number"
            value={weight}
            onChange={(e) => onChangeWeight(Number(e.target.value))}
          />
          <span className="suffix">g</span>
        </div>

        <div className="label">品質保証（洗濯方法タグ）</div>
        <div className="chips">
          {washTags.map((t) => (
            <span key={t} className="chip">
              <ShieldCheck size={14} />
              {t}
              <button
                onClick={() =>
                  onChangeWashTags(washTags.filter((x) => x !== t))
                }
              >
                <X size={14} />
              </button>
            </span>
          ))}
          <button
            className="btn"
            onClick={() => onChangeWashTags([...washTags, "新タグ"])}
          >
            + 追加
          </button>
        </div>

        <div className="label">商品IDタグ</div>
        <select
          className="select"
          value={productIdTag}
          onChange={(e) => onChangeProductIdTag(e.target.value)}
        >
          <option>QRコード</option>
          <option>バーコード</option>
        </select>
      </div>
    </section>
  );
};

export default ProductBlueprintCard;
