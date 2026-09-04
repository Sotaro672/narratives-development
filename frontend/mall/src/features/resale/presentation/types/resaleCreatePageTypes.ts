// frontend/amol/src/features/resale/presentation/types/resaleCreatePageTypes.ts

import type {
  MediaUploaderItem,
} from "../../../../components/ui/MediaUploader";

/**
 * ウォレットまたはトークン詳細画面から、
 * 再販出品画面へ渡されるルート状態。
 */
export type ResaleCreatePageLocationState = {
  assetId?: string;
  productId?: string;
  brandName?: string;
  productName?: string;
  tokenBlueprintId?: string;
  tokenName?: string;
  tokenIconUrl?: string;
  tokenDescription?: string;
};

export type ResaleCreateTarget = {
  assetId: string;
  productId: string;
  brandName: string;
  productName: string;
  tokenBlueprintId: string;
  tokenName: string;
  tokenIconUrl: string;
  tokenDescription: string;
};

/**
 * 再販商品の状態画像。
 */
export type ResaleConditionMediaItem =
  Omit<MediaUploaderItem, "type" | "previewUrl"> & {
    type: "image";
    previewUrl: string;
    file: File;
  };