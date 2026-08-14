// frontend/console/shell/src/features/model/presentation/hook/useModelCard.tsx

import * as React from "react";

import type {
  ApparelModelNumber,
  SizeRow,
  UseModelCardParams,
  UseModelCardResult,
  UseSizeVariationCardParams,
  UseSizeVariationCardResult,
  SizePatch,
} from "../../application/modelCreateService";

export type { SizeRow };

/* =========================================================
 * ModelNumber 用 hook ロジック
 * =======================================================*/

const makeKey = (sizeLabel: string, color: string) =>
  `${sizeLabel}__${color}`;

export function useModelCard(
  params: UseModelCardParams,
): UseModelCardResult {
  const {
    sizes,
    colors,
    modelNumbers,
    colorRgbMap = {},
    onChangeModelNumber: appOnChangeModelNumber,
  } = params;

  const [codeMap, setCodeMap] = React.useState<Record<string, string>>({});

  React.useEffect(() => {
    const next: Record<string, string> = {};

    sizes.forEach((size) => {
      colors.forEach((color) => {
        const found =
          modelNumbers.find(
            (modelNumber) =>
              modelNumber.size === size.sizeLabel &&
              modelNumber.color === color,
          )?.code ?? "";

        next[makeKey(size.sizeLabel, color)] = found;
      });
    });

    setCodeMap(next);
  }, [sizes, colors, modelNumbers]);

  const getCode = React.useCallback<UseModelCardResult["getCode"]>(
    (sizeLabel, color) =>
      codeMap[makeKey(sizeLabel, color)] ?? "",
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

        appOnChangeModelNumber?.(
          sizeLabel,
          color,
          nextCode,
        );
      },
      [appOnChangeModelNumber],
    );

  const flatModelNumbers = React.useMemo<ApparelModelNumber[]>(() => {
    const result: ApparelModelNumber[] = [];

    sizes.forEach((size) => {
      colors.forEach((color) => {
        const code =
          codeMap[makeKey(size.sizeLabel, color)] ?? "";
        const rgb = colorRgbMap[color];

        result.push({
          size: size.sizeLabel,
          color,
          code,
          ...(rgb ? { rgb } : {}),
        });
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

export function useSizeVariationCard(
  params: UseSizeVariationCardParams,
): UseSizeVariationCardResult {
  const {
    sizes,
    mode = "edit",
    measurementOptions,
    onChangeSize,
  } = params;

  const isEdit = mode === "edit";

  const readonlyInputProps =
    React.useMemo<UseSizeVariationCardResult["readonlyInputProps"]>(
      () =>
        isEdit
          ? {}
          : {
              variant: "readonly",
              readOnly: true,
            },
      [isEdit],
    );

  const measurementHeaders =
    React.useMemo<UseSizeVariationCardResult["measurementHeaders"]>(
      () =>
        measurementOptions?.map(
          (measurement) => measurement.label,
        ) ?? [],
      [measurementOptions],
    );

  const handleChange =
    React.useCallback<UseSizeVariationCardResult["handleChange"]>(
      (id, key) =>
        (event: React.ChangeEvent<HTMLInputElement>) => {
          if (!isEdit || !onChangeSize) return;

          const value = event.target.value;

          if (key === "sizeLabel") {
            onChangeSize(id, {
              sizeLabel: value,
            });
            return;
          }

          onChangeSize(id, {
            [key]:
              value === ""
                ? undefined
                : Number(value),
          } as SizePatch);
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

export type {
  SizePatch,
} from "../../application/modelCreateService";

export default useModelCard;