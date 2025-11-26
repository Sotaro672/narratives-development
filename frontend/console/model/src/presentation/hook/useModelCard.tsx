// frontend/console/model/src/presentation/hook/useModelCard.tsx 

import * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type { SizeRow as CatalogSizeRow } from "../../domain/entity/catalog";

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
 *
 * - UI ローカル状態（codeMap）を管理
 * - application 層から渡された onChangeModelNumber も同時に呼び出すことで、
 *   「画面ローカル状態」と「アプリケーション状態」を分離したまま同期できる
 */
export function useModelCard(params: UseModelCardParams): UseModelCardResult {
  const { sizes, colors, modelNumbers } = params;

  // ★ color名 → rgb(hex など) のマップを application 層から受け取れるようにする
  //   - 型定義自体は modelCreateService.tsx 側の UseModelCardParams に
  //     colorRgbMap?: Record<string, string> を追加しておく想定
  const colorRgbMap: Record<string, string> =
    (params as any).colorRgbMap ?? {};

  // ★ application 層（例: useProductBlueprintDetail / useProductBlueprintCreate）
  //   から渡される「本体状態更新用のハンドラ」
  //   - 型定義には含めず any キャストで受けておく（後方互換のため）
  const appOnChangeModelNumber:
    | ((sizeLabel: string, color: string, nextCode: string) => void)
    | undefined = (params as any).onChangeModelNumber;

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
   *
   * - 自分の codeMap を更新
   * - application 層に onChangeModelNumber が渡されていれば、それも呼ぶ
   *   → useProductBlueprintDetail / useProductBlueprintCreate などの
   *     「アプリケーション状態」はこのコールバックで更新される
   */
  const onChangeModelNumber =
    React.useCallback<UseModelCardResult["onChangeModelNumber"]>(
      (sizeLabel, color, nextCode) => {
        const key = makeKey(sizeLabel, color);

        // 1) UI ローカル状態を更新
        setCodeMap((prev) => ({
          ...prev,
          [key]: nextCode,
        }));

        // 2) アプリケーション層の状態も更新（渡されていれば）
        if (appOnChangeModelNumber) {
          appOnChangeModelNumber(sizeLabel, color, nextCode);
        }
      },
      [appOnChangeModelNumber],
    );

  /**
   * API送信などに使える平坦化された ModelNumber の配列
   * - ここで color 名に紐づく rgb も一緒に詰めておく
   *   （ModelNumber 側に rgb?: string を追加しておく前提）
   */
  const flatModelNumbers: ModelNumber[] = React.useMemo(() => {
    const result: ModelNumber[] = [];

    sizes.forEach((s) => {
      colors.forEach((c) => {
        const key = makeKey(s.sizeLabel, c);
        const code = codeMap[key] ?? "";
        const rgb = colorRgbMap[c]; // 例: "#00ff00"

        // rgb フィールドを持つ ModelNumber に対応させる
        result.push({
          size: s.sizeLabel,
          color: c,
          code,
          // ModelNumber 型に rgb?: string を追加してあればそのまま入る
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
  // measurementOptions がなければ空配列
  const measurementHeaders: UseSizeVariationCardResult["measurementHeaders"] =
    React.useMemo(() => {
      if (!measurementOptions || measurementOptions.length === 0) {
        return [];
      }
      // ラベル数の制限なしでそのまま利用
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

/** SizePatch も hook モジュールからそのまま使えるように re-export */
export type { SizePatch } from "../../application/modelCreateService";

export default useModelCard;
