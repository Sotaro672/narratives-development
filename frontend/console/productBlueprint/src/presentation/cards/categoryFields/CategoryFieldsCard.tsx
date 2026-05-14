// frontend/console/productBlueprint/src/presentation/cards/categoryFields/CategoryFieldsCard.tsx

import * as React from "react";
import { SlidersHorizontal } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shell/src/shared/ui";
import { Button } from "../../../../../shell/src/shared/ui/button";
import { Input } from "../../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shell/src/shared/ui/popover";

import { FIT_OPTIONS, type Fit } from "../../../domain/entity/apparel";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
} from "../../../domain/entity/productBlueprintCategory";

import {
  getCategoryCardVisibility,
  isNumberCategoryField,
  toCategoryInputValue,
  toCategoryNumberOrNull,
} from "../../../domain/entity/categoryCardVisibility";

import WashTagField from "./WashTagField";

type CategoryFieldsCardProps = {
  categoryCode: string;
  categoryFields?: CategoryFieldValues | null;
  mode?: "edit" | "view";
  onChangeCategoryField?: (key: string, value: CategoryFieldValue) => void;
};

function normalizeCategoryCode(categoryCode: string): string {
  return String(categoryCode ?? "").trim();
}

function isChildCategoryCode(categoryCode: string): boolean {
  const code = normalizeCategoryCode(categoryCode);

  if (!code) {
    return false;
  }

  return code.includes(".");
}

function resolveCategoryFieldsCardTitle(categoryCode: string): string {
  const code = normalizeCategoryCode(categoryCode);

  if (code.startsWith("apparel.")) {
    return "衣類情報";
  }

  if (code.startsWith("alcohol.")) {
    return "酒類情報";
  }

  if (code.startsWith("cosmetics.")) {
    return "化粧品情報";
  }

  if (code.startsWith("healthcare.")) {
    return "ヘルスケア情報";
  }

  if (code.startsWith("other.")) {
    return "その他情報";
  }

  return "カテゴリ情報";
}

function getCategoryFieldValue(
  categoryFields: CategoryFieldValues | null | undefined,
  key: string,
): CategoryFieldValue {
  const value = categoryFields?.[key];

  if (
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean" ||
    value === null
  ) {
    return value;
  }

  return null;
}

function getStringFieldValue(
  categoryFields: CategoryFieldValues | null | undefined,
  key: string,
): string {
  const value = categoryFields?.[key];
  return typeof value === "string" ? value : "";
}

function getNumberFieldValue(
  categoryFields: CategoryFieldValues | null | undefined,
  key: string,
): number {
  const value = categoryFields?.[key];
  return typeof value === "number" && !Number.isNaN(value) ? value : 0;
}

function getWashTagsValue(
  categoryFields: CategoryFieldValues | null | undefined,
): string[] {
  const rawValue =
    categoryFields?.washTags ?? categoryFields?.qualityAssurance ?? [];

  if (!Array.isArray(rawValue)) {
    return [];
  }

  return rawValue.filter(
    (item): item is string => typeof item === "string" && item.trim() !== "",
  );
}

const CategoryFieldsCard: React.FC<CategoryFieldsCardProps> = ({
  categoryCode,
  categoryFields,
  mode = "edit",
  onChangeCategoryField,
}) => {
  const normalizedCategoryCode = normalizeCategoryCode(categoryCode);
  const isEdit = mode === "edit";

  const visibility = React.useMemo(
    () => getCategoryCardVisibility(normalizedCategoryCode),
    [normalizedCategoryCode],
  );

  const handleChangeCategoryField = React.useCallback(
    (key: string, rawValue: string) => {
      if (!onChangeCategoryField) {
        return;
      }

      if (isNumberCategoryField(key)) {
        onChangeCategoryField(key, toCategoryNumberOrNull(rawValue));
        return;
      }

      onChangeCategoryField(key, rawValue.trim() === "" ? null : rawValue);
    },
    [onChangeCategoryField],
  );

  const handleChangeWashTags = React.useCallback(
    (nextTags: string[]) => {
      onChangeCategoryField?.("washTags", nextTags as CategoryFieldValue);
    },
    [onChangeCategoryField],
  );

  const fitValue = getStringFieldValue(categoryFields, "fit") as Fit;
  const materialValue = getStringFieldValue(categoryFields, "material");
  const weightValue = getNumberFieldValue(categoryFields, "weight");
  const washTagsValue = getWashTagsValue(categoryFields);

  const cardTitle = resolveCategoryFieldsCardTitle(normalizedCategoryCode);

  const hasVisibleFields =
    visibility.showVintage ||
    visibility.showRegion ||
    visibility.showWeight ||
    visibility.showFit ||
    visibility.showMaterial ||
    visibility.showAlcoholContent ||
    visibility.showVolume ||
    visibility.showWashTags;

  /**
   * hooks をすべて呼び出した後で return する。
   * 親カテゴリだけ選択されている状態では非表示。
   */
  if (!isChildCategoryCode(normalizedCategoryCode)) {
    return null;
  }

  /**
   * 子カテゴリでも表示対象 field がない場合は非表示。
   */
  if (!hasVisibleFields) {
    return null;
  }

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <SlidersHorizontal size={16} />
        <CardTitle className="box__title">{cardTitle}</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {visibility.showVintage && (
          <>
            <div className="label">ヴィンテージ</div>
            <div className="flex gap-8 items-center">
              {isEdit ? (
                <Input
                  type="number"
                  value={toCategoryInputValue(
                    getCategoryFieldValue(categoryFields, "vintage"),
                  )}
                  onChange={(e) =>
                    handleChangeCategoryField("vintage", e.target.value)
                  }
                  aria-label="ヴィンテージ"
                />
              ) : (
                <Input
                  value={toCategoryInputValue(
                    getCategoryFieldValue(categoryFields, "vintage"),
                  )}
                  variant="readonly"
                  readOnly
                  aria-label="ヴィンテージ"
                />
              )}
            </div>
          </>
        )}

        {visibility.showRegion && (
          <>
            <div className="label">地域・産地</div>
            {isEdit ? (
              <Input
                value={toCategoryInputValue(
                  getCategoryFieldValue(categoryFields, "region"),
                )}
                onChange={(e) =>
                  handleChangeCategoryField("region", e.target.value)
                }
                aria-label="地域・産地"
              />
            ) : (
              <Input
                value={toCategoryInputValue(
                  getCategoryFieldValue(categoryFields, "region"),
                )}
                variant="readonly"
                readOnly
                aria-label="地域・産地"
              />
            )}
          </>
        )}

        {visibility.showWeight && (
          <>
            <div className="label">重さ</div>
            <div className="flex gap-8 items-center">
              {isEdit ? (
                <>
                  <Input
                    type="number"
                    value={weightValue}
                    onChange={(e) =>
                      handleChangeCategoryField("weight", e.target.value)
                    }
                    aria-label="重さ"
                  />
                  <span className="suffix">g</span>
                </>
              ) : (
                <>
                  <Input
                    value={weightValue ? `${weightValue}` : ""}
                    variant="readonly"
                    readOnly
                    aria-label="重さ"
                  />
                  <span className="suffix">g</span>
                </>
              )}
            </div>
          </>
        )}

        {visibility.showFit && (
          <>
            <div className="label">フィット</div>
            {isEdit ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="フィットを選択"
                  >
                    {fitValue || "フィットを選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {FIT_OPTIONS.map((option: { value: Fit; label: string }) => (
                    <div
                      key={option.value}
                      className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                        fitValue === option.value
                          ? "bg-blue-100 text-blue-700 font-medium"
                          : ""
                      }`}
                      onClick={() =>
                        onChangeCategoryField?.("fit", option.value)
                      }
                    >
                      {option.label}
                    </div>
                  ))}
                </PopoverContent>
              </Popover>
            ) : (
              <Input
                value={fitValue}
                variant="readonly"
                readOnly
                aria-label="フィット"
              />
            )}
          </>
        )}

        {visibility.showMaterial && (
          <>
            <div className="label">素材</div>
            {isEdit ? (
              <Input
                value={materialValue}
                onChange={(e) =>
                  handleChangeCategoryField("material", e.target.value)
                }
                aria-label="素材"
              />
            ) : (
              <Input
                value={materialValue}
                variant="readonly"
                readOnly
                aria-label="素材"
              />
            )}
          </>
        )}

        {visibility.showAlcoholContent && (
          <>
            <div className="label">アルコール度数</div>
            <div className="flex gap-8 items-center">
              {isEdit ? (
                <>
                  <Input
                    type="number"
                    value={toCategoryInputValue(
                      getCategoryFieldValue(categoryFields, "alcoholContent"),
                    )}
                    onChange={(e) =>
                      handleChangeCategoryField(
                        "alcoholContent",
                        e.target.value,
                      )
                    }
                    aria-label="アルコール度数"
                  />
                  <span className="suffix">%</span>
                </>
              ) : (
                <>
                  <Input
                    value={toCategoryInputValue(
                      getCategoryFieldValue(categoryFields, "alcoholContent"),
                    )}
                    variant="readonly"
                    readOnly
                    aria-label="アルコール度数"
                  />
                  <span className="suffix">%</span>
                </>
              )}
            </div>
          </>
        )}

        {visibility.showVolume && (
          <>
            <div className="label">容量</div>
            <div className="flex gap-8 items-center">
              {isEdit ? (
                <>
                  <Input
                    type="number"
                    value={toCategoryInputValue(
                      getCategoryFieldValue(categoryFields, "volume"),
                    )}
                    onChange={(e) =>
                      handleChangeCategoryField("volume", e.target.value)
                    }
                    aria-label="容量"
                  />
                  <span className="suffix">ml</span>
                </>
              ) : (
                <>
                  <Input
                    value={toCategoryInputValue(
                      getCategoryFieldValue(categoryFields, "volume"),
                    )}
                    variant="readonly"
                    readOnly
                    aria-label="容量"
                  />
                  <span className="suffix">ml</span>
                </>
              )}
            </div>
          </>
        )}

        {visibility.showWashTags && (
          <WashTagField
            value={washTagsValue}
            mode={mode}
            onChange={handleChangeWashTags}
          />
        )}
      </CardContent>
    </Card>
  );
};

export default CategoryFieldsCard;