//frontend\console\model\src\presentation\hook\useModelCard.tsx
import * as React from "react";

// hook 用の型は application 層にまとめる
import type {
  ModelNumber,
  SizeRow,
  UseModelCardParams,
  UseModelCardResult,
  UseSizeVariationCardParams,
  UseSizeVariationCardResult,
  SizePatch,
} from "../../application/modelCreateService";

/** SizeRow は hook モジュールからも参照できるように application 層の型を再エクスポート */
export type { SizeRow };

/* =========================================================
 * ModelNumber 用 hook ロジック
 * =======================================================*/

/** 内部キー生成（sizeLabel + color 用） */
const makeKey = (sizeLabel: string, color: string) =>
  `${sizeLabel}__${color}`;

/**
 * ModelNumberCard のロジックをすべてこの hook に集約する
 *
 * - UI ローカル状態（codeMap）を管理
 * - application 層から渡された onChangeModelNumber も同時に呼び出すことで、
 *   「画面ローカル状態」と「アプリケーション状態」を分離したまま同期できる
 */
export function useModelCard(params: UseModelCardParams): UseModelCardResult {
  const { sizes, colors, modelNumbers } = params;

  const colorRgbMap: Record<string, string> =
    (params as any).colorRgbMap ?? {};

  const appOnChangeModelNumber:
    | ((sizeLabel: string, color: string, nextCode: string) => void)
    | undefined = (params as any).onChangeModelNumber;

  const [codeMap, setCodeMap] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    const next: Record<string, string> = {};

    sizes.forEach((s) => {
      colors.forEach((c) => {
        const found =
          modelNumbers.find(
            (m) => m.size === s.sizeLabel && m.color === c,
          )?.code ?? "";

        next[makeKey(s.sizeLabel, c)] = found;
      });
    });

    setCodeMap(next);
  }, [sizes, colors, modelNumbers]);

  const getCode = React.useCallback<UseModelCardResult["getCode"]>(
    (sizeLabel, color) => {
      const key = makeKey(sizeLabel, color);
      return codeMap[key] ?? "";
    },
    [codeMap],
  );

  const onChangeModelNumber =
    React.useCallback<UseModelCardResult["onChangeModelNumber"]>(
      (sizeLabel, color, nextCode) => {
        const key = makeKey(sizeLabel, color);

        setCodeMap((prev) => ({
          ...prev,
          [key]: nextCode,
        }));

        if (appOnChangeModelNumber) {
          appOnChangeModelNumber(sizeLabel, color, nextCode);
        }
      },
      [appOnChangeModelNumber],
    );

  const flatModelNumbers: ModelNumber[] = React.useMemo(() => {
    const result: ModelNumber[] = [];

    sizes.forEach((s) => {
      colors.forEach((c) => {
        const key = makeKey(s.sizeLabel, c);
        const code = codeMap[key] ?? "";
        const rgb = colorRgbMap[c];

        result.push({
          size: s.sizeLabel,
          color: c,
          code,
          ...(rgb ? { rgb } : {}),
        } as ModelNumber);
      });
    });

    return result;
  }, [sizes, colors, codeMap, colorRgbMap]);

  return {
    getCode,
    onChangeModelNumber,
    flatModelNumbers,
  };
}

/* =========================================================
 * SizeVariationCard 用 hook ロジック
 * =======================================================*/

/**
 * SizeVariationCard のロジックをこの hook に集約
 */
export function useSizeVariationCard(
  params: UseSizeVariationCardParams,
): UseSizeVariationCardResult {
  const { sizes, mode = "edit", measurementOptions, onChangeSize } = params;

  const isEdit = mode === "edit";

  const readonlyInputProps: UseSizeVariationCardResult["readonlyInputProps"] =
    React.useMemo(
      () =>
        !isEdit
          ? ({ variant: "readonly" as const, readOnly: true } as const)
          : ({} as const),
      [isEdit],
    );

  const measurementHeaders: UseSizeVariationCardResult["measurementHeaders"] =
    React.useMemo(() => {
      if (!measurementOptions || measurementOptions.length === 0) {
        return [];
      }
      return measurementOptions.map((m) => m.label);
    }, [measurementOptions]);

  const handleChange: UseSizeVariationCardResult["handleChange"] =
    React.useCallback(
      (id, key) =>
        (e: React.ChangeEvent<HTMLInputElement>) => {
          if (!isEdit || !onChangeSize) return;

          const v = e.target.value;

          if (key === "sizeLabel") {
            onChangeSize(id, { sizeLabel: v });
          } else {
            onChangeSize(id, {
              [key]: v === "" ? undefined : Number(v),
            } as SizePatch);
          }
        },
      [isEdit, onChangeSize],
    );

  return {
    isEdit,
    readonlyInputProps,
    measurementHeaders,
    handleChange,
  };
}

export type { SizePatch } from "../../application/modelCreateService";

export default useModelCard;