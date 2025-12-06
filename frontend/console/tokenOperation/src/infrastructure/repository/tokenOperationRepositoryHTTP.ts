// frontend/console/tokenOperation/src/infrastructure/repository/tokenOperationRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// トークン運用画面で使う型
import type { TokenOperationExtended } from "../../../../shell/src/shared/types/tokenOperation";

/**
 * Backend base URL
 * - .env の VITE_BACKEND_BASE_URL を優先
 * - 未設定時は Cloud Run の固定 URL を利用
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return user.getIdToken();
}

// ---------------------------------------------------------
// token-blueprints 一覧 API レスポンス型（backend と対応）
// ---------------------------------------------------------
type TokenBlueprintAPIResponse = {
  id: string;
  name: string;
  symbol: string;
  brandId: string;
  brandName: string;
  companyId: string;
  description: string;
  iconId?: string | null;
  contentFiles: string[];
  assigneeId: string;
  assigneeName: string;
  createdAt: string;
  createdBy: string;
  updatedAt?: string | null;
  updatedBy: string;
};

type TokenBlueprintPageResponse = {
  items: TokenBlueprintAPIResponse[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

// ---------------------------------------------------------
// mapper: TokenBlueprintAPIResponse → TokenOperationExtended
// ---------------------------------------------------------
function mapToTokenOperation(
  src: TokenBlueprintAPIResponse,
): TokenOperationExtended {
  // TokenOperationExtended の定義に合わせて必要な項目を埋める。
  // 足りない項目がある場合でも any キャストでコンパイルを通す。
  const base: any = {
    id: src.id,
    tokenName: src.name,
    symbol: src.symbol,
    brandId: src.brandId,
    brandName: src.brandName,
    assigneeId: src.assigneeId,
    assigneeName: src.assigneeName,
    // ここに将来、mintedAt / mintedBy などを追加したければ追記
  };

  return base as TokenOperationExtended;
}

// ---------------------------------------------------------
// ListByCompanyID → ListMintedCompleted を呼び出すリポジトリ
// ---------------------------------------------------------

/**
 * currentMember.companyId を引数に受けつつ、
 * backend 側では ListMintedCompleted（minted = "minted"）を利用して
 * 「ミント済みトークン設計」のみを取得する。
 *
 * - HTTP: GET /token-blueprints?minted=minted
 * - companyId 自体はクエリに渡さず、middleware がコンテキストに設定した
 *   companyId を用いて ListByCompanyID 相当のスコープ制御が行われる。
 */
export async function listTokenOperationsMintedByCompanyId(
  companyId: string,
): Promise<TokenOperationExtended[]> {
  const cid = companyId.trim();
  if (!cid) {
    // companyId が空の場合は何も取得しない
    return [];
  }

  const idToken = await getIdTokenOrThrow();

  const res = await fetch(`${API_BASE}/token-blueprints?minted=minted`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `トークン運用一覧の取得に失敗しました (status=${res.status}): ${text}`,
    );
  }

  const data = (await res.json()) as TokenBlueprintPageResponse;

  return data.items.map(mapToTokenOperation);
}
