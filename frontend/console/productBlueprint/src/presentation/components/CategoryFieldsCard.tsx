// frontend/console/productBlueprint/src/presentation/components/CategoryFieldsCard.tsx

import * as React from "react";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shell/src/shared/ui/popover";

import {
  FIT_OPTIONS,
  type Fit,
} from "../../domain/entity/apparel";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
} from "../../domain/entity/productBlueprintCategory";

import {
  getCategoryCardVisibility,
  isNumberCategoryField,
  toCategoryInputValue,
  toCategoryNumberOrNull,
} from "../../domain/entity/categoryCardVisibility";

import CategoryTextField from "./CategoryTextField";
import CategoryNumberField from "./CategoryNumberField";

type CategoryFieldsCardProps = {
  categoryCode: string;

  fit: Fit;
  material: string;
  weight: number;
  categoryFields?: CategoryFieldValues | null;

  mode?: "edit" | "view";

  onChangeFit?: (v: Fit) => void;
  onChangeMaterials?: (v: string) => void;
  onChangeWeight?: (v: number) => void;
  onChangeCategoryField?: (key: string, value: CategoryFieldValue) => void;
};

const CategoryFieldsCard: React.FC<CategoryFieldsCardProps> = ({
  categoryCode,
  fit,
  material,
  weight,
  categoryFields,
  mode = "edit",
  onChangeFit,
  onChangeMaterials,
  onChangeWeight,
  onChangeCategoryField,
}) => {
  const isEdit = mode === "edit";

  const visibility = React.useMemo(
    () => getCategoryCardVisibility(categoryCode),
    [categoryCode],
  );

  const getCategoryFieldValue = React.useCallback(
    (key: string): CategoryFieldValue => {
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
    },
    [categoryFields],
  );

  const handleChangeCategoryField = React.useCallback(
    (key: string, rawValue: string) => {
      if (key === "material") {
        onChangeMaterials?.(rawValue);
      }

      if (key === "weight") {
        onChangeWeight?.(toCategoryNumberOrNull(rawValue) ?? 0);
      }

      if (!onChangeCategoryField) {
        return;
      }

      if (isNumberCategoryField(key)) {
        onChangeCategoryField(key, toCategoryNumberOrNull(rawValue));
        return;
      }

      onChangeCategoryField(key, rawValue.trim() === "" ? null : rawValue);
    },
    [onChangeCategoryField, onChangeMaterials, onChangeWeight],
  );

  const materialValue = String(material ?? "");
  const weightValue =
    typeof weight === "number" && !Number.isNaN(weight) ? weight : 0;

  return (
    <>
      {visibility.showVintage && (
        <CategoryNumberField
          label="ヴィンテージ"
          ariaLabel="ヴィンテージ"
          value={toCategoryInputValue(getCategoryFieldValue("vintage"))}
          mode={mode}
          onChange={(value) => handleChangeCategoryField("vintage", value)}
        />
      )}

      {visibility.showRegion && (
        <CategoryTextField
          label="地域・産地"
          ariaLabel="地域・産地"
          value={toCategoryInputValue(getCategoryFieldValue("region"))}
          mode={mode}
          onChange={(value) => handleChangeCategoryField("region", value)}
        />
      )}

      {visibility.showWeight && (
        <CategoryNumberField
          label="重さ"
          ariaLabel="重さ"
          value={weightValue}
          suffix="g"
          mode={mode}
          onChange={(value) => handleChangeCategoryField("weight", value)}
        />
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
                  {fit || "フィットを選択してください。"}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" className="p-1">
                {FIT_OPTIONS.map((option: { value: Fit; label: string }) => (
                  <div
                    key={option.value}
                    className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                      fit === option.value
                        ? "bg-blue-100 text-blue-700 font-medium"
                        : ""
                    }`}
                    onClick={() => {
                      onChangeFit?.(option.value);
                      onChangeCategoryField?.("fit", option.value);
                    }}
                  >
                    {option.label}
                  </div>
                ))}
              </PopoverContent>
            </Popover>
          ) : (
            <Input
              value={fit}
              variant="readonly"
              readOnly
              aria-label="フィット"
            />
          )}
        </>
      )}

      {visibility.showMaterial && (
        <CategoryTextField
          label="素材"
          ariaLabel="素材"
          value={materialValue}
          mode={mode}
          onChange={(value) => handleChangeCategoryField("material", value)}
        />
      )}

      {visibility.showAlcoholContent && (
        <CategoryNumberField
          label="アルコール度数"
          ariaLabel="アルコール度数"
          value={toCategoryInputValue(getCategoryFieldValue("alcoholContent"))}
          suffix="%"
          mode={mode}
          onChange={(value) =>
            handleChangeCategoryField("alcoholContent", value)
          }
        />
      )}

      {visibility.showVolume && (
        <CategoryNumberField
          label="容量"
          ariaLabel="容量"
          value={toCategoryInputValue(getCategoryFieldValue("volume"))}
          suffix="ml"
          mode={mode}
          onChange={(value) => handleChangeCategoryField("volume", value)}
        />
      )}
    </>
  );
};

export default CategoryFieldsCard;