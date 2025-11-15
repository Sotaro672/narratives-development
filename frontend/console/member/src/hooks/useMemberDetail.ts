// frontend/member/src/hooks/useMemberDetail.ts

import { useCallback, useEffect, useState } from "react";
import type { Member } from "../domain/entity/member";

// ★ バックエンド呼び出し用：Firebase Auth の ID トークンを付与
import { auth } from "../../../shell/src/auth/config/firebaseClient";

// 環境変数からバックエンドのベースURLを取得（末尾スラッシュを除去）
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    ""
  ) ?? "";

export function useMemberDetail(memberId?: string) {
  const [member, setMember] = useState<Member | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(async () => {
    if (!memberId) return;

    setLoading(true);
    setError(null);

    try {
      // 認証トークン
      const token = await auth.currentUser?.getIdToken();
      if (!token) {
        throw new Error("未認証のためメンバー情報を取得できません。");
      }

      const res = await fetch(`${API_BASE}/members/${encodeURIComponent(memberId)}`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
      });

      if (!res.ok) {
        const text = await res.text().catch(() => "");
        throw new Error(`メンバー取得に失敗しました (status ${res.status}) ${text || ""}`);
      }

      const raw = (await res.json()) as Member | null;
      if (!raw) {
        setMember(null);
        return;
      }

      // 姓名が空の場合の正規化（ID にはフォールバックしない）
      const noFirst =
        raw.firstName === null || raw.firstName === undefined || raw.firstName === "";
      const noLast =
        raw.lastName === null || raw.lastName === undefined || raw.lastName === "";

      const normalized: Member = {
        ...raw,
        id: raw.id ?? memberId,
        firstName: noFirst ? null : raw.firstName ?? null,
        lastName: noLast ? null : raw.lastName ?? null,
      };

      setMember(normalized);
    } catch (e: any) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  }, [memberId]);

  useEffect(() => {
    void load();
  }, [load]);

  // PageHeader 用の表示名
  const memberName = (() => {
    if (!member) return "不明なメンバー";
    const full = `${member.lastName ?? ""} ${member.firstName ?? ""}`.trim();
    // ★ 氏名が無い場合は「招待中」と表示し、ID にはフォールバックしない
    return full || "招待中";
  })();

  return {
    member,
    memberName,
    loading,
    error,
    reload: load,
  };
}
