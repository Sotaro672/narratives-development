// frontend/amol/src/features/resale/presentation/types/resaleCreatePageTypes.ts

import type {
  MediaUploaderItem,
} from "../../../../components/ui/MediaUploader";

/**
 * ウォレットまたはトークン詳細画面から、
 * 再販出品画面へ渡されるルート状態。
 */
export type ResaleCreatePageLocationState = {
  mintAddress?: string;
  productId?: string;
  brandId?: string;
  brandName?: string;
  productName?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  tokenName?: string;
  tokenIconUrl?: string;
};

/**
 * location.stateを正規化した出品対象情報。
 *
 * 画面および出品処理では、undefinedを扱わず
 * 空文字列へ正規化して利用する。
 */
export type ResaleCreateTarget = {
  mintAddress: string;
  productId: string;
  brandId: string;
  brandName: string;
  productName: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  tokenName: string;
  tokenIconUrl: string;
};

/**
 * 再販商品の状態画像。
 *
 * MediaUploaderItemをベースにしつつ、
 * 再販出品では画像のみを扱うためtypeをimageに限定する。
 * Object URLと元ファイルは必須とする。
 */
export type ResaleConditionMediaItem =
  Omit<
    MediaUploaderItem,
    | "type"
    | "previewUrl"
  > & {
    type: "image";
    previewUrl: string;
    file: File;
  };