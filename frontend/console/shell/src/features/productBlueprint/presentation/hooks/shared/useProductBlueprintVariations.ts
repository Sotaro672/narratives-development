// frontend/console/shell/src/features/productBlueprint/presentation/hooks/shared/useProductBlueprintVariations.ts

import * as React from "react";
import type {
  AlcoholModelNumber,
  ApparelModelNumber,
  SizePatch,
  SizeRow,
  VolumeRow,
} from "../../../../model/application/modelCreateService";
import {
  APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
  isApparelCategoryCode,
  type MeasurementOption,
} from "../../../../../shared/types/apparel";
import { isAlcoholCategoryCode } from "../../../domain/alcohol";
import type { ProductBlueprintCategorySnapshot } from "../../../domain/productBlueprintCategory";

type VolumePatch = Partial<Omit<VolumeRow, "id">>;

export type ProductBlueprintVariationsState = {
  colors: string[];
  colorRgbMap: Record<string, string>;
  sizes: SizeRow[];
  modelNumbers: ApparelModelNumber[];
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];
};

export type UseProductBlueprintVariationsParams = {
  productBlueprintCategory?: ProductBlueprintCategorySnapshot | null;
  initialState?: Partial<ProductBlueprintVariationsState> | null;
};

export type UseProductBlueprintVariationsResult = ProductBlueprintVariationsState & {
  categoryCode: string;
  isApparelCategory: boolean;
  isAlcoholCategory: boolean;
  measurementOptions: MeasurementOption[];
  colorInput: string;
  getCode: (sizeLabel: string, color: string) => string;
  setFromUiState: (
    next: Partial<ProductBlueprintVariationsState> | null | undefined,
  ) => void;
  resetVariations: () => void;
  onChangeColorInput: (value: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, rgbHex: string) => void;
  onAddSize: () => void;
  onRemoveSize: (id: string) => void;
  onChangeSize: (id: string, patch: SizePatch) => void;
  onChangeModelNumber: (sizeLabel: string, color: string, nextCode: string) => void;
  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: VolumePatch) => void;
  onChangeAlcoholModelNumber: (volumeLabel: string, nextCode: string) => void;
};

type ApparelMeasurementCategoryCode =
  keyof typeof APPAREL_CATEGORY_MEASUREMENT_OPTIONS;

const SIZE_NUMBER_KEYS: Array<keyof SizePatch> = [
  "length",
  "width",
  "chest",
  "shoulder",
  "sleeveLength",
  "waist",
  "hip",
  "rise",
  "inseam",
  "thigh",
  "hemWidth",
];

function isApparelMeasurementCategoryCode(
  value: string,
): value is ApparelMeasurementCategoryCode {
  return Object.prototype.hasOwnProperty.call(
    APPAREL_CATEGORY_MEASUREMENT_OPTIONS,
    value,
  );
}

function createId(prefix: string): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }

  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function createEmptyVariationsState(): ProductBlueprintVariationsState {
  return {
    colors: [],
    colorRgbMap: {},
    sizes: [],
    modelNumbers: [],
    volumes: [],
    alcoholModelNumbers: [],
  };
}

function createSizeRow(): SizeRow {
  return {
    id: createId("size"),
    sizeLabel: "",
    length: undefined,
    width: undefined,
    chest: undefined,
    shoulder: undefined,
    sleeveLength: undefined,
    waist: undefined,
    hip: undefined,
    rise: undefined,
    inseam: undefined,
    thigh: undefined,
    hemWidth: undefined,
  };
}

function createVolumeRow(): VolumeRow {
  return {
    id: createId("volume"),
    volumeValue: 0,
    volumeUnit: "ml",
  };
}

function normalizeString(value: unknown): string {
  return String(value ?? "").trim();
}

function normalizeNonNegativeNumber(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }

  return value < 0 ? 0 : value;
}

function normalizeVolumeUnit(value: unknown): string {
  return normalizeString(value) || "ml";
}

function toVolumeLabel(
  volume: Pick<VolumeRow, "volumeValue" | "volumeUnit">,
): string {
  const value = normalizeNonNegativeNumber(volume.volumeValue);
  const unit = normalizeVolumeUnit(volume.volumeUnit);

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function toAlcoholModelNumberVolumeLabel(
  modelNumber: AlcoholModelNumber,
): string {
  return toVolumeLabel({
    volumeValue: modelNumber.volume.value,
    volumeUnit: modelNumber.volume.unit,
  });
}

function normalizeColors(values: unknown): string[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const color = normalizeString(value);

    if (!color || seen.has(color)) {
      continue;
    }

    seen.add(color);
    result.push(color);
  }

  return result;
}

function normalizeSizePatch(patch: SizePatch): SizePatch {
  const next: SizePatch = {
    ...patch,
  };

  for (const key of SIZE_NUMBER_KEYS) {
    const value = next[key];

    if (typeof value === "number") {
      next[key] = normalizeNonNegativeNumber(value) as never;
    }
  }

  if (next.sizeLabel !== undefined) {
    next.sizeLabel = String(next.sizeLabel ?? "");
  }

  return next;
}

function normalizeVolumePatch(patch: VolumePatch): VolumePatch {
  const next: VolumePatch = {
    ...patch,
  };

  if (next.volumeValue !== undefined) {
    next.volumeValue = normalizeNonNegativeNumber(next.volumeValue);
  }

  if (next.volumeUnit !== undefined) {
    next.volumeUnit = normalizeVolumeUnit(next.volumeUnit);
  }

  return next;
}

function normalizeVariationsState(
  input: Partial<ProductBlueprintVariationsState> | null | undefined,
): ProductBlueprintVariationsState {
  const colors = normalizeColors(input?.colors);
  const validColors = new Set(colors);

  const sizes = Array.isArray(input?.sizes)
    ? input.sizes.map((size) => ({
        ...size,
      }))
    : [];

  const validSizeLabels = new Set(
    sizes
      .map((size) => normalizeString(size.sizeLabel))
      .filter(Boolean),
  );

  const modelNumbers: ApparelModelNumber[] =
    Array.isArray(input?.modelNumbers)
      ? input.modelNumbers
          .map(
            (modelNumber): ApparelModelNumber => ({
              ...modelNumber,
              size: normalizeString(modelNumber.size),
              color: normalizeString(modelNumber.color),
              code: normalizeString(modelNumber.code),
            }),
          )
          .filter(
            (modelNumber) =>
              validSizeLabels.has(modelNumber.size) &&
              validColors.has(modelNumber.color),
          )
      : [];

  const colorRgbMap: Record<string, string> = {};

  if (input?.colorRgbMap && typeof input.colorRgbMap === "object") {
    for (const [color, value] of Object.entries(input.colorRgbMap)) {
      const normalizedColor = normalizeString(color);
      const normalizedValue = normalizeString(value);

      if (
        !normalizedColor ||
        !normalizedValue ||
        !validColors.has(normalizedColor)
      ) {
        continue;
      }

      colorRgbMap[normalizedColor] = normalizedValue.startsWith("#")
        ? normalizedValue
        : `#${normalizedValue}`;
    }
  }

  const volumes = Array.isArray(input?.volumes)
    ? input.volumes.map((volume) => ({
        ...volume,
        volumeValue: normalizeNonNegativeNumber(volume.volumeValue),
        volumeUnit: normalizeVolumeUnit(volume.volumeUnit),
      }))
    : [];

  const validVolumeLabels = new Set(
    volumes.map(toVolumeLabel).filter(Boolean),
  );

  const alcoholModelNumbers = Array.isArray(input?.alcoholModelNumbers)
    ? input.alcoholModelNumbers
        .map(
          (modelNumber): AlcoholModelNumber => ({
            kind: "alcohol",
            volume: {
              value: normalizeNonNegativeNumber(modelNumber.volume.value),
              unit: normalizeVolumeUnit(modelNumber.volume.unit),
            },
            code: normalizeString(modelNumber.code),
          }),
        )
        .filter((modelNumber) =>
          validVolumeLabels.has(
            toAlcoholModelNumberVolumeLabel(modelNumber),
          ),
        )
    : [];

  return {
    colors,
    colorRgbMap,
    sizes,
    modelNumbers,
    volumes,
    alcoholModelNumbers,
  };
}

export function useProductBlueprintVariations(
  params: UseProductBlueprintVariationsParams = {},
): UseProductBlueprintVariationsResult {
  const initialStateRef = React.useRef<ProductBlueprintVariationsState>(
    normalizeVariationsState(params.initialState),
  );

  const [state, setState] = React.useState<ProductBlueprintVariationsState>(
    initialStateRef.current,
  );

  const [colorInput, setColorInput] = React.useState("");

  const categoryCode = React.useMemo(
    () => normalizeString(params.productBlueprintCategory?.code),
    [params.productBlueprintCategory?.code],
  );

  const isApparelCategory = React.useMemo(
    () => isApparelCategoryCode(categoryCode),
    [categoryCode],
  );

  const isAlcoholCategory = React.useMemo(
    () => isAlcoholCategoryCode(categoryCode),
    [categoryCode],
  );

  const measurementOptions = React.useMemo<MeasurementOption[]>(() => {
    if (!isApparelMeasurementCategoryCode(categoryCode)) {
      return [];
    }

    return APPAREL_CATEGORY_MEASUREMENT_OPTIONS[categoryCode] ?? [];
  }, [categoryCode]);

  const getCode = React.useCallback(
    (sizeLabel: string, color: string): string => {
      const normalizedSize = normalizeString(sizeLabel);
      const normalizedColor = normalizeString(color);

      return (
        state.modelNumbers.find(
          (modelNumber) =>
            modelNumber.size === normalizedSize &&
            modelNumber.color === normalizedColor,
        )?.code ?? ""
      );
    },
    [state.modelNumbers],
  );

  const setFromUiState = React.useCallback(
    (
      next: Partial<ProductBlueprintVariationsState> | null | undefined,
    ) => {
      setState(normalizeVariationsState(next));
      setColorInput("");
    },
    [],
  );

  const resetVariations = React.useCallback(() => {
    setState(createEmptyVariationsState());
    setColorInput("");
  }, []);

  const onAddColor = React.useCallback(() => {
    if (!isApparelCategory) {
      return;
    }

    const color = normalizeString(colorInput);

    if (!color) {
      return;
    }

    setState((previous) => {
      if (previous.colors.includes(color)) {
        return previous;
      }

      return {
        ...previous,
        colors: [...previous.colors, color],
      };
    });

    setColorInput("");
  }, [colorInput, isApparelCategory]);

  const onRemoveColor = React.useCallback((name: string) => {
    const color = normalizeString(name);

    if (!color) {
      return;
    }

    setState((previous) => {
      const nextRgbMap = {
        ...previous.colorRgbMap,
      };

      delete nextRgbMap[color];

      return {
        ...previous,
        colors: previous.colors.filter((value) => value !== color),
        colorRgbMap: nextRgbMap,
        modelNumbers: previous.modelNumbers.filter(
          (modelNumber) => modelNumber.color !== color,
        ),
      };
    });
  }, []);

  const onChangeColorRgb = React.useCallback(
    (name: string, rgbHex: string) => {
      const color = normalizeString(name);
      const rawValue = normalizeString(rgbHex);

      if (!color) {
        return;
      }

      setState((previous) => {
        const nextRgbMap = {
          ...previous.colorRgbMap,
        };

        if (!rawValue) {
          delete nextRgbMap[color];
        } else {
          nextRgbMap[color] = rawValue.startsWith("#")
            ? rawValue
            : `#${rawValue}`;
        }

        return {
          ...previous,
          colorRgbMap: nextRgbMap,
        };
      });
    },
    [],
  );

  const onAddSize = React.useCallback(() => {
    if (!isApparelCategory) {
      return;
    }

    setState((previous) => ({
      ...previous,
      sizes: [...previous.sizes, createSizeRow()],
    }));
  }, [isApparelCategory]);

  const onRemoveSize = React.useCallback((id: string) => {
    setState((previous) => {
      const target = previous.sizes.find((size) => size.id === id);
      const sizeLabel = normalizeString(target?.sizeLabel);

      return {
        ...previous,
        sizes: previous.sizes.filter((size) => size.id !== id),
        modelNumbers: sizeLabel
          ? previous.modelNumbers.filter(
              (modelNumber) => modelNumber.size !== sizeLabel,
            )
          : previous.modelNumbers,
      };
    });
  }, []);

  const onChangeSize = React.useCallback(
    (id: string, patch: SizePatch) => {
      const safePatch = normalizeSizePatch(patch);

      setState((previous) => {
        const previousRow = previous.sizes.find((size) => size.id === id);

        if (!previousRow) {
          return previous;
        }

        const previousLabel = normalizeString(previousRow.sizeLabel);
        const nextLabel =
          safePatch.sizeLabel === undefined
            ? previousLabel
            : normalizeString(safePatch.sizeLabel);

        let nextModelNumbers = previous.modelNumbers;

        if (previousLabel !== nextLabel) {
          if (!nextLabel) {
            nextModelNumbers = previous.modelNumbers.filter(
              (modelNumber) => modelNumber.size !== previousLabel,
            );
          } else if (previousLabel) {
            nextModelNumbers = previous.modelNumbers.map((modelNumber) =>
              modelNumber.size === previousLabel
                ? {
                    ...modelNumber,
                    size: nextLabel,
                  }
                : modelNumber,
            );
          }
        }

        return {
          ...previous,
          sizes: previous.sizes.map((size) =>
            size.id === id
              ? {
                  ...size,
                  ...safePatch,
                }
              : size,
          ),
          modelNumbers: nextModelNumbers,
        };
      });
    },
    [],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      if (!isApparelCategory) {
        return;
      }

      const normalizedSize = normalizeString(sizeLabel);
      const normalizedColor = normalizeString(color);
      const normalizedCode = normalizeString(nextCode);

      if (!normalizedSize || !normalizedColor) {
        return;
      }

      setState((previous) => {
        const index = previous.modelNumbers.findIndex(
          (modelNumber) =>
            modelNumber.size === normalizedSize &&
            modelNumber.color === normalizedColor,
        );

        if (!normalizedCode) {
          if (index === -1) {
            return previous;
          }

          return {
            ...previous,
            modelNumbers: previous.modelNumbers.filter(
              (_modelNumber, modelNumberIndex) =>
                modelNumberIndex !== index,
            ),
          };
        }

        const next: ApparelModelNumber = {
          kind: "apparel",
          size: normalizedSize,
          color: normalizedColor,
          code: normalizedCode,
        };

        if (index === -1) {
          return {
            ...previous,
            modelNumbers: [...previous.modelNumbers, next],
          };
        }

        const nextModelNumbers = [...previous.modelNumbers];
        nextModelNumbers[index] = next;

        return {
          ...previous,
          modelNumbers: nextModelNumbers,
        };
      });
    },
    [isApparelCategory],
  );

  const onAddVolume = React.useCallback(() => {
    if (!isAlcoholCategory) {
      return;
    }

    setState((previous) => ({
      ...previous,
      volumes: [...previous.volumes, createVolumeRow()],
    }));
  }, [isAlcoholCategory]);

  const onRemoveVolume = React.useCallback((id: string) => {
    setState((previous) => {
      const target = previous.volumes.find((volume) => volume.id === id);
      const targetLabel = target ? toVolumeLabel(target) : "";

      return {
        ...previous,
        volumes: previous.volumes.filter((volume) => volume.id !== id),
        alcoholModelNumbers: targetLabel
          ? previous.alcoholModelNumbers.filter(
              (modelNumber) =>
                toAlcoholModelNumberVolumeLabel(modelNumber) !== targetLabel,
            )
          : previous.alcoholModelNumbers,
      };
    });
  }, []);

  const onChangeVolume = React.useCallback(
    (id: string, patch: VolumePatch) => {
      const safePatch = normalizeVolumePatch(patch);

      setState((previous) => {
        const previousRow = previous.volumes.find(
          (volume) => volume.id === id,
        );

        if (!previousRow) {
          return previous;
        }

        const nextRow: VolumeRow = {
          ...previousRow,
          ...safePatch,
        };

        const previousLabel = toVolumeLabel(previousRow);
        const nextLabel = toVolumeLabel(nextRow);
        let nextModelNumbers = previous.alcoholModelNumbers;

        if (previousLabel !== nextLabel) {
          if (!nextLabel) {
            nextModelNumbers = previous.alcoholModelNumbers.filter(
              (modelNumber) =>
                toAlcoholModelNumberVolumeLabel(modelNumber) !== previousLabel,
            );
          } else if (previousLabel) {
            nextModelNumbers = previous.alcoholModelNumbers.map(
              (modelNumber) =>
                toAlcoholModelNumberVolumeLabel(modelNumber) === previousLabel
                  ? {
                      ...modelNumber,
                      volume: {
                        value: nextRow.volumeValue,
                        unit: nextRow.volumeUnit,
                      },
                    }
                  : modelNumber,
            );
          }
        }

        return {
          ...previous,
          volumes: previous.volumes.map((volume) =>
            volume.id === id ? nextRow : volume,
          ),
          alcoholModelNumbers: nextModelNumbers,
        };
      });
    },
    [],
  );

  const onChangeAlcoholModelNumber = React.useCallback(
    (volumeLabel: string, nextCode: string) => {
      if (!isAlcoholCategory) {
        return;
      }

      const normalizedLabel = normalizeString(volumeLabel);
      const normalizedCode = normalizeString(nextCode);

      if (!normalizedLabel) {
        return;
      }

      setState((previous) => {
        const volume = previous.volumes.find(
          (row) => toVolumeLabel(row) === normalizedLabel,
        );

        if (!volume) {
          return previous;
        }

        const index = previous.alcoholModelNumbers.findIndex(
          (modelNumber) =>
            toAlcoholModelNumberVolumeLabel(modelNumber) === normalizedLabel,
        );

        if (!normalizedCode) {
          if (index === -1) {
            return previous;
          }

          return {
            ...previous,
            alcoholModelNumbers: previous.alcoholModelNumbers.filter(
              (_modelNumber, modelNumberIndex) =>
                modelNumberIndex !== index,
            ),
          };
        }

        const next: AlcoholModelNumber = {
          kind: "alcohol",
          volume: {
            value: volume.volumeValue,
            unit: volume.volumeUnit,
          },
          code: normalizedCode,
        };

        if (index === -1) {
          return {
            ...previous,
            alcoholModelNumbers: [
              ...previous.alcoholModelNumbers,
              next,
            ],
          };
        }

        const nextModelNumbers = [...previous.alcoholModelNumbers];
        nextModelNumbers[index] = next;

        return {
          ...previous,
          alcoholModelNumbers: nextModelNumbers,
        };
      });
    },
    [isAlcoholCategory],
  );

  return {
    categoryCode,
    isApparelCategory,
    isAlcoholCategory,
    measurementOptions,
    colors: state.colors,
    colorInput,
    colorRgbMap: state.colorRgbMap,
    sizes: state.sizes,
    modelNumbers: state.modelNumbers,
    volumes: state.volumes,
    alcoholModelNumbers: state.alcoholModelNumbers,
    getCode,
    setFromUiState,
    resetVariations,
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
  };
}