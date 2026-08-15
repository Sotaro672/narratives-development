// frontend/amol/src/features/list/types/list.ts

import type {
  PageResult,
} from "../pageResult";

/**
 * 商品一覧APIで返される価格情報です。
 *
 * APIの互換性を考慮し、価格はamountまたはpriceを許容します。
 */
export type ListPriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

/**
 * GET /mall/listsで返される商品一覧の項目です。
 */
export type MallListItem = {
  id: string;
  title: string;
  description: string;
  image: string;
  prices: ListPriceRow[];

  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

/**
 * GET /mall/listsのレスポンスです。
 */
export type MallListIndexResponse =
  PageResult<MallListItem>;

/**
 * カード表示の補完に使用する商品設計情報です。
 */
export type CatalogProductBlueprint = {
  id?: string;
  productName?: string;
  brandName?: string;
};

/**
 * GET /mall/catalog/:listIdのうち、
 * 商品一覧カードで使用する項目だけを定義します。
 */
export type MallCatalogResponse = {
  productBlueprint?: CatalogProductBlueprint;
};

/**
 * 商品一覧画面で表示するカード用データです。
 */
export type MallListCardItem =
  MallListItem & {
    productName?: string;
    brandName?: string;
  };

/**
 * 一覧画面の読み込み結果です。
 */
export type LoadListPageResult = {
  items: MallListCardItem[];
  page: number;
  totalPages: number;
};