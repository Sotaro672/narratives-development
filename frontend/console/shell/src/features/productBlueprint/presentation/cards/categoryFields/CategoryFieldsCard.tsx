// frontend/console/shell/src/features/productBlueprint/presentation/cards/categoryFields/CategoryFieldsCard.tsx

import * as React from "react";
import { SlidersHorizontal } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../../shared/ui";
import { Button } from "../../../../../shared/ui/button";
import { Input } from "../../../../../shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shared/ui/popover";
import {
  FIT_OPTIONS,
  type Fit,
} from "../../../../../shared/types/apparel";

import type {
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategoryPath,
} from "../../../domain/productBlueprintCategory";
import {
  getCategoryCardVisibility,
  isNumberCategoryField,
  toCategoryInputValue,
  toCategoryNumberOrNull,
} from "../../../domain/categoryCardVisibility";

import WashTagField from "./WashTagField";

type CategoryFieldsCardProps = {
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
  categoryFields?: CategoryFieldValues | null;
  mode?: "edit" | "view";
  onChangeCategoryField?: (
    key: string,
    value: CategoryFieldValue,
  ) => void;
};

type NumericInputNoteProps = {
  id: string;
  example: string;
};

type CategoryCardVisibility = ReturnType<
  typeof getCategoryCardVisibility
>;

function resolveCategoryFieldsCardTitle(
  visibility: CategoryCardVisibility,
): string {
  if (visibility.showAlcoholContent) {
    return "酒類情報";
  }

  if (
    visibility.showWeight ||
    visibility.showFit ||
    visibility.showWashTags
  ) {
    return "衣類情報";
  }

  if (visibility.showVolume) {
    return "化粧品情報";
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
): number | "" {
  const value = categoryFields?.[key];

  return typeof value === "number" && Number.isFinite(value)
    ? value
    : "";
}

function isFit(value: string): value is Fit {
  return FIT_OPTIONS.some((option) => option.value === value);
}

function getFitFieldValue(
  categoryFields: CategoryFieldValues | null | undefined,
): Fit | "" {
  const value = getStringFieldValue(categoryFields, "fit");
  return isFit(value) ? value : "";
}

function getWashTagsValue(
  categoryFields: CategoryFieldValues | null | undefined,
): string[] {
  const value = categoryFields?.washTags;

  if (!Array.isArray(value)) {
    return [];
  }

  return value.filter(
    (item): item is string =>
      typeof item === "string" && item.trim() !== "",
  );
}

function NumericInputNote({
  id,
  example,
}: NumericInputNoteProps) {
  return (
    <p id={id} className="mt-1 text-xs text-slate-500">
      半角数字のみ入力できます（例：{example}）。
    </p>
  );
}

const CategoryFieldsCard: React.FC<CategoryFieldsCardProps> = ({
  productBlueprintCategoryPath,
  categoryFields,
  mode = "edit",
  onChangeCategoryField,
}) => {
  const isEdit = mode === "edit";

  const visibility = React.useMemo(
    () =>
      getCategoryCardVisibility(
        productBlueprintCategoryPath,
      ),
    [productBlueprintCategoryPath],
  );

  /*
   * alcoholのvolumeはmodel variation側で扱う。
   * CategoryFieldsCardでは化粧品のvolumeだけを表示する。
   */
  const showCosmeticsVolume =
    visibility.showVolume && !visibility.showAlcoholContent;

  const handleChangeCategoryField = React.useCallback(
    (key: string, rawValue: string) => {
      if (!onChangeCategoryField) {
        return;
      }

      if (isNumberCategoryField(key)) {
        onChangeCategoryField(
          key,
          toCategoryNumberOrNull(rawValue),
        );
        return;
      }

      onChangeCategoryField(
        key,
        rawValue.trim() === "" ? null : rawValue,
      );
    },
    [onChangeCategoryField],
  );

  const handleChangeWashTags = React.useCallback(
    (nextTags: string[]) => {
      onChangeCategoryField?.("washTags", nextTags);
    },
    [onChangeCategoryField],
  );

  const handleChangeFit = React.useCallback(
    (fit: Fit) => {
      onChangeCategoryField?.("fit", fit);
    },
    [onChangeCategoryField],
  );

  const fitValue = getFitFieldValue(categoryFields);
  const materialValue = getStringFieldValue(
    categoryFields,
    "material",
  );
  const weightValue = getNumberFieldValue(
    categoryFields,
    "weight",
  );
  const washTagsValue = getWashTagsValue(categoryFields);

  const cardTitle =
    resolveCategoryFieldsCardTitle(visibility);

  const hasVisibleFields =
    visibility.showVintage ||
    visibility.showRegion ||
    visibility.showWeight ||
    visibility.showFit ||
    visibility.showMaterial ||
    visibility.showAlcoholContent ||
    showCosmeticsVolume ||
    visibility.showWashTags;

  if (!hasVisibleFields) {
    return null;
  }

  return (
    <Card className={`pbc${isEdit ? "" : " view-mode"}`}>
      <CardHeader className="box__header">
        <SlidersHorizontal size={16} />
        <CardTitle className="box__title">
          {cardTitle}
        </CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        {visibility.showVintage && (
          <>
            <div className="label">ヴィンテージ</div>

            <div>
              <div className="flex items-center gap-8">
                {isEdit ? (
                  <Input
                    type="number"
                    inputMode="numeric"
                    min={0}
                    step={1}
                    value={toCategoryInputValue(
                      getCategoryFieldValue(
                        categoryFields,
                        "vintage",
                      ),
                    )}
                    placeholder="例：2020"
                    onChange={(event) =>
                      handleChangeCategoryField(
                        "vintage",
                        event.target.value,
                      )
                    }
                    aria-label="ヴィンテージ"
                    aria-describedby="vintage-input-note"
                  />
                ) : (
                  <Input
                    value={toCategoryInputValue(
                      getCategoryFieldValue(
                        categoryFields,
                        "vintage",
                      ),
                    )}
                    variant="readonly"
                    readOnly
                    aria-label="ヴィンテージ"
                  />
                )}
              </div>

              {isEdit && (
                <NumericInputNote
                  id="vintage-input-note"
                  example="2020"
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
                  getCategoryFieldValue(
                    categoryFields,
                    "region",
                  ),
                )}
                onChange={(event) =>
                  handleChangeCategoryField(
                    "region",
                    event.target.value,
                  )
                }
                aria-label="地域・産地"
              />
            ) : (
              <Input
                value={toCategoryInputValue(
                  getCategoryFieldValue(
                    categoryFields,
                    "region",
                  ),
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

            <div>
              <div className="flex items-center gap-8">
                {isEdit ? (
                  <>
                    <Input
                      type="number"
                      inputMode="decimal"
                      min={0}
                      step="any"
                      value={weightValue}
                      placeholder="例：350"
                      onChange={(event) =>
                        handleChangeCategoryField(
                          "weight",
                          event.target.value,
                        )
                      }
                      aria-label="重さ"
                      aria-describedby="weight-input-note"
                    />

                    <span className="suffix">g</span>
                  </>
                ) : (
                  <>
                    <Input
                      value={
                        weightValue === ""
                          ? ""
                          : String(weightValue)
                      }
                      variant="readonly"
                      readOnly
                      aria-label="重さ"
                    />

                    <span className="suffix">g</span>
                  </>
                )}
              </div>

              {isEdit && (
                <NumericInputNote
                  id="weight-input-note"
                  example="350"
                />
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
                    type="button"
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="フィットを選択"
                  >
                    {fitValue || "フィットを選択してください。"}
                  </Button>
                </PopoverTrigger>

                <PopoverContent align="start" className="p-1">
                  {FIT_OPTIONS.map((option) => {
                    const isSelected =
                      fitValue === option.value;

                    return (
                      <button
                        key={option.value}
                        type="button"
                        className={`block w-full rounded-md px-3 py-2 text-left hover:bg-blue-50 ${
                          isSelected
                            ? "bg-blue-100 font-medium text-blue-700"
                            : ""
                        }`}
                        onClick={() =>
                          handleChangeFit(option.value)
                        }
                      >
                        {option.label}
                      </button>
                    );
                  })}
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
                onChange={(event) =>
                  handleChangeCategoryField(
                    "material",
                    event.target.value,
                  )
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

            <div>
              <div className="flex items-center gap-8">
                {isEdit ? (
                  <>
                    <Input
                      type="number"
                      inputMode="decimal"
                      min={0}
                      step="any"
                      value={toCategoryInputValue(
                        getCategoryFieldValue(
                          categoryFields,
                          "alcoholContent",
                        ),
                      )}
                      placeholder="例：15"
                      onChange={(event) =>
                        handleChangeCategoryField(
                          "alcoholContent",
                          event.target.value,
                        )
                      }
                      aria-label="アルコール度数"
                      aria-describedby="alcohol-content-input-note"
                    />

                    <span className="suffix">%</span>
                  </>
                ) : (
                  <>
                    <Input
                      value={toCategoryInputValue(
                        getCategoryFieldValue(
                          categoryFields,
                          "alcoholContent",
                        ),
                      )}
                      variant="readonly"
                      readOnly
                      aria-label="アルコール度数"
                    />

                    <span className="suffix">%</span>
                  </>
                )}
              </div>

              {isEdit && (
                <NumericInputNote
                  id="alcohol-content-input-note"
                  example="15、12.5"
                />
              )}
            </div>
          </>
        )}

        {showCosmeticsVolume && (
          <>
            <div className="label">容量</div>

            <div>
              <div className="flex items-center gap-8">
                {isEdit ? (
                  <>
                    <Input
                      type="number"
                      inputMode="decimal"
                      min={0}
                      step="any"
                      value={toCategoryInputValue(
                        getCategoryFieldValue(
                          categoryFields,
                          "volume",
                        ),
                      )}
                      placeholder="例：100"
                      onChange={(event) =>
                        handleChangeCategoryField(
                          "volume",
                          event.target.value,
                        )
                      }
                      aria-label="容量"
                      aria-describedby="volume-input-note"
                    />

                    <span className="suffix">ml</span>
                  </>
                ) : (
                  <>
                    <Input
                      value={toCategoryInputValue(
                        getCategoryFieldValue(
                          categoryFields,
                          "volume",
                        ),
                      )}
                      variant="readonly"
                      readOnly
                      aria-label="容量"
                    />

                    <span className="suffix">ml</span>
                  </>
                )}
              </div>

              {isEdit && (
                <NumericInputNote
                  id="volume-input-note"
                  example="100、250.5"
                />
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