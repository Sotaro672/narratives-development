// frontend/console/productBlueprint/src/presentation/components/ProductBlueprintCategoryField.tsx

import * as React from "react";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shell/src/shared/ui/popover";

import type { ProductBlueprintCategorySnapshot } from "../../domain/entity/productBlueprintCategory";

export type ProductBlueprintCategoryOption = ProductBlueprintCategorySnapshot;

type ProductBlueprintCategoryFieldProps = {
  categoryId: string;
  category: ProductBlueprintCategorySnapshot | null;
  categoryOptions?: ProductBlueprintCategoryOption[];
  categoryLoading?: boolean;
  categoryError?: Error | null;
  mode?: "edit" | "view";
  onChangeCategory?: (
    category: ProductBlueprintCategorySnapshot | null,
  ) => void;
};

function asTrimmedString(value: unknown): string {
  return String(value ?? "").trim();
}

export function resolveProductBlueprintCategoryLabel(
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

const ProductBlueprintCategoryField: React.FC<
  ProductBlueprintCategoryFieldProps
> = ({
  categoryId,
  category,
  categoryOptions,
  categoryLoading,
  categoryError,
  mode = "edit",
  onChangeCategory,
}) => {
  const isEdit = mode === "edit";

  const displayCategory = React.useMemo(() => {
    if (category) {
      return resolveProductBlueprintCategoryLabel(category);
    }

    if (categoryId) {
      const found = categoryOptions?.find((option) => option.id === categoryId);
      return resolveProductBlueprintCategoryLabel(found) || categoryId;
    }

    return "";
  }, [category, categoryId, categoryOptions]);

  return (
    <>
      <div className="label">商品カテゴリ</div>
      {isEdit && categoryOptions && onChangeCategory ? (
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
            {categoryOptions.map((option) => (
              <div
                key={option.id}
                className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                  categoryId === option.id
                    ? "bg-blue-100 text-blue-700 font-medium"
                    : ""
                }`}
                onClick={() => onChangeCategory(option)}
              >
                {resolveProductBlueprintCategoryLabel(option)}
              </div>
            ))}
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

      {isEdit && categoryLoading && (
        <p className="text-xs text-slate-400">商品カテゴリを取得中…</p>
      )}
      {isEdit && categoryError && (
        <p className="text-xs text-red-500">
          商品カテゴリ一覧の取得に失敗しました。
        </p>
      )}
    </>
  );
};

export default ProductBlueprintCategoryField;