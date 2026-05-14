// frontend/console/productBlueprint/src/presentation/hooks/create/useProductBlueprintCreateVariations.ts

import * as React from "react";

import type { ModelNumber } from "../../../../../model/src/application/modelCreateService";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
  type MeasurementOption,
  type ApparelSizeRow as SizeRow,
} from "../../../domain/entity/apparel";

import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

function newSizeRow(): SizeRow {
  return {
    id:
      typeof crypto !== "undefined" && "randomUUID" in crypto
        ? crypto.randomUUID()
        : `size-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    sizeLabel: "",

    // トップス
    length: undefined,
    width: undefined,
    chest: undefined,
    shoulder: undefined,
    sleeveLength: undefined,

    // ボトムス
    waist: undefined,
    hip: undefined,
    rise: undefined,
    inseam: undefined,
    thigh: undefined,
    hemWidth: undefined,
  };
}

export type UseProductBlueprintCreateVariationsResult = {
  isApparelCategory: boolean;
  measurementOptions: MeasurementOption[];

  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  onChangeColorInput: (value: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, rgbHex: string) => void;

  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  resetVariations: () => void;
};

export function useProductBlueprintCreateVariations(
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null,
): UseProductBlueprintCreateVariationsResult {
  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    {},
  );

  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumber[]>([]);

  const categoryCode = React.useMemo(
    () => String(productBlueprintCategory?.code ?? "").trim(),
    [productBlueprintCategory],
  );

  const isApparelCategory = React.useMemo(
    () => isApparelCategoryCode(categoryCode),
    [categoryCode],
  );

  const measurementOptions: MeasurementOption[] = React.useMemo(() => {
    if (!isApparelCategoryCode(categoryCode)) {
      return [];
    }

    return APPAREL_CATEGORY_MEASUREMENT_OPTIONS[categoryCode] ?? [];
  }, [categoryCode]);

  const resetVariations = React.useCallback(() => {
    setColors([]);
    setColorInput("");
    setColorRgbMap({});
    setSizes([]);
    setModelNumbers([]);
  }, []);

  React.useEffect(() => {
    if (!isApparelCategory) {
      resetVariations();
    }
  }, [isApparelCategory, resetVariations]);

  const onAddColor = React.useCallback(() => {
    if (!isApparelCategory) {
      return;
    }

    const value = colorInput.trim();

    if (!value || colors.includes(value)) {
      return;
    }

    setColors((prev) => [...prev, value]);
    setColorInput("");
  }, [isApparelCategory, colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    const colorName = name.trim();

    if (!colorName) {
      return;
    }

    setColors((prev) => prev.filter((color) => color !== colorName));

    setColorRgbMap((prev) => {
      const next = { ...prev };
      delete next[colorName];
      return next;
    });

    setModelNumbers((prev) =>
      prev.filter((modelNumber) => modelNumber.color !== colorName),
    );
  }, []);

  const onChangeColorRgb = React.useCallback(
    (name: string, rgbHex: string) => {
      const key = name.trim();

      if (!key) {
        return;
      }

      setColorRgbMap((prev) => ({
        ...prev,
        [key]: rgbHex,
      }));
    },
    [],
  );

  const onAddSize = React.useCallback(() => {
    if (!isApparelCategory) {
      return;
    }

    setSizes((prev) => [...prev, newSizeRow()]);
  }, [isApparelCategory]);

  const onRemoveSize = React.useCallback(
    (id: string) => {
      const target = sizes.find((size) => size.id === id);
      const labelRaw = target?.sizeLabel;
      const sizeLabel =
        typeof labelRaw === "string"
          ? labelRaw.trim()
          : String(labelRaw ?? "").trim();

      setSizes((prev) => prev.filter((size) => size.id !== id));

      if (sizeLabel) {
        setModelNumbers((prev) =>
          prev.filter((modelNumber) => modelNumber.size !== sizeLabel),
        );
      }
    },
    [sizes],
  );

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      const safePatch: Partial<Omit<SizeRow, "id">> = { ...patch };

      const clampField = (key: keyof Omit<SizeRow, "id">) => {
        const value = safePatch[key];

        if (typeof value === "number") {
          safePatch[key] = (value < 0 ? 0 : value) as never;
        }
      };

      // トップス
      clampField("length");
      clampField("width");
      clampField("chest");
      clampField("shoulder");
      clampField("sleeveLength");

      // ボトムス
      clampField("waist");
      clampField("hip");
      clampField("rise");
      clampField("inseam");
      clampField("thigh");
      clampField("hemWidth");

      const prevRow = sizes.find((size) => size.id === id);
      const prevLabelRaw = prevRow?.sizeLabel;
      const prevLabel =
        typeof prevLabelRaw === "string"
          ? prevLabelRaw.trim()
          : String(prevLabelRaw ?? "").trim();

      const nextLabelRaw = safePatch.sizeLabel;
      const nextLabel =
        typeof nextLabelRaw === "string"
          ? nextLabelRaw.trim()
          : nextLabelRaw == null
            ? null
            : String(nextLabelRaw).trim();

      if (nextLabel !== null && nextLabel !== prevLabel) {
        if (!nextLabel) {
          if (prevLabel) {
            setModelNumbers((prev) =>
              prev.filter((modelNumber) => modelNumber.size !== prevLabel),
            );
          }
        } else if (prevLabel) {
          setModelNumbers((prev) =>
            prev.map((modelNumber) =>
              modelNumber.size === prevLabel
                ? { ...modelNumber, size: nextLabel }
                : modelNumber,
            ),
          );
        }
      }

      setSizes((prev) =>
        prev.map((size) => (size.id === id ? { ...size, ...safePatch } : size)),
      );
    },
    [sizes],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const index = prev.findIndex(
          (modelNumber) =>
            modelNumber.size === sizeLabel && modelNumber.color === color,
        );

        const trimmed = nextCode.trim();

        if (!trimmed) {
          if (index === -1) {
            return prev;
          }

          const copy = [...prev];
          copy.splice(index, 1);
          return copy;
        }

        const next: ModelNumber = {
          size: sizeLabel,
          color,
          code: trimmed,
        };

        if (index === -1) {
          return [...prev, next];
        }

        const copy = [...prev];
        copy[index] = next;
        return copy;
      });
    },
    [],
  );

  React.useEffect(() => {
    const validColors = new Set(
      colors.map((color) => color.trim()).filter(Boolean),
    );

    const validSizes = new Set(
      sizes
        .map((size) => size.sizeLabel)
        .map((value) =>
          typeof value === "string" ? value.trim() : String(value ?? "").trim(),
        )
        .filter(Boolean),
    );

    setModelNumbers((prev) =>
      prev.filter(
        (modelNumber) =>
          validColors.has(modelNumber.color) && validSizes.has(modelNumber.size),
      ),
    );
  }, [colors, sizes]);

  return {
    isApparelCategory,
    measurementOptions,

    colors,
    colorInput,
    colorRgbMap,
    sizes,
    modelNumbers,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,

    resetVariations,
  };
}