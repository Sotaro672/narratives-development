// frontend/console/model/src/application/modelCreateService.tsx

// React のイベント型だけ型として利用
import type * as React from "react";

// 採寸系の型は model ドメインの catalog から
import type {
  MeasurementOption,
  SizeRow,
  MeasurementKey,
} from "../domain/entity/catalog";

// ★ HTTP リポジトリ（CreateModelVariation 用）
import {
  createModelVariations,
  type CreateModelVariationRequest,
} from "../infrastructure/repository/modelRepositoryHTTP";

/**
 * モデル作成（CreateModelVariation など）のための
 * アプリケーション層の型定義とユーティリティをまとめるファイル。
 *
 * - Presentation (hook) 層から参照される型をここに集約
 * - HTTP 呼び出しは infrastructure/repository/modelRepositoryHTTP.ts に移譲
 */

/* =========================================================
 * ModelNumber 関連
 * =======================================================*/

export type ModelNumber = {
  size: string; // "S" | "M" | ...
  color: string; // "ホワイト" | ...
  code: string; // "LM-SB-S-WHT"
  /** ColorVariationCard → useModelCard 経由で渡される RGB(hex or int) */
  rgb?: string | number;
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
  /** 色名 → RGB(hex) の対応マップ */
  colorRgbMap?: Record<string, string>;
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
   * - rgb フィールドも保持
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
 * ProductBlueprint Create 後に受け取る JSON 用の型
 * =======================================================*/

/**
 * measurements 部分の型
 *
 * - catalog.ts 側の MeasurementKey（「着丈」「身幅」…）をキーとしたマップに変更
 * - これにより、トップス／ボトムス両方の全採寸項目を表現できる
 */
export type NewModelVariationMeasurements = Partial<
  Record<MeasurementKey, number | null>
>;

/**
 * 1 モデルバリエーション分の payload 型
 * - productBlueprintCreateService.ts で buildMeasurements 済みの JSON を構築し、
 *   その 1 要素分と対応する。
 *
 * rgb は ColorVariationCard 側から渡される想定の任意フィールド
 */
export type NewModelVariationPayload = {
  sizeLabel: string;
  color: string; // 色名（例: "グリーン"）
  rgb?: number | string; // RGB(hex or int)
  modelNumber: string;
  createdBy: string;
  measurements: NewModelVariationMeasurements;
};

/**
 * productBlueprintCreateService.ts から渡される JSON 全体
 * - 作成済み productBlueprint の ID と、その ID に紐づく variations 一覧。
 */
export type ModelVariationsFromProductBlueprint = {
  /** backend の productBlueprint / model が共有する productBlueprintId */
  productBlueprintId: string;
  /** color × size × modelNumber × measurements の一覧 */
  variations: NewModelVariationPayload[];
};

/**
 * productBlueprintCreateService.ts からの JSON を受け取り、
 * CreateModelVariation API を叩くためのエントリポイント。
 */
export async function createModelVariationsFromProductBlueprint(
  payload: ModelVariationsFromProductBlueprint,
): Promise<void> {
  // 受け取った payload のログ（採寸 / rgb 含む）
  console.log(
    "[modelCreateService] createModelVariationsFromProductBlueprint payload:",
    payload,
  );

  // NewModelVariationPayload[] → CreateModelVariationRequest[] へ変換
  // ★ ここで measurements と productBlueprintId / rgb を丸ごと渡す
  const requests: CreateModelVariationRequest[] = payload.variations.map(
    (v, idx) => {
      const measurements = v.measurements ?? {};

      // rgb(hex) を数値（0xRRGGBB）に変換
      let rgbInt: number | undefined = undefined;
      if (typeof v.rgb === "string") {
        const hex = v.rgb.replace("#", "");
        if (/^[0-9a-fA-F]{6}$/.test(hex)) {
          rgbInt = parseInt(hex, 16);
        } else {
          console.warn(
            "[modelCreateService] invalid rgb hex; skip convert",
            v.rgb,
          );
        }
      } else if (typeof v.rgb === "number") {
        rgbInt = v.rgb;
      }

      console.log("[modelCreateService] map variation → request", {
        index: idx,
        color: v.color,
        sizeLabel: v.sizeLabel,
        modelNumber: v.modelNumber,
        rawRgb: v.rgb,
        rgbInt,
        measurements,
      });

      return {
        productBlueprintId: payload.productBlueprintId,
        modelNumber: v.modelNumber,
        size: v.sizeLabel,
        color: v.color,
        rgb: rgbInt, // backend へ送る値
        measurements,
      };
    },
  );

  console.log(
    "[modelCreateService] mapped CreateModelVariationRequest array:",
    {
      productBlueprintId: payload.productBlueprintId,
      requests,
    },
  );

  // 実際に backend (/models/{productBlueprintId}/variations) を叩く
  await createModelVariations(payload.productBlueprintId, requests);
}
