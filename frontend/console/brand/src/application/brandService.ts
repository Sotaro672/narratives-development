// frontend/console/brand/src/application/brandService.ts

/// <reference types="vite/client" />

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";
import { getAuthHeaders } from "../../../shell/src/auth/application/authService";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId?: string | null;  // memberId
  managerName?: string;       // ★ ここに「姓 名」を入れる
  registeredAt: string;       // YYYY/MM/DD
};

// バックエンドから返ってくる Brand の最小形
type Brand = {
  id: string;
  name?: string | null;
  isActive?: boolean | null;
  // ★ JSON 上は manager というキーで返ってくる
  manager?: string | null;
  // 将来「managerId」に変える場合も想定して互換確保
  managerId?: string | null;
  createdAt?: string | null;
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
    // eslint-disable-next-line no-console
    console.log("[brandService] fetchManagerName GET", url);

    const res = await fetch(url, { headers });
    const ct = res.headers.get("content-type") ?? "";
    if (!ct.includes("application/json")) {
      // eslint-disable-next-line no-console
      console.error(
        "[brandService] fetchManagerName unexpected content-type:",
        ct,
      );
      return "";
    }
    if (!res.ok) {
      // eslint-disable-next-line no-console
      console.error(
        "[brandService] fetchManagerName HTTP error:",
        res.status,
      );
      return "";
    }

    const data: any = await res.json();
    const name = formatLastFirst(data.lastName, data.firstName);
    return name;
  } catch (e) {
    // eslint-disable-next-line no-console
    console.error("[brandService] fetchManagerName error:", e);
    return "";
  }
}

// ===========================
// ブランド一覧取得
// ===========================
export async function listBrands(companyId: string): Promise<BrandRow[]> {
  // eslint-disable-next-line no-console
  console.log("[brandService] listBrands start, companyId =", companyId);
  // eslint-disable-next-line no-console
  console.log("[brandService] API_BASE =", API_BASE);

  if (!companyId) return [];

  // ① brandRepository からブランド取得
  const result = await brandRepositoryHTTP.list({
    filter: { companyId },
    sort: { column: "created_at", order: "desc" },
    page: 1,
    perPage: 200,
  });
  // eslint-disable-next-line no-console
  console.log("[brandService] brandRepositoryHTTP.list result =", result);

  const brands = (result.items ?? []) as Brand[];
  // eslint-disable-next-line no-console
  console.log("[brandService] brands =", brands);

  // ② Brand → BrandRow（まず managerId だけ詰める）
  const baseRows: BrandRow[] = brands.map((b) => {
    const rawManager =
      (b.manager ?? b.managerId ?? "").toString().trim() || null;

    const row: BrandRow = {
      id: b.id,
      name: String(b.name ?? "").trim(),
      isActive: !!b.isActive,
      managerId: rawManager,
      registeredAt: formatDateYmd(b.createdAt),
    };

    // eslint-disable-next-line no-console
    console.log(
      "[brandService] mapped BrandRow row id=",
      row.id,
      "managerId=",
      row.managerId,
    );
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
      managerName: idToName.get(mid) ?? "",
    };
  });

  // eslint-disable-next-line no-console
  console.log("[brandService] rowsWithName =", rowsWithName);
  return rowsWithName;
}
