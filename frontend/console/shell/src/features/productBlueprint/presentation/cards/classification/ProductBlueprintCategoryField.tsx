// frontend/console/shell/src/features/productBlueprint/presentation/cards/classification/ProductBlueprintCategoryField.tsx

import * as React from "react";

import { Button } from "../../../../../shared/ui/button";
import { Input } from "../../../../../shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shared/ui/popover";

import type {
  ProductBlueprintCategorySnapshot,
} from "../../../domain/productBlueprintCategory";

export type ProductBlueprintCategoryOption =
  ProductBlueprintCategorySnapshot;

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

const EMPTY_CATEGORY_OPTIONS: ProductBlueprintCategoryOption[] = [];

export function resolveProductBlueprintCategoryLabel(
  category:
    | ProductBlueprintCategorySnapshot
    | null
    | undefined,
): string {
  return category?.nameJa ?? "";
}

function findCategoryById(
  options: ProductBlueprintCategoryOption[],
  id: string,
): ProductBlueprintCategoryOption | null {
  if (id === "") {
    return null;
  }

  return options.find((option) => option.id === id) ?? null;
}

const ProductBlueprintCategoryField: React.FC<
  ProductBlueprintCategoryFieldProps
> = ({
  categoryId,
  category,
  categoryOptions,
  categoryLoading = false,
  categoryError = null,
  mode = "edit",
  onChangeCategory,
}) => {
  const isEdit = mode === "edit";
  const options = categoryOptions ?? EMPTY_CATEGORY_OPTIONS;

  const parentCategories = React.useMemo(
    () => options.filter((option) => option.parentId === null),
    [options],
  );

  const selectedCategory = React.useMemo(() => {
    if (categoryId !== "") {
      const categoryOption = findCategoryById(options, categoryId);
      if (categoryOption) {
        return categoryOption;
      }
    }

    return category;
  }, [category, categoryId, options]);

  const selectedParentIdFromCategory =
    selectedCategory?.parentId ?? "";

  const [selectedParentId, setSelectedParentId] =
    React.useState(selectedParentIdFromCategory);

  React.useEffect(() => {
    if (selectedParentIdFromCategory !== "") {
      setSelectedParentId(selectedParentIdFromCategory);
    }
  }, [selectedParentIdFromCategory]);

  const selectedParent = React.useMemo(
    () => findCategoryById(parentCategories, selectedParentId),
    [parentCategories, selectedParentId],
  );

  const childCategories = React.useMemo(() => {
    if (selectedParentId === "") {
      return [];
    }

    return options.filter(
      (option) => option.parentId === selectedParentId,
    );
  }, [options, selectedParentId]);

  const selectedChild = React.useMemo(() => {
    if (
      !selectedCategory ||
      selectedCategory.parentId !== selectedParentId
    ) {
      return null;
    }

    return selectedCategory;
  }, [selectedCategory, selectedParentId]);

  const displayParentLabel =
    resolveProductBlueprintCategoryLabel(selectedParent);

  const displayChildLabel =
    resolveProductBlueprintCategoryLabel(selectedChild);

  const handleSelectParent = React.useCallback(
    (parent: ProductBlueprintCategoryOption) => {
      const nextParentId = parent.id;
      setSelectedParentId(nextParentId);

      if (
        selectedCategory &&
        selectedCategory.parentId !== nextParentId
      ) {
        onChangeCategory?.(null);
      }
    },
    [onChangeCategory, selectedCategory],
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

          {isEdit && onChangeCategory ? (
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-between pbc-select-trigger"
                  aria-label="商品カテゴリを選択"
                  disabled={categoryLoading || parentCategories.length === 0}
                >
                  {displayParentLabel || "選択してください。"}
                </Button>
              </PopoverTrigger>

              <PopoverContent align="start" className="w-64 p-1">
                <div className="max-h-64 space-y-1 overflow-y-auto">
                  {parentCategories.map((parent) => {
                    const isSelected =
                      selectedParentId === parent.id;

                    return (
                      <button
                        key={parent.id}
                        type="button"
                        className={`block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-blue-50 ${
                          isSelected
                            ? "bg-blue-100 font-medium text-blue-700"
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

          {selectedParentId !== "" ? (
            isEdit && onChangeCategory ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="詳細カテゴリを選択"
                    disabled={categoryLoading || childCategories.length === 0}
                  >
                    {displayChildLabel || "選択してください。"}
                  </Button>
                </PopoverTrigger>

                <PopoverContent align="start" className="w-64 p-1">
                  <div className="max-h-64 space-y-1 overflow-y-auto">
                    {childCategories.map((child) => {
                      const isSelected =
                        selectedChild?.id === child.id;

                      return (
                        <button
                          key={child.id}
                          type="button"
                          className={`block w-full rounded-md px-3 py-2 text-left text-sm hover:bg-blue-50 ${
                            isSelected
                              ? "bg-blue-100 font-medium text-blue-700"
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
        <p className="text-xs text-slate-400">
          商品カテゴリを取得中…
        </p>
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