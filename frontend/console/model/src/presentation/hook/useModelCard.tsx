// frontend/console/model/src/presentation/hook/useModelCard.tsx

import * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type {
  SizeRow as CatalogSizeRow,
} from "../../domain/entity/catalog";

// hook 用の型は application 層にまとめる
import type {
  ModelNumber,
  UseModelCardParams,
  UseModelCardResult,
  UseSizeVariationCardParams,
  UseSizeVariationCardResult,
  SizePatch,
} from "../../application/modelCreateService";

/** SizeRow は hook モジュールからも参照できるように alias で再エクスポート */
export type SizeRow = CatalogSizeRow;

/* =========================================================
 * ModelNumber 用 hook ロジック
 * =======================================================*/

/** 内部キー生成（sizeLabel + color 用） */
const makeKey = (sizeLabel: string, color: string) =>
  `${sizeLabel}__${color}`;

/**
 * ModelNumberCard のロジックをすべてこの hook に集約する
 */
export function useModelCard(params: UseModelCardParams): UseModelCardResult {
  const { sizes, colors, modelNumbers } = params;

  /** ローカル状態（サイズ×カラーごとのコード） */
  const [codeMap, setCodeMap] = React.useState<Record<string, string>>({});

  /**
   * props.modelNumbers → codeMap へ同期
   * サイズや色の変更も含めてフル同期
   */
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

  /**
   * 値取得
   */
  const getCode = React.useCallback<UseModelCardResult["getCode"]>(
    (sizeLabel, color) => {
      const key = makeKey(sizeLabel, color);
      return codeMap[key] ?? "";
    },
    [codeMap],
  );

  /**
   * 値更新（Card 側で Input onChange → ここへ伝達）
   */
  const onChangeModelNumber =
    React.useCallback<UseModelCardResult["onChangeModelNumber"]>(
      (sizeLabel, color, nextCode) => {
        const key = makeKey(sizeLabel, color);

        setCodeMap((prev) => ({
          ...prev,
          [key]: nextCode,
        }));
      },
      [],
    );

  /**
   * API送信などに使える平坦化された ModelNumber の配列
   */
  const flatModelNumbers: ModelNumber[] = React.useMemo(() => {
    const result: ModelNumber[] = [];

    sizes.forEach((s) => {
      colors.forEach((c) => {
        const key = makeKey(s.sizeLabel, c);
        result.push({
          size: s.sizeLabel,
          color: c,
          code: codeMap[key] ?? "",
        });
      });
    });

    return result;
  }, [sizes, colors, codeMap]);

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

  // 閲覧モードのみ readOnly スタイルを適用
  const readonlyInputProps: UseSizeVariationCardResult["readonlyInputProps"] =
    React.useMemo(
      () =>
        !isEdit
          ? ({ variant: "readonly" as const, readOnly: true } as const)
          : ({} as const),
      [isEdit],
    );

  // カタログの measurement に応じてヘッダ名を切り替える
  //   - 現在の実装は 4 列分の数値カラムを持っているため、最大 4 つまで使用
  const measurementHeaders: UseSizeVariationCardResult["measurementHeaders"] =
    React.useMemo(() => {
      if (!measurementOptions || measurementOptions.length === 0) {
        // フォールバック（従来のヘッダー）
        return ["胸囲", "ウエスト", "着丈", "肩幅"];
      }
      return measurementOptions.map((m) => m.label).slice(0, 4);
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

/** SizePatch も hook モジュールからそのまま使えるように re-export */
export type { SizePatch } from "../../application/modelCreateService";

export default useModelCard;
