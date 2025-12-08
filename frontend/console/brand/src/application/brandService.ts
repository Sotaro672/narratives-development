// frontend/console/brand/src/application/brandService.ts
/// <reference types="vite/client" />

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";
import { getAuthHeaders } from "../../../shell/src/auth/application/authService";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId?: string | null; // memberId
  managerName?: string; // 「姓 名」
  registeredAt: string; // YYYY/MM/DD
  updatedAt: string; // YYYY/MM/DD 追加
};

// バックエンドから返ってくる Brand の最小形
type Brand = {
  id: string;
  name?: string | null;
  isActive?: boolean | null;
  // JSON 上は manager / managerId どちらでも来る可能性を考慮
  manager?: string | null;
  managerId?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null; // 追加
};

// ===========================
// 共通ユーティリティ
// ===========================

// ISO → YYYY/MM/DD
function formatDateYmd(iso?: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
}

// 「姓 → 名」の表示
function formatLastFirst(last?: string | null, first?: string | null): string {
  const ln = (last ?? "").trim();
  const fn = (first ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

// backend base URL（/members を叩く用）
// 1. .env.local の VITE_BACKEND_BASE_URL を読む
// 2. 取れなかった場合は Cloud Run の URL にフォールバック
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const API_BASE =
  ENV_BASE || "https://narratives-backend-871263659099.asia-northeast1.run.app";

// managerId(memberId) → 「姓 名」を backend /members/:id 経由で取得
async function fetchManagerName(memberId: string): Promise<string> {
  const id = (memberId ?? "").trim();
  if (!id) return "";

  try {
    const headers = await getAuthHeaders();
    const url = `${API_BASE}/members/${encodeURIComponent(id)}`;

    const res = await fetch(url, { headers });
    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) {
      return "";
    }
    if (!res.ok) {
      return "";
    }

    const data: any = await res.json();
    const name = formatLastFirst(data.lastName, data.firstName);
    return name;
  } catch {
    return "";
  }
}

// backend brand.Service.FormatName と揃えた brand 名整形関数
export function formatBrandName(name: string | null | undefined): string {
  return (name ?? "").trim();
}

// ===========================
// companyId のブランド一覧取得
//   - 実際のフィルタリングは backend BrandUsecase.List + companyIDFromContext に統一
//   - フロントでは companyId は「まだログイン情報が取れていない場合は呼ばない」ためのガード用途のみ
// ===========================
export async function listBrands(companyId: string): Promise<BrandRow[]> {
  // companyId が空の間は呼ばない（認証コンテキスト未準備のガード）
  if (!companyId) return [];

  // ① brandRepository からブランド取得
  //    ※ companyId での絞り込みは backend 側 (BrandUsecase.List + companyIDFromContext) が担当
  const page = await brandRepositoryHTTP.list({
    filter: {
      // companyId は渡さない（フロント側でのフィルタリングは削除）
      deleted: false,
      isActive: true,
    },
    sort: { column: "created_at", order: "desc" },
    page: 1,
    perPage: 200,
  });

  const brands = (page.items ?? []) as Brand[];

  // ② Brand → BrandRow（まず managerId / name / registeredAt / updatedAt だけ詰める）
  const baseRows: BrandRow[] = brands.map((b) => {
    const rawManager =
      (b.manager ?? b.managerId ?? "").toString().trim() || null;

    const row: BrandRow = {
      id: b.id,
      name: formatBrandName(b.name ?? ""), // ★ backend FormatName と揃えた整形
      isActive: !!b.isActive,
      managerId: rawManager,
      registeredAt: formatDateYmd(b.createdAt),
      updatedAt: formatDateYmd(b.updatedAt), // ★ 追加
    };

    return row;
  });

  // ③ managerId 一覧を抽出
  const managerIds = Array.from(
    new Set(
      baseRows
        .map((r) => (r.managerId ?? "").trim())
        .filter((v) => v !== ""),
    ),
  );

  // ④ managerId → managerName を backend /members/:id で解決
  const idToName = new Map<string, string>();
  await Promise.all(
    managerIds.map(async (mid) => {
      const name = await fetchManagerName(mid);
      if (name) {
        idToName.set(mid, name);
      }
    }),
  );

  // ⑤ managerName を埋めて返却
  const rowsWithName: BrandRow[] = baseRows.map((r) => {
    const mid = (r.managerId ?? "").trim();
    return {
      ...r,
      managerName: mid ? idToName.get(mid) ?? "" : "",
    };
  });

  return rowsWithName;
}

// ===========================
// Solana brand wallet 開設リクエスト
// ※ backend 側に /brands/:id/wallet のようなエンドポイントを用意しておき、
//   既存 brand に対してウォレットを後付けで開設したい場合などに利用する想定。
//   （ブランド新規作成時は BrandUsecase.Create 内で自動開設される設計）
// ===========================
export async function requestOpenSolanaWalletForBrand(
  brandId: string,
): Promise<void> {
  const id = (brandId ?? "").trim();
  if (!id) return;

  const headers = await getAuthHeaders();
  await fetch(`${API_BASE}/brands/${encodeURIComponent(id)}/wallet`, {
    method: "POST",
    headers,
  }).catch(() => {
    // 通信エラー時も UI は止めず、エラー表示等は呼び出し側で実装する想定
  });
}
