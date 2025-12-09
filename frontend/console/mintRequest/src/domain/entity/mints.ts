// frontend/console/mintRequest/src/domain/entity/mints.ts

/**
 * Mint エンティティ (mints テーブル 1 レコード)
 *
 * backend/internal/domain/mint/entity.go の Mint 構造体に対応するフロント側型。
 *
 * 想定テーブル構造:
 * - id                 : string
 * - brandId            : string
 * - tokenBlueprintId   : string
 * - products           : string[]
 * - createdAt          : string (ISO8601 文字列想定)
 * - createdBy          : string
 * - mintedAt           : string | null
 * - minted             : boolean
 * - scheduledBurnDate  : string | null
 */

/**
 * フロントエンド用 Mint 型
 * 日付系フィールドは JSON との相性を考慮し string（ISO8601）ベースで扱う。
 */
export type Mint = {
  /** ドキュメント ID */
  id: string;

  /** 紐づくブランド ID（必須） */
  brandId: string;

  /** 紐づくトークン設計 ID（必須） */
  tokenBlueprintId: string;

  /** inspectionResults: passed の productId 一覧（必須 / 空配列不可） */
  products: string[];

  /** 作成日時（ISO8601 文字列） */
  createdAt: string;

  /** 作成者（memberId 等） */
  createdBy: string;

  /** ミント完了日時（未ミントの場合は null / undefined） */
  mintedAt?: string | null;

  /** ミント済みフラグ */
  minted: boolean;

  /** 焼却予定日時（未設定の場合は null / undefined） */
  scheduledBurnDate?: string | null;
};

/**
 * backend から受け取った素の JSON っぽいオブジェクトを
 * Mint 型に「なじませる」ための軽量ヘルパー。
 * （必要になったら利用。現状は単純にフィールドを写しているだけ）
 */
export function toMint(raw: any): Mint {
  return {
    id: String(raw.id ?? raw.ID ?? ""),
    brandId: String(raw.brandId ?? raw.BrandID ?? ""),
    tokenBlueprintId: String(
      raw.tokenBlueprintId ?? raw.TokenBlueprintID ?? "",
    ),
    products: Array.isArray(raw.products)
      ? raw.products.map((p: any) => String(p))
      : [],
    createdAt: String(raw.createdAt ?? raw.CreatedAt ?? ""),
    createdBy: String(raw.createdBy ?? raw.CreatedBy ?? ""),
    mintedAt:
      raw.mintedAt ?? raw.MintedAt
        ? String(raw.mintedAt ?? raw.MintedAt)
        : null,
    minted: Boolean(raw.minted ?? raw.Minted),
    scheduledBurnDate:
      raw.scheduledBurnDate ?? raw.ScheduledBurnDate
        ? String(raw.scheduledBurnDate ?? raw.ScheduledBurnDate)
        : null,
  };
}
