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
import { Checkbox } from "../../../../shell/src/shared/ui/checkbox";

import {
  FIT_OPTIONS,
  WASH_TAG_OPTIONS,
  type Fit,
  type WashTagOption,
} from "../../domain/entity/apparel";

import type {
  ProductBlueprintCategorySnapshot,
} from "../../domain/entity/productBlueprintCategory";

type BrandOption = {
  id: string;
  name: string;
};

type CategoryOption = ProductBlueprintCategorySnapshot;

export type ProductBlueprintPatchInput = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;

  productBlueprintCategoryId?: string | null;
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;

  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  assigneeId?: string | null;
};

type ProductBlueprintCardProps = {
  productBlueprintPatch?: ProductBlueprintPatchInput;

  productName?: string;
  brand?: string;

  brandId?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  productBlueprintCategoryId?: string;
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;
  productBlueprintCategoryOptions?: CategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  onChangeProductBlueprintCategory?: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;

  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];

  onChangeProductName?: (v: string) => void;
  onChangeFit?: (v: Fit) => void;
  onChangeMaterials?: (v: string) => void;
  onChangeWeight?: (v: number) => void;
  onChangeWashTags?: (nextTags: string[]) => void;

  mode?: "edit" | "view";
};

function asTrimmedString(v: unknown): string {
  return String(v ?? "").trim();
}

function resolveCategoryLabel(
  category: ProductBlueprintCategorySnapshot | null | undefined,
): string {
  if (!category) return "";

  return (
    asTrimmedString(category.nameJa) ||
    asTrimmedString(category.nameEn) ||
    asTrimmedString(category.code) ||
    asTrimmedString(category.id)
  );
}

function resolveCategoryOptionLabel(opt: ProductBlueprintCategorySnapshot): string {
  return resolveCategoryLabel(opt);
}

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productBlueprintPatch,

  productName,
  brand,
  brandId,
  brandOptions,
  brandLoading,
  brandError,
  onChangeBrandId,

  productBlueprintCategoryId,
  productBlueprintCategory,
  productBlueprintCategoryOptions,
  productBlueprintCategoryLoading,
  productBlueprintCategoryError,
  onChangeProductBlueprintCategory,

  fit,
  materials,
  weight,
  washTags,
  onChangeProductName,
  onChangeFit,
  onChangeMaterials,
  onChangeWeight,
  onChangeWashTags,
  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const mergedProductName = productName ?? productBlueprintPatch?.productName ?? "";
  const mergedBrandId = brandId ?? productBlueprintPatch?.brandId ?? "";
  const mergedBrandName = brand ?? productBlueprintPatch?.brandName ?? "";

  const mergedCategory =
    productBlueprintCategory ??
    productBlueprintPatch?.productBlueprintCategory ??
    null;

  const mergedCategoryId =
    productBlueprintCategoryId ??
    productBlueprintPatch?.productBlueprintCategoryId ??
    mergedCategory?.id ??
    "";

  const mergedFit =
    fit ??
    ((typeof productBlueprintPatch?.fit === "string"
      ? (productBlueprintPatch.fit as Fit)
      : undefined) as Fit | undefined) ??
    ("" as Fit);

  const mergedMaterials = materials ?? productBlueprintPatch?.material ?? "";
  const mergedWeight =
    typeof weight === "number"
      ? weight
      : typeof productBlueprintPatch?.weight === "number"
        ? productBlueprintPatch.weight
        : 0;

  const mergedWashTags = Array.isArray(washTags)
    ? washTags
    : Array.isArray(productBlueprintPatch?.qualityAssurance)
      ? productBlueprintPatch.qualityAssurance.filter(
          (x): x is string => typeof x === "string" && x.trim() !== "",
        )
      : [];

  const safeProductName = mergedProductName ?? "";
  const safeMaterials = mergedMaterials ?? "";
  const safeWeight =
    typeof mergedWeight === "number" && !Number.isNaN(mergedWeight) ? mergedWeight : 0;
  const safeWashTags = Array.isArray(mergedWashTags) ? mergedWashTags : [];
  const safeFit = mergedFit ?? ("" as Fit);

  const displayCategory = React.useMemo(() => {
    if (mergedCategory) {
      return resolveCategoryLabel(mergedCategory);
    }

    if (mergedCategoryId) {
      const found = productBlueprintCategoryOptions?.find(
        (opt) => opt.id === mergedCategoryId,
      );
      return resolveCategoryLabel(found) || mergedCategoryId;
    }

    return "";
  }, [mergedCategory, mergedCategoryId, productBlueprintCategoryOptions]);

  const selectedBrandName =
    brandOptions?.find((b) => b.id === mergedBrandId)?.name ?? "";

  const displayBrandName =
    String(mergedBrandName ?? "").trim() ||
    String(selectedBrandName ?? "").trim() ||
    (mergedBrandId ? `(${mergedBrandId})` : "");

  const washTagGroups = React.useMemo(() => {
    const map = new Map<string, WashTagOption[]>();

    for (const opt of WASH_TAG_OPTIONS) {
      const cat = opt.category;
      const list = map.get(cat) ?? [];
      list.push(opt);
      map.set(cat, list);
    }

    return Array.from(map.entries());
  }, []);

  const handleToggleWashTag = React.useCallback(
    (value: string) => {
      if (!onChangeWashTags) return;

      if (safeWashTags.includes(value)) {
        onChangeWashTags(safeWashTags.filter((t) => t !== value));
      } else {
        onChangeWashTags([...safeWashTags, value]);
      }
    },
    [onChangeWashTags, safeWashTags],
  );

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">基本情報</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
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

        <div className="label">ブランド</div>
        {isEdit && brandOptions && onChangeBrandId ? (
          <div className="mb-2 space-y-1">
            <Popover>
              <PopoverTrigger>
                <Button
                  variant="outline"
                  className="w-full justify-between pbc-select-trigger"
                  aria-label="ブランドを選択"
                >
                  {selectedBrandName || "ブランドを選択してください。"}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" className="p-1">
                {brandOptions.map((b) => (
                  <div
                    key={b.id}
                    className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                      mergedBrandId === b.id
                        ? "bg-blue-100 text-blue-700 font-medium"
                        : ""
                    }`}
                    onClick={() => onChangeBrandId(b.id)}
                  >
                    {b.name}
                  </div>
                ))}
              </PopoverContent>
            </Popover>

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
          <Input
            value={displayBrandName}
            variant="readonly"
            readOnly
            aria-label="ブランド"
          />
        )}

        <div className="pbc-fit-row">
          <div className="flex-1">
            <div className="label">商品カテゴリ</div>
            {isEdit && productBlueprintCategoryOptions && onChangeProductBlueprintCategory ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="商品カテゴリを選択"
                  >
                    {displayCategory || "商品カテゴリを選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {productBlueprintCategoryOptions.map(
                    (opt: ProductBlueprintCategorySnapshot) => (
                      <div
                        key={opt.id}
                        className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                          mergedCategoryId === opt.id
                            ? "bg-blue-100 text-blue-700 font-medium"
                            : ""
                        }`}
                        onClick={() => onChangeProductBlueprintCategory(opt)}
                      >
                        {resolveCategoryOptionLabel(opt)}
                      </div>
                    ),
                  )}
                </PopoverContent>
              </Popover>
            ) : (
              <Input
                value={displayCategory}
                variant="readonly"
                readOnly
                aria-label="商品カテゴリ"
              />
            )}

            {isEdit && productBlueprintCategoryLoading && (
              <p className="text-xs text-slate-400">商品カテゴリを取得中…</p>
            )}
            {isEdit && productBlueprintCategoryError && (
              <p className="text-xs text-red-500">
                商品カテゴリ一覧の取得に失敗しました。
              </p>
            )}
          </div>

          <div className="flex-1">
            <div className="label">フィット</div>
            {isEdit ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="フィットを選択"
                  >
                    {safeFit || "フィットを選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {FIT_OPTIONS.map((opt: { value: Fit; label: string }) => (
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
          </div>
        </div>

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

        <div className="label">重さ</div>
        <div className="flex gap-8 items-center">
          {isEdit ? (
            <>
              <Input
                type="number"
                value={safeWeight}
                onChange={(e) => onChangeWeight?.(Number(e.target.value) || 0)}
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

        <div className="label">品質保証（洗濯方法タグ）</div>
        <div className="chips flex flex-wrap gap-2">
          {safeWashTags.map((t) => (
            <Badge key={t} className="chip inline-flex items-center gap-1.5 px-2 py-1">
              <ShieldCheck size={14} />
              {t}
              {isEdit && onChangeWashTags && (
                <button
                  onClick={() => onChangeWashTags(safeWashTags.filter((x) => x !== t))}
                  className="chip-remove"
                  aria-label={`${t} を削除`}
                >
                  <X size={12} />
                </button>
              )}
            </Badge>
          ))}
        </div>

        {isEdit && onChangeWashTags && (
          <div className="mt-2 flex flex-wrap gap-2">
            {washTagGroups.map(([category, options]) => (
              <Popover key={category}>
                <PopoverTrigger>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="btn"
                    aria-label={`${category} のタグを追加`}
                  >
                    {category}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-2 space-y-1 w-64">
                  {options.map((opt: WashTagOption) => {
                    const checked = safeWashTags.includes(opt.value);
                    const checkboxId = `wash-tag-${opt.value}`;

                    return (
                      <label
                        key={opt.value}
                        htmlFor={checkboxId}
                        className="flex items-center gap-2 text-sm cursor-pointer py-0.5"
                      >
                        <Checkbox
                          id={checkboxId}
                          checked={checked}
                          onCheckedChange={() => handleToggleWashTag(opt.value)}
                        />
                        <span>{opt.label}</span>
                      </label>
                    );
                  })}
                </PopoverContent>
              </Popover>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;