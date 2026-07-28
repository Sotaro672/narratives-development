// frontend/console/shell/src/features/productBlueprint/presentation/hooks/detail/useVariationsEditor.ts

import * as React from "react";

import type {
  ApparelModelNumberRow as ModelNumberRow,
  ApparelSizeInput,
} from "../../../../../shared/types/apparel";

import type {
  AlcoholModelNumber,
  VolumeRow,
} from "../../../../model/application/modelCreateService";

import { useModelCard } from "../../../../model/presentation/hook/useModelCard";

type SizeRow = ApparelSizeInput & {
  id: string;
};

/**
 * UI state derived from ModelVariation list
 * (already mapped by variationMapper, etc.)
 */
export type VariationsUiState = {
  colors: string[];
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  /** color名 → rgb hex（#rrggbb） */
  colorRgbMap: Record<string, string>;

  /**
   * alcohol model variation用。
   *
   * volumeはProductBlueprint.categoryFieldsではなく
   * model domain側で扱う。
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
  getCode: (
    sizeLabel: string,
    color: string,
  ) => string;

  // initialize/replace
  // （例：variations取得後）
  setFromUiState: (
    next: VariationsUiState,
  ) => void;

  // color
  onChangeColorInput: (
    value: string,
  ) => void;

  onAddColor: () => void;

  onRemoveColor: (
    name: string,
  ) => void;

  onChangeColorRgb: (
    name: string,
    hex: string,
  ) => void;

  // size
  onRemoveSize: (
    id: string,
  ) => void;

  onAddSize: () => void;

  onChangeSize: (
    id: string,
    patch: Partial<
      Omit<SizeRow, "id">
    >,
  ) => void;

  // apparel model number
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  // alcohol volume
  onAddVolume: () => void;

  onRemoveVolume: (
    id: string,
  ) => void;

  onChangeVolume: (
    id: string,
    patch: Partial<
      Omit<VolumeRow, "id">
    >,
  ) => void;

  // alcohol model number
  onChangeAlcoholModelNumber: (
    volumeLabel: string,
    nextCode: string,
  ) => void;
};

function createId(
  prefix: string,
): string {
  return (
    typeof crypto !== "undefined" &&
    "randomUUID" in crypto
  )
    ? crypto.randomUUID()
    : `${prefix}-${Date.now()}-${Math.random()
        .toString(36)
        .slice(2, 8)}`;
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

function normalizeVolumeValue(
  value: unknown,
): number {
  if (
    typeof value !== "number" ||
    !Number.isFinite(value)
  ) {
    return 0;
  }

  return value < 0
    ? 0
    : value;
}

function normalizeVolumeUnit(
  value: unknown,
): string {
  const unit =
    String(
      value ?? "",
    ).trim();

  return unit || "ml";
}

function toVolumeLabel(
  row: Pick<
    VolumeRow,
    "volumeValue" | "volumeUnit"
  >,
): string {
  const value =
    normalizeVolumeValue(
      row.volumeValue,
    );

  const unit =
    normalizeVolumeUnit(
      row.volumeUnit,
    );

  if (value <= 0) {
    return "";
  }

  return `${value}${unit}`;
}

function toAlcoholModelNumberVolumeLabel(
  modelNumber: AlcoholModelNumber,
): string {
  return toVolumeLabel({
    volumeValue:
      modelNumber.volume.value,

    volumeUnit:
      modelNumber.volume.unit,
  });
}

/**
 * Presentation-level editor state for variations.
 *
 * colors / sizes / modelNumbers / colorRgbMap /
 * volumes / alcoholModelNumbersを管理する。
 *
 * - editor logicをuseProductBlueprintDetail.tsxから分離する
 * - variationMapperのmapVariationsToUiStateが生成したstateを受け取る
 */
export function useVariationsEditor(
  initial?: Partial<VariationsUiState>,
): UseVariationsEditorResult {
  const [
    colorInput,
    setColorInput,
  ] =
    React.useState<string>(
      "",
    );

  const [
    colors,
    setColors,
  ] =
    React.useState<string[]>(
      initial?.colors ?? [],
    );

  const [
    sizes,
    setSizes,
  ] =
    React.useState<SizeRow[]>(
      initial?.sizes ?? [],
    );

  const [
    modelNumbers,
    setModelNumbers,
  ] =
    React.useState<
      ModelNumberRow[]
    >(
      initial?.modelNumbers ??
        [],
    );

  const [
    colorRgbMap,
    setColorRgbMap,
  ] =
    React.useState<
      Record<string, string>
    >(
      initial?.colorRgbMap ??
        {},
    );

  const [
    volumes,
    setVolumes,
  ] =
    React.useState<VolumeRow[]>(
      initial?.volumes ?? [],
    );

  const [
    alcoholModelNumbers,
    setAlcoholModelNumbers,
  ] =
    React.useState<
      AlcoholModelNumber[]
    >(
      initial
        ?.alcoholModelNumbers ??
        [],
    );

  const setFromUiState =
    React.useCallback(
      (
        next:
          VariationsUiState,
      ) => {
        setColors(
          Array.isArray(
            next.colors,
          )
            ? next.colors
            : [],
        );

        setSizes(
          Array.isArray(
            next.sizes,
          )
            ? next.sizes
            : [],
        );

        setModelNumbers(
          Array.isArray(
            next.modelNumbers,
          )
            ? next.modelNumbers
            : [],
        );

        setColorRgbMap(
          next.colorRgbMap ??
            {},
        );

        setVolumes(
          Array.isArray(
            next.volumes,
          )
            ? next.volumes
            : [],
        );

        setAlcoholModelNumbers(
          Array.isArray(
            next.alcoholModelNumbers,
          )
            ? next.alcoholModelNumbers
            : [],
        );

        setColorInput("");
      },
      [],
    );

  // ---------------------------------
  // ModelNumberCard用ロジックは
  // useModelCardに委譲
  // ---------------------------------

  const {
    getCode,
    onChangeModelNumber:
      uiOnChangeModelNumber,
  } =
    useModelCard({
      sizes:
        Array.isArray(sizes)
          ? sizes
          : [],

      colors:
        Array.isArray(colors)
          ? colors
          : [],

      modelNumbers:
        Array.isArray(
          modelNumbers,
        )
          ? (
              modelNumbers as any
            )
          : [],

      colorRgbMap:
        colorRgbMap ?? {},
    });

  // ---------------------------------
  // Internal:
  // apparel modelNumbers state update
  // ---------------------------------

  const patchModelNumberState =
    React.useCallback(
      (
        sizeLabel: string,
        color: string,
        nextCode: string,
      ) => {
        setModelNumbers(
          (previous) => {
            const index =
              previous.findIndex(
                (modelNumber) =>
                  modelNumber.size ===
                    sizeLabel &&
                  modelNumber.color ===
                    color,
              );

            const trimmed =
              nextCode.trim();

            // 空の場合は削除する
            if (!trimmed) {
              if (index === -1) {
                return previous;
              }

              const copy = [
                ...previous,
              ];

              copy.splice(
                index,
                1,
              );

              return copy;
            }

            const next:
              ModelNumberRow = {
              size: sizeLabel,
              color,
              code: trimmed,
            };

            if (index === -1) {
              return [
                ...previous,
                next,
              ];
            }

            const copy = [
              ...previous,
            ];

            copy[index] =
              next;

            return copy;
          },
        );
      },
      [],
    );

  const onChangeModelNumber =
    React.useCallback(
      (
        sizeLabel: string,
        color: string,
        nextCode: string,
      ) => {
        uiOnChangeModelNumber(
          sizeLabel,
          color,
          nextCode,
        );

        patchModelNumberState(
          sizeLabel,
          color,
          nextCode,
        );
      },
      [
        uiOnChangeModelNumber,
        patchModelNumberState,
      ],
    );

  // ---------------------------------
  // Color handlers
  // ---------------------------------

  const onAddColor =
    React.useCallback(() => {
      const value =
        colorInput.trim();

      if (
        !value ||
        colors.includes(value)
      ) {
        return;
      }

      setColors(
        (previous) => [
          ...previous,
          value,
        ],
      );

      setColorInput("");
    }, [
      colorInput,
      colors,
    ]);

  const onRemoveColor =
    React.useCallback(
      (
        name: string,
      ) => {
        const key =
          name.trim();

        if (!key) {
          return;
        }

        setColors(
          (previous) =>
            previous.filter(
              (color) =>
                color !== key,
            ),
        );

        setColorRgbMap(
          (previous) => {
            const next = {
              ...previous,
            };

            delete next[key];

            return next;
          },
        );

        setModelNumbers(
          (
            previousModelNumbers,
          ) =>
            previousModelNumbers.filter(
              (modelNumber) =>
                modelNumber.color !==
                key,
            ),
        );
      },
      [],
    );

  const onChangeColorRgb =
    React.useCallback(
      (
        name: string,
        hex: string,
      ) => {
        const colorName =
          name.trim();

        let value =
          String(
            hex ?? "",
          ).trim();

        if (
          !colorName ||
          !value
        ) {
          return;
        }

        if (
          !value.startsWith("#")
        ) {
          value = `#${value}`;
        }

        setColorRgbMap(
          (previous) => ({
            ...previous,
            [colorName]: value,
          }),
        );
      },
      [],
    );

  // ---------------------------------
  // Size handlers
  // ---------------------------------

  const onRemoveSize =
    React.useCallback(
      (
        id: string,
      ) => {
        setSizes(
          (previous) => {
            const target =
              previous.find(
                (size) =>
                  size.id === id,
              );

            const next =
              previous.filter(
                (size) =>
                  size.id !== id,
              );

            if (target) {
              const sizeLabel =
                (
                  target.sizeLabel ??
                  ""
                ).trim();

              if (sizeLabel) {
                setModelNumbers(
                  (
                    previousModelNumbers,
                  ) =>
                    previousModelNumbers.filter(
                      (
                        modelNumber,
                      ) =>
                        modelNumber.size !==
                        sizeLabel,
                    ),
                );
              }
            }

            return next;
          },
        );
      },
      [],
    );

  const onAddSize =
    React.useCallback(() => {
      setSizes(
        (previous) => [
          ...previous,
          newSizeRow(),
        ],
      );
    }, []);

  const onChangeSize =
    React.useCallback(
      (
        id: string,
        patch: Partial<
          Omit<SizeRow, "id">
        >,
      ) => {
        const safePatch:
          Partial<
            Omit<SizeRow, "id">
          > = {
          ...patch,
        };

        const clampField = (
          key: keyof Omit<
            SizeRow,
            "id"
          >,
        ) => {
          const value =
            safePatch[key];

          if (
            typeof value ===
            "number"
          ) {
            safePatch[key] = (
              value < 0
                ? 0
                : value
            ) as never;
          }
        };

        // トップス
        clampField("length");
        clampField("width");
        clampField("chest");
        clampField("shoulder");
        clampField(
          "sleeveLength",
        );

        // ボトムス
        clampField("waist");
        clampField("hip");
        clampField("rise");
        clampField("inseam");
        clampField("thigh");
        clampField("hemWidth");

        const previousRow =
          sizes.find(
            (size) =>
              size.id === id,
          );

        const previousLabel =
          String(
            previousRow
              ?.sizeLabel ??
              "",
          ).trim();

        const nextLabelRaw =
          safePatch.sizeLabel;

        const nextLabel =
          typeof nextLabelRaw ===
          "string"
            ? nextLabelRaw.trim()
            : nextLabelRaw == null
              ? null
              : String(
                  nextLabelRaw,
                ).trim();

        if (
          nextLabel !== null &&
          nextLabel !==
            previousLabel
        ) {
          if (!nextLabel) {
            if (previousLabel) {
              setModelNumbers(
                (previous) =>
                  previous.filter(
                    (
                      modelNumber,
                    ) =>
                      modelNumber.size !==
                      previousLabel,
                  ),
              );
            }
          } else if (
            previousLabel
          ) {
            setModelNumbers(
              (previous) =>
                previous.map(
                  (
                    modelNumber,
                  ) =>
                    modelNumber.size ===
                    previousLabel
                      ? {
                          ...modelNumber,
                          size:
                            nextLabel,
                        }
                      : modelNumber,
                ),
            );
          }
        }

        setSizes(
          (previous) =>
            previous.map(
              (size) =>
                size.id === id
                  ? {
                      ...size,
                      ...safePatch,
                    }
                  : size,
            ),
        );
      },
      [
        sizes,
      ],
    );

  // ---------------------------------
  // Alcohol volume handlers
  // ---------------------------------

  const onAddVolume =
    React.useCallback(() => {
      setVolumes(
        (previous) => [
          ...previous,
          newVolumeRow(),
        ],
      );
    }, []);

  const onRemoveVolume =
    React.useCallback(
      (
        id: string,
      ) => {
        const target =
          volumes.find(
            (volume) =>
              volume.id === id,
          );

        const targetLabel =
          target
            ? toVolumeLabel(
                target,
              )
            : "";

        setVolumes(
          (previous) =>
            previous.filter(
              (volume) =>
                volume.id !== id,
            ),
        );

        if (targetLabel) {
          setAlcoholModelNumbers(
            (previous) =>
              previous.filter(
                (
                  modelNumber,
                ) =>
                  toAlcoholModelNumberVolumeLabel(
                    modelNumber,
                  ) !==
                  targetLabel,
              ),
          );
        }
      },
      [
        volumes,
      ],
    );

  const onChangeVolume =
    React.useCallback(
      (
        id: string,
        patch: Partial<
          Omit<VolumeRow, "id">
        >,
      ) => {
        const safePatch:
          Partial<
            Omit<
              VolumeRow,
              "id"
            >
          > = {
          ...patch,
        };

        if (
          safePatch.volumeValue !==
          undefined
        ) {
          safePatch.volumeValue =
            normalizeVolumeValue(
              safePatch.volumeValue,
            );
        }

        if (
          safePatch.volumeUnit !==
          undefined
        ) {
          safePatch.volumeUnit =
            normalizeVolumeUnit(
              safePatch.volumeUnit,
            );
        }

        const previousRow =
          volumes.find(
            (volume) =>
              volume.id === id,
          );

        const previousLabel =
          previousRow
            ? toVolumeLabel(
                previousRow,
              )
            : "";

        const nextRow:
          VolumeRow | null =
          previousRow
            ? {
                ...previousRow,
                ...safePatch,
              }
            : null;

        const nextLabel =
          nextRow
            ? toVolumeLabel(
                nextRow,
              )
            : "";

        if (
          previousLabel &&
          nextRow &&
          nextLabel &&
          previousLabel !==
            nextLabel
        ) {
          setAlcoholModelNumbers(
            (previous) =>
              previous.map(
                (
                  modelNumber,
                ) =>
                  toAlcoholModelNumberVolumeLabel(
                    modelNumber,
                  ) ===
                  previousLabel
                    ? {
                        ...modelNumber,

                        volume: {
                          value:
                            nextRow
                              .volumeValue,

                          unit:
                            nextRow
                              .volumeUnit,
                        },
                      }
                    : modelNumber,
              ),
          );
        }

        if (
          previousLabel &&
          !nextLabel
        ) {
          setAlcoholModelNumbers(
            (previous) =>
              previous.filter(
                (
                  modelNumber,
                ) =>
                  toAlcoholModelNumberVolumeLabel(
                    modelNumber,
                  ) !==
                  previousLabel,
              ),
          );
        }

        setVolumes(
          (previous) =>
            previous.map(
              (volume) =>
                volume.id === id
                  ? {
                      ...volume,
                      ...safePatch,
                    }
                  : volume,
            ),
        );
      },
      [
        volumes,
      ],
    );

  const onChangeAlcoholModelNumber =
    React.useCallback(
      (
        volumeLabel: string,
        nextCode: string,
      ) => {
        const label =
          volumeLabel.trim();

        if (!label) {
          return;
        }

        const volumeRow =
          volumes.find(
            (volume) =>
              toVolumeLabel(
                volume,
              ) === label,
          );

        if (!volumeRow) {
          return;
        }

        setAlcoholModelNumbers(
          (previous) => {
            const index =
              previous.findIndex(
                (
                  modelNumber,
                ) =>
                  toAlcoholModelNumberVolumeLabel(
                    modelNumber,
                  ) === label,
              );

            const trimmed =
              nextCode.trim();

            if (!trimmed) {
              if (index === -1) {
                return previous;
              }

              const copy = [
                ...previous,
              ];

              copy.splice(
                index,
                1,
              );

              return copy;
            }

            const next:
              AlcoholModelNumber = {
              kind: "alcohol",

              volume: {
                value:
                  volumeRow
                    .volumeValue,

                unit:
                  volumeRow
                    .volumeUnit,
              },

              code: trimmed,
            };

            if (index === -1) {
              return [
                ...previous,
                next,
              ];
            }

            const copy = [
              ...previous,
            ];

            copy[index] =
              next;

            return copy;
          },
        );
      },
      [
        volumes,
      ],
    );

  // ---------------------------------
  // Cleanup invalid model numbers
  // ---------------------------------

  React.useEffect(() => {
    const validColors =
      new Set(
        colors
          .map(
            (color) =>
              color.trim(),
          )
          .filter(Boolean),
      );

    const validSizes =
      new Set(
        sizes
          .map(
            (size) =>
              size.sizeLabel,
          )
          .map(
            (value) =>
              typeof value ===
              "string"
                ? value.trim()
                : String(
                    value ?? "",
                  ).trim(),
          )
          .filter(Boolean),
      );

    setModelNumbers(
      (previous) =>
        previous.filter(
          (modelNumber) =>
            validColors.has(
              modelNumber.color,
            ) &&
            validSizes.has(
              modelNumber.size,
            ),
        ),
    );
  }, [
    colors,
    sizes,
  ]);

  React.useEffect(() => {
    const validVolumeLabels =
      new Set(
        volumes
          .map(
            toVolumeLabel,
          )
          .filter(Boolean),
      );

    setAlcoholModelNumbers(
      (previous) =>
        previous.filter(
          (modelNumber) =>
            validVolumeLabels.has(
              toAlcoholModelNumberVolumeLabel(
                modelNumber,
              ),
            ),
        ),
    );
  }, [
    volumes,
  ]);

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

    onChangeColorInput:
      setColorInput,

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