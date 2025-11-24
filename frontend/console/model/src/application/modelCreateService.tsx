// frontend/console/model/src/application/modelCreateService.tsx

// React のイベント型だけ型として利用
import type * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type {
  MeasurementOption,
  SizeRow,
} from "../domain/entity/catalog";

/**
 * モデル作成（CreateModelVariation など）のための
 * アプリケーション層の型定義とユーティリティをまとめるファイル。
 *
 * - Presentation (hook) 層から参照される型をここに集約
 * - 将来的に HTTP 呼び出しロジックもここに追加していく想定
 */

/* =========================================================
 * ModelNumber 関連
 * =======================================================*/

export type ModelNumber = {
  size: string;  // "S" | "M" | ...
  color: string; // "ホワイト" | ...
  code: string;  // "LM-SB-S-WHT"
};

export type SizeLike = { id: string; sizeLabel: string };

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

/* =========================================================
 * SizeVariationCard 関連
 * =======================================================*/

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

/* =========================================================
 * （今後追加用）CreateModelVariation 用の HTTP サービスなど
 * =======================================================*/

// 例: 将来的にここに CreateModelVariation の HTTP 呼び出しを実装する
// export async function createModelVariation(...) { ... }
