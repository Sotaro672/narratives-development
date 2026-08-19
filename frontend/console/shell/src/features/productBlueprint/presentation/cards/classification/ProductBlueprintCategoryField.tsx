// frontend/console/shell/src/features/productBlueprint/presentation/cards/classification/ProductBlueprintCategoryField.tsx

import * as React from "react";

import { Button } from "../../../../../shared/ui/button";
import { Input } from "../../../../../shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shared/ui/popover";

import {
  APPAREL_CATEGORY_OPTIONS,
} from "../../../../../shared/types/apparel";

import {
  ALCOHOL_CATEGORY_OPTIONS,
} from "../../../domain/alcohol";
import {
  COSMETICS_CATEGORY_OPTIONS,
} from "../../../domain/cosmetics";
import {
  HEALTHCARE_CATEGORY_OPTIONS,
} from "../../../domain/healthcare";
import {
  OTHER_CATEGORY_OPTIONS,
} from "../../../domain/other";
import {
  toProductBlueprintCategoryPathKey,
  type ProductBlueprintCategoryPath,
} from "../../../domain/productBlueprintCategory";

export type ProductBlueprintCategoryOption =
  ProductBlueprintCategoryPath;

type ProductBlueprintCategoryFieldProps = {
  productBlueprintCategoryPath: ProductBlueprintCategoryPath | null;
  productBlueprintCategoryOptions?: ProductBlueprintCategoryOption[];
  productBlueprintCategoryLoading?: boolean;
  productBlueprintCategoryError?: Error | null;
  mode?: "edit" | "view";
  onChangeProductBlueprintCategoryPath?: (
    productBlueprintCategoryPath: ProductBlueprintCategoryPath | null,
  ) => void;
};

const EMPTY_CATEGORY_OPTIONS: ProductBlueprintCategoryOption[] = [];

const ROOT_CATEGORY_LABELS: Readonly<
  Record<string, string>
> = {
  apparel: "衣類",
  alcohol: "酒類",
  cosmetics: "化粧品",
  healthcare: "ヘルスケア",
  other: "その他",
};

const CATEGORY_LABEL_BY_PATH_KEY: Readonly<
  Record<string, string>
> = Object.fromEntries(
  [
    ...APPAREL_CATEGORY_OPTIONS,
    ...ALCOHOL_CATEGORY_OPTIONS,
    ...COSMETICS_CATEGORY_OPTIONS,
    ...HEALTHCARE_CATEGORY_OPTIONS,
    ...OTHER_CATEGORY_OPTIONS,
  ].map(
    (option) => [
      option.value,
      option.label,
    ],
  ),
);

function isSameCategoryPath(
  left: ProductBlueprintCategoryPath | null | undefined,
  right: ProductBlueprintCategoryPath | null | undefined,
): boolean {
  if (!left || !right) {
    return left === right;
  }

  if (left.length !== right.length) {
    return false;
  }

  return left.every(
    (segment, index) =>
      segment === right[index],
  );
}

function findCategoryByPath(
  options: ProductBlueprintCategoryOption[],
  path: ProductBlueprintCategoryPath | null | undefined,
): ProductBlueprintCategoryOption | null {
  if (
    !path ||
    path.length === 0
  ) {
    return null;
  }

  return (
    options.find(
      (option) =>
        isSameCategoryPath(
          option,
          path,
        ),
    ) ?? null
  );
}

function buildParentCategoryOptions(
  options: ProductBlueprintCategoryOption[],
): ProductBlueprintCategoryOption[] {
  const parentCategories:
    ProductBlueprintCategoryOption[] = [];

  const seen =
    new Set<string>();

  for (const option of options) {
    const root =
      option[0];

    if (
      !root ||
      seen.has(root)
    ) {
      continue;
    }

    seen.add(root);

    parentCategories.push(
      [root],
    );
  }

  return parentCategories;
}

function buildChildCategoryOptions(
  options: ProductBlueprintCategoryOption[],
  root: string,
): ProductBlueprintCategoryOption[] {
  if (root === "") {
    return [];
  }

  const childCategories:
    ProductBlueprintCategoryOption[] = [];

  const seen =
    new Set<string>();

  for (const option of options) {
    if (
      option.length <= 1 ||
      option[0] !== root
    ) {
      continue;
    }

    const pathKey =
      toProductBlueprintCategoryPathKey(
        option,
      );

    if (seen.has(pathKey)) {
      continue;
    }

    seen.add(pathKey);

    childCategories.push(
      [...option],
    );
  }

  return childCategories;
}

export function resolveProductBlueprintCategoryLabel(
  productBlueprintCategoryPath:
    ProductBlueprintCategoryPath | null | undefined,
): string {
  if (
    !productBlueprintCategoryPath ||
    productBlueprintCategoryPath.length === 0
  ) {
    return "";
  }

  if (
    productBlueprintCategoryPath.length === 1
  ) {
    const root =
      productBlueprintCategoryPath[0] ?? "";

    return (
      ROOT_CATEGORY_LABELS[root] ??
      root
    );
  }

  const pathKey =
    toProductBlueprintCategoryPathKey(
      productBlueprintCategoryPath,
    );

  return (
    CATEGORY_LABEL_BY_PATH_KEY[pathKey] ??
    productBlueprintCategoryPath[
      productBlueprintCategoryPath.length - 1
    ] ??
    ""
  );
}

function resolveProductBlueprintParentCategoryLabel(
  productBlueprintCategoryPath:
    ProductBlueprintCategoryPath | null | undefined,
): string {
  const root =
    productBlueprintCategoryPath?.[0] ?? "";

  if (root === "") {
    return "";
  }

  return (
    ROOT_CATEGORY_LABELS[root] ??
    root
  );
}

const ProductBlueprintCategoryField: React.FC<
  ProductBlueprintCategoryFieldProps
> = ({
  productBlueprintCategoryPath,
  productBlueprintCategoryOptions,
  productBlueprintCategoryLoading = false,
  productBlueprintCategoryError = null,
  mode = "edit",
  onChangeProductBlueprintCategoryPath,
}) => {
  const isEdit = mode === "edit";
  const canEditCategory =
    isEdit &&
    Boolean(
      onChangeProductBlueprintCategoryPath,
    );

  const options =
    productBlueprintCategoryOptions ??
    EMPTY_CATEGORY_OPTIONS;

  const parentCategories = React.useMemo(
    () =>
      buildParentCategoryOptions(
        options,
      ),
    [options],
  );

  const selectedCategory = React.useMemo(() => {
    if (
      !productBlueprintCategoryPath ||
      productBlueprintCategoryPath.length === 0
    ) {
      return null;
    }

    return (
      findCategoryByPath(
        options,
        productBlueprintCategoryPath,
      ) ?? [
        ...productBlueprintCategoryPath,
      ]
    );
  }, [
    options,
    productBlueprintCategoryPath,
  ]);

  const selectedParentRootFromCategory =
    selectedCategory?.[0] ?? "";

  const [
    selectedParentRoot,
    setSelectedParentRoot,
  ] = React.useState(
    selectedParentRootFromCategory,
  );

  React.useEffect(() => {
    setSelectedParentRoot(
      selectedParentRootFromCategory,
    );
  }, [selectedParentRootFromCategory]);

  const selectedParent = React.useMemo(
    () =>
      selectedParentRoot === ""
        ? null
        : [selectedParentRoot],
    [selectedParentRoot],
  );

  const childCategories = React.useMemo(
    () =>
      buildChildCategoryOptions(
        options,
        selectedParentRoot,
      ),
    [
      options,
      selectedParentRoot,
    ],
  );

  const selectedChild = React.useMemo(() => {
    if (
      !selectedCategory ||
      selectedCategory.length <= 1 ||
      selectedCategory[0] !== selectedParentRoot
    ) {
      return null;
    }

    return (
      findCategoryByPath(
        childCategories,
        selectedCategory,
      ) ?? selectedCategory
    );
  }, [
    childCategories,
    selectedCategory,
    selectedParentRoot,
  ]);

  const displayParentLabel = canEditCategory
    ? resolveProductBlueprintCategoryLabel(
        selectedParent,
      )
    : resolveProductBlueprintParentCategoryLabel(
        selectedCategory,
      );

  const displayChildLabel = canEditCategory
    ? resolveProductBlueprintCategoryLabel(
        selectedChild,
      )
    : (
        selectedCategory &&
        selectedCategory.length > 1
      )
      ? resolveProductBlueprintCategoryLabel(
          selectedCategory,
        )
      : "";

  const handleSelectParent = React.useCallback(
    (parent: ProductBlueprintCategoryOption) => {
      const nextParentRoot =
        parent[0] ?? "";

      setSelectedParentRoot(
        nextParentRoot,
      );

      if (
        selectedCategory &&
        selectedCategory[0] !== nextParentRoot
      ) {
        onChangeProductBlueprintCategoryPath?.(
          null,
        );
      }
    },
    [
      onChangeProductBlueprintCategoryPath,
      selectedCategory,
    ],
  );

  const handleSelectChild = React.useCallback(
    (child: ProductBlueprintCategoryOption) => {
      onChangeProductBlueprintCategoryPath?.(
        [...child],
      );
    },
    [onChangeProductBlueprintCategoryPath],
  );

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <div className="label">商品カテゴリ</div>

          {canEditCategory ? (
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-between pbc-select-trigger"
                  aria-label="商品カテゴリを選択"
                  disabled={
                    productBlueprintCategoryLoading ||
                    parentCategories.length === 0
                  }
                >
                  {displayParentLabel || "選択してください。"}
                </Button>
              </PopoverTrigger>

              <PopoverContent align="start" className="w-64 p-1">
                <div className="max-h-64 space-y-1 overflow-y-auto">
                  {parentCategories.map((parent) => {
                    const parentRoot =
                      parent[0] ?? "";

                    const isSelected =
                      selectedParentRoot ===
                      parentRoot;

                    return (
                      <button
                        key={parentRoot}
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

          {canEditCategory ? (
            selectedParentRoot !== "" ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="詳細カテゴリを選択"
                    disabled={
                      productBlueprintCategoryLoading ||
                      childCategories.length === 0
                    }
                  >
                    {displayChildLabel || "選択してください。"}
                  </Button>
                </PopoverTrigger>

                <PopoverContent align="start" className="w-64 p-1">
                  <div className="max-h-64 space-y-1 overflow-y-auto">
                    {childCategories.map((child) => {
                      const childPathKey =
                        toProductBlueprintCategoryPathKey(
                          child,
                        );

                      const isSelected =
                        isSameCategoryPath(
                          selectedChild,
                          child,
                        );

                      return (
                        <button
                          key={childPathKey}
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
                value=""
                variant="readonly"
                readOnly
                aria-label="詳細カテゴリ"
              />
            )
          ) : (
            <Input
              value={displayChildLabel}
              variant="readonly"
              readOnly
              aria-label="詳細カテゴリ"
            />
          )}
        </div>
      </div>

      {canEditCategory && productBlueprintCategoryLoading && (
        <p className="text-xs text-slate-400">
          商品カテゴリを取得中…
        </p>
      )}

      {canEditCategory && productBlueprintCategoryError && (
        <p className="text-xs text-red-500">
          商品カテゴリ一覧の取得に失敗しました。
        </p>
      )}
    </>
  );
};

export default ProductBlueprintCategoryField;