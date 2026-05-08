// frontend\console\list\src\infrastructure\http\list\types.ts

/**
 * =============
 * Types
 * =============
 *
 * backend response を正とする。
 * - snake_case / PascalCase / 旧互換フィールドは持たない
 * - HTTP DTO は backend の JSON field 名に合わせて camelCase へ統一する
 */

/**
 * PriceCard / list detail で使用する価格行。
 *
 * backend response:
 * {
 *   "modelId": "...",
 *   "displayOrder": 1,
 *   "stock": 1,
 *   "size": "ｓ",
 *   "color": "グリーン",
 *   "rgb": 4289797,
 *   "price": 1000
 * }
 */
export type ListDetailPriceRowDTO = {
  modelId: string;
  displayOrder?: number | null;
  stock: number;
  size: string;
  color: string;
  rgb?: number | null;
  price: number | null;
};

export type CreateListInput = {
  /**
   * backend が docId を要求する場合に備えて。
   * 未指定なら inventoryId を採用する想定。
   */
  id?: string;

  /**
   * inventory/list/create から作成する想定。
   * inventoryId は「productBlueprintId__tokenBlueprintId」をそのまま通す。
   * frontend では split しない。
   */
  inventoryId?: string;

  title: string;
  description: string;

  /**
   * PriceCard の rows。
   * backend へ送るのは modelId + price のみ。
   * size / color / stock / rgb は UI 表示用。
   */
  priceRows?: Array<{
    modelId: string;
    price: number | null;

    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  /**
   * 画面の「出品｜保留」。
   * create payload に送らない場合は mapper 側で除外する。
   */
  decision?: "list" | "hold";

  assigneeId?: string;

  /**
   * backend で auth から取るなら省略可。
   */
  createdBy?: string;
};

export type UpdateListInput = {
  listId: string;

  title?: string;
  description?: string;

  /**
   * detail 側の priceRows。
   * backend へ送るのは modelId + price のみ。
   */
  priceRows?: Array<{
    modelId: string;
    price: number | null;

    size?: string;
    color?: string;
    stock?: number;
    rgb?: number | null;
  }>;

  /**
   * UI の "list" | "hold" を backend の status に変換して送る。
   */
  decision?: "list" | "hold";

  assigneeId?: string;

  /**
   * backend で auth から取るなら省略可。
   */
  updatedBy?: string;
};

/**
 * 一覧用 DTO。
 *
 * 必要に応じて後で backend response に合わせて具体化する。
 * ただし、名揺れ吸収用の Record<string, any> は detail には使わない。
 */
export type ListDTO = {
  id: string;
  inventoryId?: string;

  status?: string;
  decision?: string;

  title?: string;
  description?: string;

  assigneeId?: string;
  assigneeName?: string;

  createdBy?: string;
  createdByName?: string;
  createdAt?: string;

  updatedBy?: string;
  updatedByName?: string;
  updatedAt?: string;

  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productBrandId?: string;
  productBrandName?: string;
  productName?: string;

  tokenBrandId?: string;
  tokenBrandName?: string;
  tokenName?: string;

  imageId?: string;
  imageUrls?: string[];

  priceRows?: ListDetailPriceRowDTO[];
  totalStock?: number;
  currencyJpy?: boolean;
};

export type ListAggregateDTO = {
  items: ListDTO[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

/**
 * ListImage DTO
 *
 * Firebase Storage 移行後の正:
 * - frontend が Firebase Storage へ直接 upload する
 * - backend は signed URL / GCS bucket / GCS object を扱わない
 * - /lists/{listId}/images/{imageId} の Firestore record に url / objectPath を保存する
 * - List.ImageID は primary imageId、つまり images サブコレクションの docID
 *
 * backend response / 保存 record の正:
 * {
 *   "id": "imageId",
 *   "url": "Firebase Storage downloadURL",
 *   "objectPath": "lists/{listId}/images/{imageId}/{fileName}",
 *   "fileName": "xxx.jpg",
 *   "contentType": "image/jpeg",
 *   "size": 12345,
 *   "displayOrder": 0,
 *   "createdBy": "uid"
 * }
 */
export type ListImageDTO = {
  id: string;

  /**
   * Firebase Storage getDownloadURL() の戻り値。
   */
  url: string;

  /**
   * Firebase Storage object path。
   */
  objectPath: string;

  fileName?: string;
  contentType?: string;
  size: number;

  displayOrder: number;

  createdBy?: string;
  createdAt?: string;

  updatedBy?: string;
  updatedAt?: string;
};

/**
 * Firebase Storage 直接アップロード後、backend に listImage を登録するための入力。
 *
 * frontend:
 * - Firebase Storage へ uploadBytes / uploadBytesResumable
 * - getDownloadURL() で url を取得
 *
 * backend:
 * - POST /lists/{listId}/images へ url / objectPath / displayOrder などを登録
 */
export type SaveListImageFromFirebaseStorageInput = {
  listId: string;

  /**
   * /lists/{listId}/images/{imageId} の docID。
   */
  id: string;

  /**
   * Firebase Storage getDownloadURL() の戻り値。
   */
  url: string;

  /**
   * Firebase Storage 上の object path。
   */
  objectPath: string;

  size: number;
  displayOrder: number;

  fileName?: string;
  contentType?: string;

  createdBy?: string;
  createdAt?: string;
};

/**
 * ListDetail DTO
 *
 * GET /lists/{listId} response を正とする。
 */
export type ListDetailDTO = {
  id: string;
  inventoryId: string;

  status: string;
  decision: string;

  title: string;
  description: string;

  assigneeId: string;
  assigneeName: string;

  createdBy: string;
  createdByName: string;
  createdAt: string;

  updatedBy?: string;
  updatedByName?: string;
  updatedAt: string;

  productBlueprintId: string;
  tokenBlueprintId: string;

  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  /**
   * primary imageId。
   * backend response に無い場合もあるので optional。
   */
  imageId?: string;

  /**
   * listImage record の url から解決した画像URL配列。
   */
  imageUrls: string[];

  /**
   * PriceCard へ渡す正の価格行。
   */
  priceRows: ListDetailPriceRowDTO[];

  totalStock: number;
  currencyJpy: boolean;
};