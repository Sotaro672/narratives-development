// frontend/console/shell/src/features/productBlueprint/presentation/cards/categoryFields/WashTagField.tsx

import * as React from "react";
import {
  ShieldCheck,
  X,
} from "lucide-react";

import {
  Badge,
} from "../../../../../shared/ui/badge";

import {
  Button,
} from "../../../../../shared/ui/button";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shared/ui/popover";

import {
  Checkbox,
} from "../../../../../shared/ui/checkbox";

// ============================
// Wash tag definitions
// ============================

type WashTagCategory =
  | "洗濯"
  | "漂白"
  | "乾燥"
  | "アイロン"
  | "ドライクリーニング"
  | "ウェットクリーニング";

type WashTagOption = {
  value: string;
  label: string;
  category: WashTagCategory;
};

const WASH_TAG_OPTIONS:
  WashTagOption[] = [
  // 洗濯
  {
    category: "洗濯",
    value: "手洗い",
    label: "手洗い",
  },
  {
    category: "洗濯",
    value: "洗濯機可",
    label: "洗濯機可",
  },
  {
    category: "洗濯",
    value: "弱い洗濯",
    label: "弱い洗濯",
  },
  {
    category: "洗濯",
    value: "液温30℃限度",
    label: "液温30℃限度",
  },
  {
    category: "洗濯",
    value: "液温40℃限度",
    label: "液温40℃限度",
  },
  {
    category: "洗濯",
    value: "水洗い不可",
    label: "水洗い不可",
  },

  // 漂白
  {
    category: "漂白",
    value: "酸素系漂白可",
    label: "酸素系漂白可",
  },
  {
    category: "漂白",
    value: "塩素系漂白可",
    label: "塩素系漂白可",
  },
  {
    category: "漂白",
    value: "漂白不可",
    label: "漂白不可",
  },

  // 乾燥
  {
    category: "乾燥",
    value: "タンブル乾燥可 低温",
    label: "タンブル乾燥可（低温）",
  },
  {
    category: "乾燥",
    value: "タンブル乾燥可 中温",
    label: "タンブル乾燥可（中温）",
  },
  {
    category: "乾燥",
    value: "タンブル乾燥不可",
    label: "タンブル乾燥不可",
  },
  {
    category: "乾燥",
    value: "つり干し",
    label: "つり干し",
  },
  {
    category: "乾燥",
    value: "日陰つり干し",
    label: "日陰つり干し",
  },
  {
    category: "乾燥",
    value: "平干し",
    label: "平干し",
  },
  {
    category: "乾燥",
    value: "日陰平干し",
    label: "日陰平干し",
  },

  // アイロン
  {
    category: "アイロン",
    value: "アイロン低温",
    label: "アイロン低温（110℃まで）",
  },
  {
    category: "アイロン",
    value: "アイロン中温",
    label: "アイロン中温（150℃まで）",
  },
  {
    category: "アイロン",
    value: "アイロン高温",
    label: "アイロン高温（200℃まで）",
  },
  {
    category: "アイロン",
    value: "アイロン不可",
    label: "アイロン不可",
  },

  // ドライクリーニング
  {
    category:
      "ドライクリーニング",
    value:
      "ドライクリーニング可",
    label:
      "ドライクリーニング可",
  },
  {
    category:
      "ドライクリーニング",
    value:
      "石油系ドライ可",
    label:
      "石油系ドライクリーニング可",
  },
  {
    category:
      "ドライクリーニング",
    value:
      "ドライクリーニング不可",
    label:
      "ドライクリーニング不可",
  },

  // ウェットクリーニング
  {
    category:
      "ウェットクリーニング",
    value:
      "ウェットクリーニング可",
    label:
      "ウェットクリーニング可",
  },
  {
    category:
      "ウェットクリーニング",
    value:
      "ウェットクリーニング弱",
    label:
      "ウェットクリーニング（弱）",
  },
  {
    category:
      "ウェットクリーニング",
    value:
      "ウェットクリーニング非常に弱",
    label:
      "ウェットクリーニング（非常に弱）",
  },
  {
    category:
      "ウェットクリーニング",
    value:
      "ウェットクリーニング不可",
    label:
      "ウェットクリーニング不可",
  },
];

// ============================
// Component
// ============================

type WashTagFieldProps = {
  value: string[];
  mode?: "edit" | "view";

  onChange?: (
    nextTags: string[],
  ) => void;
};

const WashTagField:
  React.FC<
    WashTagFieldProps
  > = ({
  value,
  mode = "edit",
  onChange,
}) => {
  const isEdit =
    mode === "edit";

  const safeValue =
    Array.isArray(value)
      ? value
      : [];

  const washTagGroups =
    React.useMemo(() => {
      const map =
        new Map<
          WashTagCategory,
          WashTagOption[]
        >();

      for (
        const option
        of WASH_TAG_OPTIONS
      ) {
        const category =
          option.category;

        const list =
          map.get(category) ??
          [];

        list.push(option);

        map.set(
          category,
          list,
        );
      }

      return Array.from(
        map.entries(),
      );
    }, []);

  const handleToggle =
    React.useCallback(
      (
        tagValue: string,
      ) => {
        if (!onChange) {
          return;
        }

        if (
          safeValue.includes(
            tagValue,
          )
        ) {
          onChange(
            safeValue.filter(
              (tag) =>
                tag !==
                tagValue,
            ),
          );

          return;
        }

        onChange([
          ...safeValue,
          tagValue,
        ]);
      },
      [
        onChange,
        safeValue,
      ],
    );

  return (
    <>
      <div className="label">
        品質保証（洗濯方法タグ）
      </div>

      <div className="chips flex flex-wrap gap-2">
        {safeValue.map(
          (tag) => (
            <Badge
              key={tag}
              className="chip inline-flex items-center gap-1.5 px-2 py-1"
            >
              <ShieldCheck
                size={14}
              />

              {tag}

              {isEdit &&
                onChange && (
                  <button
                    type="button"
                    onClick={() =>
                      onChange(
                        safeValue.filter(
                          (item) =>
                            item !==
                            tag,
                        ),
                      )
                    }
                    className="chip-remove"
                    aria-label={`${tag} を削除`}
                  >
                    <X
                      size={12}
                    />
                  </button>
                )}
            </Badge>
          ),
        )}
      </div>

      {isEdit &&
        onChange && (
          <div className="mt-2 flex flex-wrap gap-2">
            {washTagGroups.map(
              ([
                category,
                options,
              ]) => (
                <Popover
                  key={category}
                >
                  <PopoverTrigger>
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      className="btn"
                      aria-label={`${category} のタグを追加`}
                    >
                      {category}
                    </Button>
                  </PopoverTrigger>

                  <PopoverContent
                    align="start"
                    className="w-64 space-y-1 p-2"
                  >
                    {options.map(
                      (
                        option,
                      ) => {
                        const checked =
                          safeValue.includes(
                            option.value,
                          );

                        const checkboxId =
                          `wash-tag-${option.value}`;

                        return (
                          <label
                            key={
                              option.value
                            }
                            htmlFor={
                              checkboxId
                            }
                            className="flex cursor-pointer items-center gap-2 py-0.5 text-sm"
                          >
                            <Checkbox
                              id={
                                checkboxId
                              }
                              checked={
                                checked
                              }
                              onCheckedChange={() =>
                                handleToggle(
                                  option.value,
                                )
                              }
                            />

                            <span>
                              {
                                option.label
                              }
                            </span>
                          </label>
                        );
                      },
                    )}
                  </PopoverContent>
                </Popover>
              ),
            )}
          </div>
        )}
    </>
  );
};

export default WashTagField;