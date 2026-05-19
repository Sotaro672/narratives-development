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
 *
 * NOTE:
 * /mint/requests?view=management の軽量 BFF response では、
 * minted ではなく mint:boolean として返る場合がある。
 * Frontend 内部では minted:boolean を正として扱うため、
 * toMint() で mint -> minted に正規化する。
 */

/**
 * フロントエンド用 Mint 型
 * 日付系フィールドは JSON との相性を考慮し string（ISO8601）ベースで扱う。
 */
export type Mint = {
  /** ドキュメント ID */
  id: string;

  /**
   * BFF 互換:
   * /mint/requests?view=management では mint:boolean が返る。
   * UI 判定では minted を正とするため、基本的には minted を使う。
   */
  mint?: boolean | null;

  /** 紐づくブランド ID */
  brandId: string;

  /** 紐づくトークン設計 ID */
  tokenBlueprintId: string;

  /** 表示用トークン名 */
  tokenName?: string | null;

  /** inspectionResults: passed の productId 一覧 */
  products: string[];

  /** 作成日時（ISO8601 文字列） */
  createdAt: string;

  /** 作成者（memberId 等） */
  createdBy: string;

  /** 作成者表示名 */
  createdByName?: string | null;

  /** リクエスト者 ID */
  requestedBy?: string | null;

  /** リクエスト者表示名 */
  requestedByName?: string | null;

  /** ミント完了日時（未ミントの場合は null / undefined） */
  mintedAt?: string | null;

  /** ミント済みフラグ */
  minted: boolean;

  /** 焼却予定日時（未設定の場合は null / undefined） */
  scheduledBurnDate?: string | null;

  /** on-chain tx signature */
  onChainTxSignature?: string | null;
};

function toText(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function toNullableText(value: unknown): string | null {
  const text = toText(value);
  return text ? text : null;
}

function toBool(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.trim().toLowerCase() === "true";
  if (typeof value === "number") return value !== 0;
  return false;
}

function toStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];

  return value
    .map((item) => String(item ?? "").trim())
    .filter((item) => item.length > 0);
}

/**
 * backend から受け取った素の JSON っぽいオブジェクトを
 * Mint 型に「なじませる」ための軽量ヘルパー。
 *
 * 重要:
 * - backend domain mint は minted:boolean
 * - management BFF row は mint:boolean
 * のため、どちらでも minted に正規化する。
 */
export function toMint(raw: any): Mint {
  const id = toText(raw?.id ?? raw?.ID ?? raw?.productionId ?? raw?.ProductionID);

  const tokenBlueprintId = toText(
    raw?.tokenBlueprintId ?? raw?.TokenBlueprintID,
  );

  const minted = Boolean(
    toBool(raw?.minted ?? raw?.Minted) ||
      toBool(raw?.mint ?? raw?.Mint),
  );

  return {
    id,

    mint:
      raw?.mint !== undefined || raw?.Mint !== undefined
        ? toBool(raw?.mint ?? raw?.Mint)
        : null,

    brandId: toText(raw?.brandId ?? raw?.BrandID),

    tokenBlueprintId,

    tokenName: toNullableText(raw?.tokenName ?? raw?.TokenName),

    products: toStringArray(raw?.products ?? raw?.Products),

    createdAt: toText(raw?.createdAt ?? raw?.CreatedAt),

    createdBy: toText(raw?.createdBy ?? raw?.CreatedBy ?? raw?.requestedBy),

    createdByName: toNullableText(raw?.createdByName ?? raw?.CreatedByName),

    requestedBy: toNullableText(raw?.requestedBy ?? raw?.RequestedBy),

    requestedByName: toNullableText(
      raw?.requestedByName ?? raw?.RequestedByName,
    ),

    mintedAt: toNullableText(raw?.mintedAt ?? raw?.MintedAt),

    minted,

    scheduledBurnDate: toNullableText(
      raw?.scheduledBurnDate ?? raw?.ScheduledBurnDate,
    ),

    onChainTxSignature: toNullableText(
      raw?.onChainTxSignature ?? raw?.OnChainTxSignature,
    ),
  };
}