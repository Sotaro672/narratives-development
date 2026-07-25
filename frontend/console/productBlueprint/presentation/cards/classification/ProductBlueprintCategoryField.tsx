// frontend/console/productBlueprint/src/presentation/cards/classification/ProductBlueprintCategoryField.tsx

import * as React from "react";

import { Button } from "../../../../../shell/src/shared/ui/button";
import { Input } from "../../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shell/src/shared/ui/popover";

import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

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

function getCategoryId(category: ProductBlueprintCategoryOption): string {
  return asTrimmedString(category.id) || asTrimmedString(category.code);
}

function getParentId(category: ProductBlueprintCategoryOption): string {
  return asTrimmedString(category.parentId);
}

function getPath(category: ProductBlueprintCategoryOption): string[] {
  return Array.isArray(category.path)
    ? category.path.map((x) => asTrimmedString(x)).filter(Boolean)
    : [];
}

function getRootPathId(category: ProductBlueprintCategoryOption): string {
  return getPath(category)[0] ?? "";
}

function isParentCategory(category: ProductBlueprintCategoryOption): boolean {
  const parentId = getParentId(category);
  const path = getPath(category);

  return !parentId && path.length <= 1;
}

function isChildOfParent(
  category: ProductBlueprintCategoryOption,
  parentId: string,
): boolean {
  const categoryId = getCategoryId(category);
  const categoryParentId = getParentId(category);
  const rootPathId = getRootPathId(category);
  const path = getPath(category);

  if (!parentId) return false;

  // 親カテゴリ自身を子カテゴリに混ぜない
  if (categoryId === parentId) return false;

  // 2階層目以降のみ子カテゴリとして扱う
  if (path.length <= 1 && !categoryParentId) return false;

  return categoryParentId === parentId || rootPathId === parentId;
}

function getDisplayOrder(category: ProductBlueprintCategoryOption): number {
  const value = category.displayOrder;

  return typeof value === "number" && Number.isFinite(value)
    ? value
    : Number.MAX_SAFE_INTEGER;
}

function sortByDisplayOrder(
  a: ProductBlueprintCategoryOption,
  b: ProductBlueprintCategoryOption,
): number {
  return getDisplayOrder(a) - getDisplayOrder(b);
}

function findCategoryById(
  options: ProductBlueprintCategoryOption[],
  id: string,
): ProductBlueprintCategoryOption | undefined {
  const normalizedId = asTrimmedString(id);

  if (!normalizedId) {
    return undefined;
  }

  return options.find((option) => getCategoryId(option) === normalizedId);
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

  const safeOptions = React.useMemo(
    () => (Array.isArray(categoryOptions) ? categoryOptions : []),
    [categoryOptions],
  );

  const parentCategories = React.useMemo(
    () => safeOptions.filter(isParentCategory).sort(sortByDisplayOrder),
    [safeOptions],
  );

  const selectedCategory = React.useMemo(() => {
    if (category) {
      return category;
    }

    return findCategoryById(safeOptions, categoryId) ?? null;
  }, [category, categoryId, safeOptions]);

  const selectedParentIdFromCategory = React.useMemo(() => {
    if (!selectedCategory) {
      return "";
    }

    const parentId = getParentId(selectedCategory);

    if (parentId) {
      return parentId;
    }

    const rootPathId = getRootPathId(selectedCategory);

    if (rootPathId) {
      return rootPathId;
    }

    return getCategoryId(selectedCategory);
  }, [selectedCategory]);

  const [selectedParentId, setSelectedParentId] = React.useState<string>("");

  React.useEffect(() => {
    if (selectedParentIdFromCategory) {
      setSelectedParentId(selectedParentIdFromCategory);
    }
  }, [selectedParentIdFromCategory]);

  const selectedParent = React.useMemo(() => {
    return findCategoryById(parentCategories, selectedParentId) ?? null;
  }, [parentCategories, selectedParentId]);

  const childCategories = React.useMemo(() => {
    if (!selectedParentId) {
      return [];
    }

    return safeOptions
      .filter((option) => isChildOfParent(option, selectedParentId))
      .sort(sortByDisplayOrder);
  }, [safeOptions, selectedParentId]);

  const selectedChild = React.useMemo(() => {
    if (!selectedCategory) {
      return null;
    }

    if (!selectedParentId) {
      return null;
    }

    if (!isChildOfParent(selectedCategory, selectedParentId)) {
      return null;
    }

    return selectedCategory;
  }, [selectedCategory, selectedParentId]);

  const displayParentLabel = resolveProductBlueprintCategoryLabel(selectedParent);
  const displayChildLabel = resolveProductBlueprintCategoryLabel(selectedChild);

  const handleSelectParent = React.useCallback(
    (parent: ProductBlueprintCategoryOption) => {
      const nextParentId = getCategoryId(parent);

      setSelectedParentId(nextParentId);

      // 親を変更したら、子カテゴリの確定値はクリアする
      if (selectedChild && getParentId(selectedChild) !== nextParentId) {
        onChangeCategory?.(null);
        return;
      }

      if (selectedChild && getRootPathId(selectedChild) !== nextParentId) {
        onChangeCategory?.(null);
      }
    },
    [onChangeCategory, selectedChild],
  );

  const handleSelectChild = React.useCallback(
    (child: ProductBlueprintCategoryOption) => {
      onChangeCategory?.(child);
    },
    [onChangeCategory],
  );

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <div className="label">商品カテゴリ</div>

          {isEdit && safeOptions.length > 0 && onChangeCategory ? (
            <Popover>
              <PopoverTrigger>
                <Button
                  variant="outline"
                  className="w-full justify-between pbc-select-trigger"
                  aria-label="商品カテゴリを選択"
                >
                  {displayParentLabel || "選択してください。"}
                </Button>
              </PopoverTrigger>

              <PopoverContent align="start" className="p-1 w-64">
                <div className="max-h-64 overflow-y-auto space-y-1">
                  {parentCategories.map((parent) => {
                    const parentId = getCategoryId(parent);
                    const isSelected = selectedParentId === parentId;

                    return (
                      <button
                        key={parentId}
                        type="button"
                        className={`block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-blue-50 ${
                          isSelected
                            ? "bg-blue-100 text-blue-700 font-medium"
                            : ""
                        }`}
                        onClick={() => handleSelectParent(parent)}
                      >
                        {resolveProductBlueprintCategoryLabel(parent)}
                      </button>
                    );
                  })}

                  {parentCategories.length === 0 && (
                    <div className="px-3 py-2 text-sm text-slate-400">
                      商品カテゴリがありません。
                    </div>
                  )}
                </div>
              </PopoverContent>
            </Popover>
          ) : (
            <Input
              value={displayParentLabel}
              variant="readonly"
              readOnly
              aria-label="商品カテゴリ"
            />
          )}
        </div>

        <div>
          <div className="label">詳細カテゴリ</div>

          {selectedParentId ? (
            isEdit && onChangeCategory ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="詳細カテゴリを選択"
                  >
                    {displayChildLabel || "選択してください。"}
                  </Button>
                </PopoverTrigger>

                <PopoverContent align="start" className="p-1 w-64">
                  <div className="max-h-64 overflow-y-auto space-y-1">
                    {childCategories.map((child) => {
                      const childId = getCategoryId(child);
                      const isSelected = selectedChild
                        ? getCategoryId(selectedChild) === childId
                        : false;

                      return (
                        <button
                          key={childId}
                          type="button"
                          className={`block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-blue-50 ${
                            isSelected
                              ? "bg-blue-100 text-blue-700 font-medium"
                              : ""
                          }`}
                          onClick={() => handleSelectChild(child)}
                        >
                          {resolveProductBlueprintCategoryLabel(child)}
                        </button>
                      );
                    })}

                    {childCategories.length === 0 && (
                      <div className="px-3 py-2 text-sm text-slate-400">
                        詳細カテゴリがありません。
                      </div>
                    )}
                  </div>
                </PopoverContent>
              </Popover>
            ) : (
              <Input
                value={displayChildLabel}
                variant="readonly"
                readOnly
                aria-label="詳細カテゴリ"
              />
            )
          ) : (
            <Input
              value=""
              variant="readonly"
              readOnly
              aria-label="詳細カテゴリ"
            />
          )}
        </div>
      </div>

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