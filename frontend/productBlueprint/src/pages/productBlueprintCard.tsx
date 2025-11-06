//frontend/productBlueprint/src/pages/productBlueprintCard.tsx
import * as React from "react";
import { ShieldCheck, X, Package2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shared/ui";
import { Badge } from "../../../shared/ui/badge";
import { Button } from "../../../shared/ui/button";
import { Input } from "../../../shared/ui/input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../shared/ui/popover";
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
  /** 表示モード（既定: "edit"） */
  mode?: "edit" | "view";
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
  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const fitOptions = [
    { value: "レギュラーフィット", label: "レギュラーフィット" },
    { value: "スリムフィット", label: "スリムフィット" },
    { value: "リラックスフィット", label: "リラックスフィット" },
    { value: "オーバーサイズ", label: "オーバーサイズ" },
  ];

  const tagOptions = [
    { value: "QRコード", label: "QRコード" },
    { value: "バーコード", label: "バーコード" },
  ];

  return (
    <Card className="pbc">
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">
          基本情報
          {mode === "view" && (
            <span className="ml-2 text-xs text-[var(--pbp-text-soft)]">（閲覧）</span>
          )}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {/* プロダクト名 */}
        <div className="label">プロダクト名</div>
        {isEdit ? (
          <Input
            value={productName}
            onChange={(e) => onChangeProductName(e.target.value)}
            aria-label="プロダクト名"
          />
        ) : (
          <Input value={productName} variant="readonly" readOnly aria-label="プロダクト名" />
        )}

        {/* ブランド（常に読み取り専用） */}
        <div className="label">ブランド</div>
        <Input value={brand} variant="readonly" readOnly aria-label="ブランド" />

        {/* フィット */}
        <div className="label">フィット</div>
        {isEdit ? (
          <Popover>
            <PopoverTrigger>
              <Button variant="outline" className="w-full justify-between" aria-label="フィットを選択">
                {fit || "選択してください"}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="p-1">
              {fitOptions.map((opt) => (
                <div
                  key={opt.value}
                  className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                    fit === opt.value ? "bg-blue-100 text-blue-700 font-medium" : ""
                  }`}
                  onClick={() => onChangeFit(opt.value as Fit)}
                >
                  {opt.label}
                </div>
              ))}
            </PopoverContent>
          </Popover>
        ) : (
          <Input value={fit} variant="readonly" readOnly aria-label="フィット" />
        )}

        {/* 素材 */}
        <div className="label">素材</div>
        {isEdit ? (
          <Input
            value={materials}
            onChange={(e) => onChangeMaterials(e.target.value)}
            aria-label="素材"
          />
        ) : (
          <Input value={materials} variant="readonly" readOnly aria-label="素材" />
        )}

        {/* 重さ */}
        <div className="label">重さ</div>
        <div className="flex gap-8 items-center">
          {isEdit ? (
            <>
              <Input
                type="number"
                value={Number.isFinite(weight) ? weight : 0}
                onChange={(e) => onChangeWeight(Number(e.target.value))}
                aria-label="重さ"
              />
              <span className="suffix">g</span>
            </>
          ) : (
            <>
              <Input
                value={`${weight ?? ""}`}
                variant="readonly"
                readOnly
                aria-label="重さ"
              />
              <span className="suffix">g</span>
            </>
          )}
        </div>

        {/* 品質保証タグ（洗濯方法タグ） */}
        <div className="label">品質保証（洗濯方法タグ）</div>
        <div className="chips flex flex-wrap gap-2">
          {washTags.map((t) => (
            <Badge
              key={t}
              className="chip inline-flex items-center gap-1.5 px-2 py-1"
            >
              <ShieldCheck size={14} />
              {t}
              {isEdit && (
                <button
                  onClick={() =>
                    onChangeWashTags(washTags.filter((x) => x !== t))
                  }
                  style={{
                    background: "transparent",
                    border: "none",
                    cursor: "pointer",
                    display: "inline-flex",
                    alignItems: "center",
                    padding: 0,
                  }}
                  aria-label={`${t} を削除`}
                >
                  <X size={12} />
                </button>
              )}
            </Badge>
          ))}

          {isEdit && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => onChangeWashTags([...washTags, "新タグ"])}
              className="btn"
            >
              + 追加
            </Button>
          )}
        </div>

        {/* 商品IDタグ */}
        <div className="label">商品IDタグ</div>
        {isEdit ? (
          <Popover>
            <PopoverTrigger>
              <Button variant="outline" className="w-full justify-between" aria-label="商品IDタグを選択">
                {productIdTag || "選択してください"}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="p-1">
              {tagOptions.map((opt) => (
                <div
                  key={opt.value}
                  className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                    productIdTag === opt.value ? "bg-blue-100 text-blue-700 font-medium" : ""
                  }`}
                  onClick={() => onChangeProductIdTag(opt.value)}
                >
                  {opt.label}
                </div>
              ))}
            </PopoverContent>
          </Popover>
        ) : (
          <Input value={productIdTag} variant="readonly" readOnly aria-label="商品IDタグ" />
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;
