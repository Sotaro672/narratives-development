// frontend/console/shell/src/auth/application/useAuth.ts
import { useEffect, useMemo, useState } from "react";
import { useAuthContext } from "./AuthContext";

// env から API ベース URL を取得（末尾スラッシュ除去）
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    "",
  ) ?? "";

/**
 * useAuth
 * - AuthContext の値に加えて、user.companyId から会社名を取得して返す
 * - 返り値に companyName / loadingCompanyName / companyError を追加
 */
export function useAuth() {
  const ctx = useAuthContext();
  const companyId = ctx.user?.companyId ?? null;

  const [companyName, setCompanyName] = useState<string | null>(null);
  const [loadingCompanyName, setLoadingCompanyName] = useState(false);
  const [companyError, setCompanyError] = useState<string | null>(null);

  useEffect(() => {
    // user or companyId が変わったら会社名を再取得
    if (!companyId) {
      setCompanyName(null);
      setCompanyError(null);
      setLoadingCompanyName(false);
      return;
    }

    // ここで string に確定させる（以降 encodeURIComponent に安全に渡せる）
    const cid: string = companyId as string;

    let aborted = false;
    const ac = new AbortController();

    async function fetchCompanyName() {
      setLoadingCompanyName(true);
      setCompanyError(null);

      try {
        // 例: GET {API_BASE}/companies/{id} -> { id, name, ... }
        const res = await fetch(
          `${API_BASE}/companies/${encodeURIComponent(cid)}`,
          {
            method: "GET",
            signal: ac.signal,
          },
        );

        if (!res.ok) {
          const text = await res.text().catch(() => "");
          throw new Error(
            `failed to fetch company: ${res.status} ${res.statusText} ${text}`,
          );
        }

        const data = (await res.json()) as { id?: string; name?: string };
        if (!aborted) {
          setCompanyName((data?.name ?? "").trim() || null);
        }
      } catch (e: any) {
        if (!aborted) {
          setCompanyName(null);
          setCompanyError(e?.message ?? "failed to fetch company name");
        }
      } finally {
        if (!aborted) setLoadingCompanyName(false);
      }
    }

    fetchCompanyName();

    return () => {
      aborted = true;
      ac.abort();
    };
  }, [companyId]);

  // 既存の ctx に companyName 等を足して返す
  return useMemo(
    () => ({
      ...ctx,
      companyName,
      loadingCompanyName,
      companyError,
    }),
    [ctx, companyName, loadingCompanyName, companyError],
  );
}
