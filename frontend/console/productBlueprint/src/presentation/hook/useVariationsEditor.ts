// frontend/console/productBlueprint/src/presentation/hook/useVariationsEditor.ts
import * as React from "react";

import type {
  SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import { useModelCard } from "../../../../model/src/presentation/hook/useModelCard";

/**
 * UI state derived from ModelVariation list (already mapped by variationMapper, etc.)
 */
export type VariationsUiState = {
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  /** color 名 → rgb hex (#rrggbb) */
  colorRgbMap: Record<string, string>;
};

export type UseVariationsEditorResult = {
  // state
  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];
  colorRgbMap: Record<string, string>;

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

  // model number
  onChangeModelNumber: (sizeLabel: string, color: string, nextCode: string) => void;
};

/**
 * Presentation-level editor state for variations (colors/sizes/modelNumbers/colorRgbMap).
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

  const setFromUiState = React.useCallback((next: VariationsUiState) => {
    setColors(next.colors ?? []);
    setSizes(next.sizes ?? []);
    setModelNumbers(next.modelNumbers ?? []);
    setColorRgbMap(next.colorRgbMap ?? {});
    setColorInput("");
  }, []);

  // ---------------------------------
  // ModelNumberCard 用ロジックは useModelCard に委譲
  // ---------------------------------
  const { getCode, onChangeModelNumber: uiOnChangeModelNumber } = useModelCard({
    sizes,
    colors,
    modelNumbers: modelNumbers as any,
    colorRgbMap,
  });

  // ---------------------------------
  // Internal: modelNumbers state update
  // ---------------------------------
  const patchModelNumberState = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      setModelNumbers((prev) => {
        const idx = prev.findIndex((m) => m.size === sizeLabel && m.color === color);
        const trimmed = nextCode.trim();

        // empty => remove
        if (!trimmed) {
          if (idx === -1) return prev;
          const copy = [...prev];
          copy.splice(idx, 1);
          return copy;
        }

        const next: ModelNumberRow = { size: sizeLabel, color, code: trimmed };

        if (idx === -1) return [...prev, next];

        const copy = [...prev];
        copy[idx] = next;
        return copy;
      });
    },
    [],
  );

  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
      // keep existing behavior: call UI helper, then update local state
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
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    const key = name.trim();
    if (!key) return;

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
    if (!colorName || !value) return;

    if (!value.startsWith("#")) value = `#${value}`;

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
          setModelNumbers((prevMN) => prevMN.filter((m) => m.size !== sizeLabel));
        }
      }

      return next;
    });
  }, []);

  const onAddSize = React.useCallback(() => {
    setSizes((prev) => {
      const nextNum =
        prev.reduce((max, row) => {
          const n = Number(row.id);
          if (Number.isNaN(n)) return max;
          return n > max ? n : max;
        }, 0) + 1;

      const next: SizeRow = {
        id: String(nextNum),
        sizeLabel: "",
      } as SizeRow;

      return [...prev, next];
    });
  }, []);

  const onChangeSize = React.useCallback(
    (id: string, patch: Partial<Omit<SizeRow, "id">>) => {
      const safePatch: Partial<Omit<SizeRow, "id">> = { ...patch };

      const clampField = (key: keyof Omit<SizeRow, "id">) => {
        const v = safePatch[key];
        if (typeof v === "number") {
          safePatch[key] = (v < 0 ? 0 : v) as any;
        }
      };

      // minimum set (matches existing behavior)
      clampField("chest");
      clampField("waist");
      clampField("length");
      clampField("shoulder");

      setSizes((prev) => prev.map((s) => (s.id === id ? { ...s, ...safePatch } : s)));
    },
    [],
  );

  return {
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

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
  };
}
