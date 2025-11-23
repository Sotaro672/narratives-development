// frontend/console/productBlueprint/src/presentation/components/productBlueprintCard.tsx

import * as React from "react";
import { ShieldCheck, X, Package2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  type Fit,
} from "../hook/useProductBlueprintDetail";
import "../styles/productBlueprint.css";

type BrandOption = {
  id: string;
  name: string;
};

type ProductBlueprintCardProps = {
  productName?: string;
  /** 選択中ブランドの「表示名」（閲覧モードなどで使用） */
  brand?: string;

  /** ▼ ブランド選択欄用 props（編集モード時） */
  brandId?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
  onChangeProductName?: (v: string) => void;
  onChangeFit?: (v: Fit) => void;
  onChangeMaterials?: (v: string) => void;
  onChangeWeight?: (v: number) => void;
  onChangeWashTags?: (nextTags: string[]) => void;
  onChangeProductIdTag?: (v: string) => void;
  /** 表示モード（既定: "edit"） */
  mode?: "edit" | "view";
};

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productName,
  brand,
  brandId,
  brandOptions,
  brandLoading,
  brandError,
  onChangeBrandId,
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

  // サニタイズ（UI 用の安全値に整形）
  const safeProductName = productName ?? "";
  const safeBrand = brand ?? "";
  const safeMaterials = materials ?? "";
  const safeWeight =
    typeof weight === "number" && !Number.isNaN(weight) ? weight : 0;
  const safeWashTags = Array.isArray(washTags) ? washTags : [];
  const safeProductIdTag = productIdTag ?? "";
  const safeFit = fit ?? ("" as Fit);

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">基本情報</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {/* プロダクト名 */}
        <div className="label">プロダクト名</div>
        {isEdit ? (
          <Input
            value={safeProductName}
            onChange={(e) => onChangeProductName?.(e.target.value)}
            aria-label="プロダクト名"
          />
        ) : (
          <Input
            value={safeProductName}
            variant="readonly"
            readOnly
            aria-label="プロダクト名"
          />
        )}

        {/* ブランド（ログ付きの選択欄に置き換え） */}
        <div className="label">ブランド</div>
        {isEdit && brandOptions && onChangeBrandId ? (
          <div className="mb-2 space-y-1">
            <select
              className="w-full border rounded px-2 py-1 text-sm"
              value={brandId ?? ""}
              onChange={(e) => {
                const next = e.target.value;
                console.log(
                  "[ProductBlueprintCard] brand <select> onChange",
                  next,
                );
                onChangeBrandId(next);
              }}
              aria-label="ブランドを選択"
            >
              <option value="">選択してください</option>
              {brandOptions.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>

            {brandLoading && (
              <p className="text-xs text-slate-400">ブランドを取得中…</p>
            )}
            {brandError && (
              <p className="text-xs text-red-500">
                ブランド一覧の取得に失敗しました。
              </p>
            )}
          </div>
        ) : (
          // ブランド選択情報が無い場合や閲覧モード時は従来どおり読み取り専用表示
          <Input
            value={safeBrand}
            variant="readonly"
            readOnly
            aria-label="ブランド"
          />
        )}

        {/* フィット */}
        <div className="label">フィット</div>
        {isEdit ? (
          <Popover>
            <PopoverTrigger>
              <Button
                variant="outline"
                className="w-full justify-between pbc-select-trigger"
                aria-label="フィットを選択"
              >
                {safeFit || "選択してください"}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="p-1">
              {FIT_OPTIONS.map((opt) => (
                <div
                  key={opt.value}
                  className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                    safeFit === opt.value
                      ? "bg-blue-100 text-blue-700 font-medium"
                      : ""
                  }`}
                  onClick={() => onChangeFit?.(opt.value)}
                >
                  {opt.label}
                </div>
              ))}
            </PopoverContent>
          </Popover>
        ) : (
          <Input
            value={safeFit}
            variant="readonly"
            readOnly
            aria-label="フィット"
          />
        )}

        {/* 素材 */}
        <div className="label">素材</div>
        {isEdit ? (
          <Input
            value={safeMaterials}
            onChange={(e) => onChangeMaterials?.(e.target.value)}
            aria-label="素材"
          />
        ) : (
          <Input
            value={safeMaterials}
            variant="readonly"
            readOnly
            aria-label="素材"
          />
        )}

        {/* 重さ */}
        <div className="label">重さ</div>
        <div className="flex gap-8 items-center">
          {isEdit ? (
            <>
              <Input
                type="number"
                value={safeWeight}
                onChange={(e) =>
                  onChangeWeight?.(Number(e.target.value) || 0)
                }
                aria-label="重さ"
              />
              <span className="suffix">g</span>
            </>
          ) : (
            <>
              <Input
                value={safeWeight ? `${safeWeight}` : ""}
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
          {safeWashTags.map((t) => (
            <Badge
              key={t}
              className="chip inline-flex items-center gap-1.5 px-2 py-1"
            >
              <ShieldCheck size={14} />
              {t}
              {isEdit && onChangeWashTags && (
                <button
                  onClick={() =>
                    onChangeWashTags(
                      safeWashTags.filter((x) => x !== t),
                    )
                  }
                  className="chip-remove"
                  aria-label={`${t} を削除`}
                >
                  <X size={12} />
                </button>
              )}
            </Badge>
          ))}

          {isEdit && onChangeWashTags && (
            <Button
              variant="secondary"
              size="sm"
              onClick={() =>
                onChangeWashTags([...safeWashTags, "新タグ"])
              }
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
              <Button
                variant="outline"
                className="w-full justify-between pbc-select-trigger"
                aria-label="商品IDタグを選択"
              >
                {safeProductIdTag || "選択してください"}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="p-1">
              {PRODUCT_ID_TAG_OPTIONS.map((opt) => (
                <div
                  key={opt.value}
                  className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                    safeProductIdTag === opt.value
                      ? "bg-blue-100 text-blue-700 font-medium"
                      : ""
                  }`}
                  onClick={() => onChangeProductIdTag?.(opt.value)}
                >
                  {opt.label}
                </div>
              ))}
            </PopoverContent>
          </Popover>
        ) : (
          <Input
            value={safeProductIdTag}
            variant="readonly"
            readOnly
            aria-label="商品IDタグ"
          />
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;
