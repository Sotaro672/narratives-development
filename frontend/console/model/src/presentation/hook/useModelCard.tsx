// frontend/console/model/src/presentation/hook/useModelCard.tsx

import * as React from "react";

// ★ 商品設計側のカタログから採寸ラベルを受け取る
import type { MeasurementOption } from "../../../../productBlueprint/src/domain/entity/catalog";

/* =========================================================
 * ModelNumber（既存）用の型・ロジック
 * =======================================================*/

export type ModelNumber = {
  size: string;  // "S" | "M" | ...
  color: string; // "ホワイト" | ...
  code: string;  // "LM-SB-S-WHT"
};

type SizeLike = { id: string; sizeLabel: string };

export type UseModelCardParams = {
  sizes: SizeLike[];
  colors: string[];
  /**
   * 初期のモデルナンバー一覧
   * ModelNumberCard の props.modelNumbers と同じ
   */
  modelNumbers: ModelNumber[];
};

export type UseModelCardResult = {
  /**
   * サイズ×カラーのコードを返す
   * ModelNumberCard に渡す
   */
  getCode: (sizeLabel: string, color: string) => string;

  /**
   * ModelNumberCard の onChangeModelNumber に渡す
   */
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  /**
   * 最終的に modelNumbers として保存／API送信に使える配列
   */
  flatModelNumbers: ModelNumber[];
};

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
  const getCode = React.useCallback(
    (sizeLabel: string, color: string) => {
      const key = makeKey(sizeLabel, color);
      return codeMap[key] ?? "";
    },
    [codeMap],
  );

  /**
   * 値更新（Card 側で Input onChange → ここへ伝達）
   */
  const onChangeModelNumber = React.useCallback(
    (sizeLabel: string, color: string, nextCode: string) => {
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
 * SizeVariationCard 用の型・ロジック（新規追加）
 * =======================================================*/

/**
 * サイズ行（UI/ドメイン共有）
 * ※ 既存の SizeVariationCard からロジックを移譲
 */
export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest?: number;
  waist?: number;
  length?: number;
  shoulder?: number;
};

/** onChangeSize の patch 用型 */
export type SizePatch = Partial<Omit<SizeRow, "id">>;

export type UseSizeVariationCardParams = {
  sizes: SizeRow[];
  mode?: "edit" | "view";
  /** 商品設計側から渡される採寸定義（itemType に連動） */
  measurementOptions?: MeasurementOption[];
  /** 1セル変更時の通知（元の SizeVariationCard の onChangeSize と同じ） */
  onChangeSize?: (id: string, patch: SizePatch) => void;
};

export type UseSizeVariationCardResult = {
  /** 編集可否（mode === "edit"） */
  isEdit: boolean;
  /** Input に渡す readOnly 系 props（閲覧モード時のみ付与） */
  readonlyInputProps: { variant?: "readonly"; readOnly?: boolean };
  /** ヘッダに表示する採寸ラベル配列（最大4つ） */
  measurementHeaders: string[];
  /**
   * Cell 用 onChange ハンドラ生成関数
   * SizeVariationCard 内の Input の onChange にそのまま渡す想定
   */
  handleChange: (
    id: string,
    key: keyof Omit<SizeRow, "id">,
  ) => (e: React.ChangeEvent<HTMLInputElement>) => void;
};

/**
 * SizeVariationCard のロジックをこの hook に集約
 */
export function useSizeVariationCard(
  params: UseSizeVariationCardParams,
): UseSizeVariationCardResult {
  const { sizes, mode = "edit", measurementOptions, onChangeSize } = params;

  const isEdit = mode === "edit";

  // 閲覧モードのみ readOnly スタイルを適用
  const readonlyInputProps = React.useMemo(
    () =>
      !isEdit
        ? ({ variant: "readonly" as const, readOnly: true } as const)
        : ({} as const),
    [isEdit],
  );

  // ★ カタログの measurement に応じてヘッダ名を切り替える
  //   - 現在の実装は 4 列分の数値カラムを持っているため、最大 4 つまで使用
  const measurementHeaders = React.useMemo(() => {
    if (!measurementOptions || measurementOptions.length === 0) {
      // フォールバック（従来のヘッダー）
      return ["胸囲", "ウエスト", "着丈", "肩幅"];
    }
    return measurementOptions.map((m) => m.label).slice(0, 4);
  }, [measurementOptions]);

  const handleChange = React.useCallback(
    (id: string, key: keyof Omit<SizeRow, "id">) =>
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

export default useModelCard;
