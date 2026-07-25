// frontend/console/productBlueprint/src/presentation/hooks/detail/useVariationsEditor.ts

import * as React from "react";

import type {
  ApparelModelNumberRow as ModelNumberRow,
  ApparelSizeRow as SizeRow,
} from "../../../domain/entity/apparel";

import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../../model/src/application/modelCreateService";

import { useModelCard } from "../../../../../model/src/presentation/hook/useModelCard";

/**
 * UI state derived from ModelVariation list (already mapped by variationMapper, etc.)
 */
export type VariationsUiState = {
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  /** color 名 → rgb hex (#rrggbb) */
  colorRgbMap: Record<string, string>;

  /**
   * alcohol model variation 用。
   * volume は ProductBlueprint.categoryFields ではなく model domain 側で扱う。
   */
  volumes?: VolumeRow[];
  alcoholModelNumbers?: AlcoholModelNumber[];
};

export type UseVariationsEditorResult = {
  // state
  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  colorRgbMap: Record<string, string>;

  // alcohol state
  volumes: VolumeRow[];
  alcoholModelNumbers: AlcoholModelNumber[];

  // model card helper
  getCode: (sizeLabel: string, color: string) => string;

  // initialize/replace (e.g. after fetching variations)
  setFromUiState: (next: VariationsUiState) => void;

  // color
  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onChangeColorRgb: (name: string, hex: string) => void;

  // size
  onRemoveSize: (id: string) => void;
  onAddSize: () => void;
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  // apparel model number
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  // alcohol volume
  onAddVolume: () => void;
  onRemoveVolume: (id: string) => void;
  onChangeVolume: (id: string, patch: Partial<Omit<VolumeRow, "id">>) => void;

  // alcohol model number
  onChangeAlcoholModelNumber: (
    volumeLabel: string,
    nextCode: string,
  ) => void;
};

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

function normalizeVolumeValue(value: unknown): number {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return 0;
  }

  return value < 0 ? 0 : value;
}

function normalizeVolumeUnit(value: unknown): string {
  const unit = String(value ?? "").trim();
  return unit || "ml";
}

function toVolumeLabel(
  row: Pick<VolumeRow, "volumeValue" | "volumeUnit">,
): string {
  const value = normalizeVolumeValue(row.volumeValue);
  const unit = normalizeVolumeUnit(row.volumeUnit);

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

/**
 * Presentation-level editor state for variations
 * (colors/sizes/modelNumbers/colorRgbMap/volumes/alcoholModelNumbers).
 *
 * - Keeps editor logic out of useProductBlueprintDetail.tsx
 * - Designed to accept state produced by variationMapper (mapVariationsToUiState)
 */
export function useVariationsEditor(
  initial?: Partial<VariationsUiState>,
): UseVariationsEditorResult {
  const [colorInput, setColorInput] = React.useState<string>("");

  const [colors, setColors] = React.useState<string[]>(initial?.colors ?? []);
  const [sizes, setSizes] = React.useState<SizeRow[]>(initial?.sizes ?? []);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumberRow[]>(
    initial?.modelNumbers ?? [],
  );
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    initial?.colorRgbMap ?? {},
  );

  const [volumes, setVolumes] = React.useState<VolumeRow[]>(
    initial?.volumes ?? [],
  );
  const [alcoholModelNumbers, setAlcoholModelNumbers] = React.useState<
    AlcoholModelNumber[]
  >(initial?.alcoholModelNumbers ?? []);

  const setFromUiState = React.useCallback((next: VariationsUiState) => {
    setColors(Array.isArray(next.colors) ? next.colors : []);
    setSizes(Array.isArray(next.sizes) ? next.sizes : []);
    setModelNumbers(Array.isArray(next.modelNumbers) ? next.modelNumbers : []);
    setColorRgbMap(next.colorRgbMap ?? {});
    setVolumes(Array.isArray(next.volumes) ? next.volumes : []);
    setAlcoholModelNumbers(
      Array.isArray(next.alcoholModelNumbers)
        ? next.alcoholModelNumbers
        : [],
    );
    setColorInput("");
  }, []);

  // ---------------------------------
  // ModelNumberCard 用ロジックは useModelCard に委譲
  // ---------------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes: Array.isArray(sizes) ? sizes : [],
    colors: Array.isArray(colors) ? colors : [],
    modelNumbers: Array.isArray(modelNumbers) ? (modelNumbers as any) : [],
    colorRgbMap: colorRgbMap ?? {},
  });

  // ---------------------------------
  // Internal: apparel modelNumbers state update
  // ---------------------------------
  const patchModelNumberState = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const idx = prev.findIndex(
          (m) => m.size === sizeLabel && m.color === color,
        );
        const trimmed = nextCode.trim();

        // empty => remove
        if (!trimmed) {
          if (idx === -1) return prev;

          const copy = [...prev];
          copy.splice(idx, 1);
          return copy;
        }

        const next: ModelNumberRow = { size: sizeLabel, color, code: trimmed };

        if (idx === -1) {
          return [...prev, next];
        }

        const copy = [...prev];
        copy[idx] = next;
        return copy;
      });
    },
    [],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      uiOnChangeModelNumber(sizeLabel, color, nextCode);
      patchModelNumberState(sizeLabel, color, nextCode);
    },
    [uiOnChangeModelNumber, patchModelNumberState],
  );

  // ---------------------------------
  // Color handlers
  // ---------------------------------
  const onAddColor = React.useCallback(() => {
    const v = colorInput.trim();

    if (!v || colors.includes(v)) {
      return;
    }

    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    const key = name.trim();

    if (!key) {
      return;
    }

    setColors((prev) => prev.filter((c) => c !== key));

    setColorRgbMap((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });

    setModelNumbers((prevMN) => prevMN.filter((m) => m.color !== key));
  }, []);

  const onChangeColorRgb = React.useCallback((name: string, hex: string) => {
    const colorName = name.trim();
    let value = String(hex ?? "").trim();

    if (!colorName || !value) {
      return;
    }

    if (!value.startsWith("#")) {
      value = `#${value}`;
    }

    setColorRgbMap((prev) => ({
      ...prev,
      [colorName]: value,
    }));
  }, []);

  // ---------------------------------
  // Size handlers
  // ---------------------------------
  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => {
      const target = prev.find((s) => s.id === id);
      const next = prev.filter((s) => s.id !== id);

      if (target) {
        const sizeLabel = (target.sizeLabel ?? "").trim();

        if (sizeLabel) {
          setModelNumbers((prevMN) =>
            prevMN.filter((m) => m.size !== sizeLabel),
          );
        }
      }

      return next;
    });
  }, []);

  const onAddSize = React.useCallback(() => {
    setSizes((prev) => [...prev, newSizeRow()]);
  }, []);

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      const safePatch: Partial<Omit<SizeRow, "id">> = { ...patch };

      const clampField = (key: keyof Omit<SizeRow, "id">) => {
        const v = safePatch[key];

        if (typeof v === "number") {
          safePatch[key] = (v < 0 ? 0 : v) as never;
        }
      };

      // model/src/domain/entity/catalog.ts の SizeRow 正規 field に合わせる
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
      const prevLabel = String(prevRow?.sizeLabel ?? "").trim();

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
        prev.map((s) => (s.id === id ? { ...s, ...safePatch } : s)),
      );
    },
    [sizes],
  );

  // ---------------------------------
  // Alcohol volume handlers
  // ---------------------------------
  const onAddVolume = React.useCallback(() => {
    setVolumes((prev) => [...prev, newVolumeRow()]);
  }, []);

  const onRemoveVolume = React.useCallback(
    (id: string) => {
      const target = volumes.find((volume) => volume.id === id);
      const targetLabel = target ? toVolumeLabel(target) : "";

      setVolumes((prev) => prev.filter((volume) => volume.id !== id));

      if (targetLabel) {
        setAlcoholModelNumbers((prev) =>
          prev.filter((modelNumber) => modelNumber.volumeLabel !== targetLabel),
        );
      }
    },
    [volumes],
  );

  const onChangeVolume = React.useCallback(
    (id: string, patch: Partial<Omit<VolumeRow, "id">>) => {
      const safePatch: Partial<Omit<VolumeRow, "id">> = { ...patch };

      if (safePatch.volumeValue !== undefined) {
        safePatch.volumeValue = normalizeVolumeValue(safePatch.volumeValue);
      }

      if (safePatch.volumeUnit !== undefined) {
        safePatch.volumeUnit = normalizeVolumeUnit(safePatch.volumeUnit);
      }

      const prevRow = volumes.find((volume) => volume.id === id);
      const prevLabel = prevRow ? toVolumeLabel(prevRow) : "";

      const nextRow: VolumeRow | null = prevRow
        ? { ...prevRow, ...safePatch }
        : null;

      const nextLabel = nextRow ? toVolumeLabel(nextRow) : "";

      if (prevLabel && nextRow && nextLabel && prevLabel !== nextLabel) {
        setAlcoholModelNumbers((prev) =>
          prev.map((modelNumber) =>
            modelNumber.volumeLabel === prevLabel
              ? {
                  ...modelNumber,
                  volume: {
                    value: nextRow.volumeValue,
                    unit: nextRow.volumeUnit,
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

  // ---------------------------------
  // Cleanup invalid model numbers
  // ---------------------------------
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
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    volumes,
    alcoholModelNumbers,

    getCode,

    setFromUiState,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onRemoveSize,
    onAddSize,
    onChangeSize,

    onChangeModelNumber,

    onAddVolume,
    onRemoveVolume,
    onChangeVolume,
    onChangeAlcoholModelNumber,
  };
}