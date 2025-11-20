// frontend/member/src/hooks/useMemberDetail.ts

import { useCallback, useEffect, useState } from "react";
import type { Member } from "../../domain/entity/member";

// アプリケーションサービスへ委譲
import { fetchMemberDetail } from "../../application/memberService";

export function useMemberDetail(memberId?: string) {
  const [member, setMember] = useState<Member | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(async () => {
    if (!memberId) return;

    setLoading(true);
    setError(null);

    try {
      const normalized = await fetchMemberDetail(memberId);
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
