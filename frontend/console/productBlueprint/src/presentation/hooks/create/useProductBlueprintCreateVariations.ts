// frontend/console/productBlueprint/src/presentation/hooks/create/useProductBlueprintCreateVariations.ts

import * as React from "react";

import type {
  AlcoholModelNumber,
  ModelNumber,
  VolumeRow,
} from "../../../../../model/src/application/modelCreateService";

import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
  type MeasurementOption,
  type ApparelSizeRow as SizeRow,
} from "../../../domain/entity/apparel";

import { isAlcoholCategoryCode } from "../../../domain/entity/alcohol";

import type { ProductBlueprintCategorySnapshot } from "../../../domain/entity/productBlueprintCategory";

function createId(prefix: string): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function newSizeRow(): SizeRow {
  return {
    id: createId("size"),
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

function newVolumeRow(): VolumeRow {
  return {
    id: createId("volume"),
    volumeValue: 0,
    volumeUnit: "ml",
  };
}

function toVolumeLabel(row: Pick<VolumeRow, "volumeValue" | "volumeUnit">): string {
  const value =
    typeof row.volumeValue === "number" && Number.isFinite(row.volumeValue)
      ? row.volumeValue
      : 0;

  const unit = String(row.volumeUnit ?? "").trim() || "ml";

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

export type UseProductBlueprintCreateVariationsResult = {
  isApparelCategory: boolean;
  isAlcoholCategory: boolean;
  measurementOptions: MeasurementOption[];

  colors: string[];
  colorInput: string;
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  /**
   * alcohol model variation 用。
   * volume は productBlueprint.categoryFields ではなく model domain 側で扱う。
   */
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];

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

  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: Partial<Omit<VolumeRow, "id">>) => void;
  onChangeAlcoholModelNumber: (
    volumeLabel: string,
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

  const [volumes, setVolumes] = React.useState<VolumeRow[]>([]);
  const [alcoholModelNumbers, setAlcoholModelNumbers] = React.useState<
    AlcoholModelNumber[]
  >([]);

  const categoryCode = React.useMemo(
    () => String(productBlueprintCategory?.code ?? "").trim(),
    [productBlueprintCategory],
  );

  const isApparelCategory = React.useMemo(
    () => isApparelCategoryCode(categoryCode),
    [categoryCode],
  );

  const isAlcoholCategory = React.useMemo(
    () => isAlcoholCategoryCode(categoryCode),
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
    setVolumes([]);
    setAlcoholModelNumbers([]);
  }, []);

  React.useEffect(() => {
    if (!isApparelCategory && !isAlcoholCategory) {
      resetVariations();
      return;
    }

    if (!isApparelCategory) {
      setColors([]);
      setColorInput("");
      setColorRgbMap({});
      setSizes([]);
      setModelNumbers([]);
    }

    if (!isAlcoholCategory) {
      setVolumes([]);
      setAlcoholModelNumbers([]);
    }
  }, [isApparelCategory, isAlcoholCategory, resetVariations]);

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
          kind: "apparel",
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

  const onAddVolume = React.useCallback(() => {
    if (!isAlcoholCategory) {
      return;
    }

    setVolumes((prev) => [...prev, newVolumeRow()]);
  }, [isAlcoholCategory]);

  const onRemoveVolume = React.useCallback(
    (id: string) => {
      const target = volumes.find((volume) => volume.id === id);
      const targetLabel = target ? toVolumeLabel(target) : "";

      setVolumes((prev) => prev.filter((volume) => volume.id !== id));

      if (targetLabel) {
        setAlcoholModelNumbers((prev) =>
          prev.filter(
            (modelNumber) => modelNumber.volumeLabel !== targetLabel,
          ),
        );
      }
    },
    [volumes],
  );

  const onChangeVolume = React.useCallback(
    (id: string, patch: Partial<Omit<VolumeRow, "id">>) => {
      const safePatch: Partial<Omit<VolumeRow, "id">> = { ...patch };

      if (
        typeof safePatch.volumeValue === "number" &&
        safePatch.volumeValue < 0
      ) {
        safePatch.volumeValue = 0;
      }

      if (typeof safePatch.volumeUnit === "string") {
        safePatch.volumeUnit = safePatch.volumeUnit.trim() || "ml";
      }

      const prevRow = volumes.find((volume) => volume.id === id);
      const prevLabel = prevRow ? toVolumeLabel(prevRow) : "";

      const nextRow = prevRow ? { ...prevRow, ...safePatch } : null;
      const nextLabel = nextRow ? toVolumeLabel(nextRow) : "";

      if (prevLabel && nextLabel && prevLabel !== nextLabel) {
        setAlcoholModelNumbers((prev) =>
          prev.map((modelNumber) =>
            modelNumber.volumeLabel === prevLabel
              ? {
                  ...modelNumber,
                  volume: {
                    value: nextRow?.volumeValue ?? 0,
                    unit: nextRow?.volumeUnit ?? "ml",
                  },
                  volumeLabel: nextLabel,
                }
              : modelNumber,
          ),
        );
      }

      if (prevLabel && !nextLabel) {
        setAlcoholModelNumbers((prev) =>
          prev.filter((modelNumber) => modelNumber.volumeLabel !== prevLabel),
        );
      }

      setVolumes((prev) =>
        prev.map((volume) =>
          volume.id === id ? { ...volume, ...safePatch } : volume,
        ),
      );
    },
    [volumes],
  );

  const onChangeAlcoholModelNumber = React.useCallback(
    (volumeLabel: string, nextCode: string) => {
      const label = volumeLabel.trim();

      if (!label) {
        return;
      }

      const volumeRow = volumes.find((volume) => toVolumeLabel(volume) === label);

      if (!volumeRow) {
        return;
      }

      setAlcoholModelNumbers((prev) => {
        const index = prev.findIndex(
          (modelNumber) => modelNumber.volumeLabel === label,
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

        const next: AlcoholModelNumber = {
          kind: "alcohol",
          volume: {
            value: volumeRow.volumeValue,
            unit: volumeRow.volumeUnit,
          },
          volumeLabel: label,
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
    [volumes],
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

  React.useEffect(() => {
    const validVolumeLabels = new Set(
      volumes.map(toVolumeLabel).filter(Boolean),
    );

    setAlcoholModelNumbers((prev) =>
      prev.filter((modelNumber) =>
        validVolumeLabels.has(modelNumber.volumeLabel),
      ),
    );
  }, [volumes]);

  return {
    isApparelCategory,
    isAlcoholCategory,
    measurementOptions,

    colors,
    colorInput,
    colorRgbMap,
    sizes,
    modelNumbers,

    volumes,
    alcoholModelNumbers,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onAddSize,
    onRemoveSize,
    onChangeSize,
    onChangeModelNumber,

    onAddVolume,
    onRemoveVolume,
    onChangeVolume,
    onChangeAlcoholModelNumber,

    resetVariations,
  };
}