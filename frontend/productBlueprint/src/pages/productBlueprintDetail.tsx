import * as React from "react";
import { Plus, X, Palette, Trash2, Tags } from "lucide-react";
import PageHeader from "./PageHeader";
import AdminCard from "./AdminCard";
import ProductBlueprintCard from "./productBlueprintCard";
import "./productBlueprintDetail.css";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

type SizeRow = {
  id: string;
  sizeLabel: string;
  chest?: number;
  waist?: number;
  length?: number;
  shoulder?: number;
};

export default function ProductBlueprintDetailPage() {
  const [productName, setProductName] = React.useState("シルクブラウス プレミアムライン");
  const [brand] = React.useState("LUMINA Fashion");
  const [fit, setFit] = React.useState<Fit>("レギュラーフィット");
  const [materials, setMaterials] = React.useState("シルク100%、裏地:ポリエステル100%");
  const [weight, setWeight] = React.useState<number>(180);
  const [washTags, setWashTags] = React.useState<string[]>(["手洗い", "ドライクリーニング", "陰干し"]);
  const [productIdTag, setProductIdTag] = React.useState("QRコード");
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>(["ホワイト", "ブラック", "ネイビー"]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([
    { id: "1", sizeLabel: "S", chest: 48, waist: 58, length: 60, shoulder: 38 },
    { id: "2", sizeLabel: "M", chest: 50, waist: 60, length: 62, shoulder: 40 },
    { id: "3", sizeLabel: "L", chest: 52, waist: 62, length: 64, shoulder: 42 },
  ]);

  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("佐藤 美咲");
  const [createdAt] = React.useState("2024/1/15");

  const onSave = () => alert("保存しました（ダミー）");

  const addColor = () => {
    if (!colorInput.trim()) return;
    setColors((prev) => [...prev, colorInput.trim()]);
    setColorInput("");
  };
  const removeColor = (name: string) => setColors((prev) => prev.filter((c) => c !== name));

  return (
    <div className="pbp">
      <PageHeader title="商品設計詳細" onSave={onSave} />

      <div className="grid-2">
        {/* 左ペイン */}
        <div>
          <ProductBlueprintCard
            productName={productName}
            brand={brand}
            fit={fit}
            materials={materials}
            weight={weight}
            washTags={washTags}
            productIdTag={productIdTag}
            onChangeProductName={setProductName}
            onChangeFit={setFit}
            onChangeMaterials={setMaterials}
            onChangeWeight={setWeight}
            onChangeWashTags={setWashTags}
            onChangeProductIdTag={setProductIdTag}
          />

          <section className="box">
            <header className="box__header">
              <Palette size={16} /> <h2 className="box__title">カラーバリエーション</h2>
            </header>
            <div className="box__body">
              <div className="chips">
                {colors.map((c) => (
                  <span key={c} className="chip">
                    {c}
                    <button onClick={() => removeColor(c)}>
                      <X size={14} />
                    </button>
                  </span>
                ))}
              </div>
              <div className="flex gap-8">
                <input
                  className="input"
                  placeholder="カラーを入力"
                  value={colorInput}
                  onChange={(e) => setColorInput(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && addColor()}
                />
                <button className="btn btn--icon" onClick={addColor}>
                  <Plus size={18} />
                </button>
              </div>
            </div>
          </section>

          <section className="box">
            <header className="box__header">
              <Tags size={16} /> <h2 className="box__title">サイズバリエーション</h2>
            </header>
            <div className="box__body">
              <table>
                <thead>
                  <tr>
                    <th>サイズ</th>
                    <th>胸囲(cm)</th>
                    <th>ウエスト(cm)</th>
                    <th>着丈(cm)</th>
                    <th>肩幅(cm)</th>
                  </tr>
                </thead>
                <tbody>
                  {sizes.map((row) => (
                    <tr key={row.id}>
                      <td><input className="readonly" value={row.sizeLabel} readOnly /></td>
                      <td><input className="readonly" value={row.chest ?? ""} readOnly /></td>
                      <td><input className="readonly" value={row.waist ?? ""} readOnly /></td>
                      <td><input className="readonly" value={row.length ?? ""} readOnly /></td>
                      <td><input className="readonly" value={row.shoulder ?? ""} readOnly /></td>
                      <td>
                        <button className="btn btn--icon" onClick={() => setSizes(sizes.filter((s) => s.id !== row.id))}>
                          <Trash2 size={16} />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        </div>

        {/* 右ペイン */}
        <AdminCard
          assignee={assignee}
          creator={creator}
          createdAt={createdAt}
          onEditAssignee={(next) => setAssignee(next)}
        />
      </div>
    </div>
  );
}
