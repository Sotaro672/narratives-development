/// <reference types="vite/client" />

// シンプルな ID→会社名 取得用リポジトリ（HTTP経由）
// - 最低限のメモリキャッシュ付き
// - バックエンド: GET {API_BASE}/companies/{id} -> { id, name, ... }

const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(/\/+$/, "") ?? "";

// 会社エンティティ（必要最低限）
export type CompanyDTO = {
  id?: string;
  name?: string;
};

// ▼ インメモリキャッシュ（Promise をキャッシュして多重リクエストも束ねる）
const nameCache = new Map<string, Promise<string | null>>();

export async function getCompanyById(companyId: string): Promise<CompanyDTO | null> {
  const id = (companyId ?? "").trim();
  if (!id) return null;

  const res = await fetch(`${API_BASE}/companies/${encodeURIComponent(id)}`, { method: "GET" });
  if (!res.ok) return null;

  const data = (await res.json()) as CompanyDTO;
  return data ?? null;
}

export async function getCompanyNameById(companyId: string): Promise<string | null> {
  const data = await getCompanyById(companyId);
  const name = (data?.name ?? "").trim();
  return name || null;
}

// キャッシュ版（推奨）
export function getCompanyNameByIdCached(companyId: string): Promise<string | null> {
  const id = (companyId ?? "").trim();
  if (!id) return Promise.resolve(null);

  const cached = nameCache.get(id);
  if (cached) return cached;

  const p = getCompanyNameById(id)
    .catch((e) => {
      // 失敗時はキャッシュから削除して呼び出し側で再試行できるようにする
      nameCache.delete(id);
      return null;
    });
  nameCache.set(id, p);
  return p;
}

// 明示的にキャッシュを無効化したい場合
export function clearCompanyNameCache(companyId?: string) {
  if (!companyId) {
    nameCache.clear();
  } else {
    nameCache.delete((companyId ?? "").trim());
  }
}
