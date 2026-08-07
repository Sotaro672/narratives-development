// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateCategoryFields.ts
import * as React from "react";
import type {
  CategoryFieldValue,
  CategoryFieldValues,
} from "../../../domain/productBlueprintCategory";

export type UseProductBlueprintCreateCategoryFieldsResult = {
  categoryFields: CategoryFieldValues;
  onChangeCategoryField: (key: string, value: CategoryFieldValue) => void;
  resetCategoryFields: () => void;
};

export function useProductBlueprintCreateCategoryFields(): UseProductBlueprintCreateCategoryFieldsResult {
  const [categoryFields, setCategoryFields] = React.useState<CategoryFieldValues>({});
  const onChangeCategoryField = React.useCallback(
    (key: string, value: CategoryFieldValue) => {
      setCategoryFields((previous) => ({
        ...previous,
        [key]: value,
      }));
    },
    [],
  );
  const resetCategoryFields = React.useCallback(() => {
    setCategoryFields({});
  }, []);
  return {
    categoryFields,
    onChangeCategoryField,
    resetCategoryFields,
  };
}