// frontend/member/src/hooks/useMemberDetail.ts

import { useCallback, useEffect, useState } from "react";
import type { Member } from "../domain/entity/member";
import type { MemberRepository } from "../domain/repository/memberRepository";
import { MemberRepositoryFS } from "../infrastructure/firestore/memberRepositoryFS";

const repository: MemberRepository = new MemberRepositoryFS();

export function useMemberDetail(memberId?: string) {
  const [member, setMember] = useState<Member | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(async () => {
    if (!memberId) return;

    setLoading(true);
    setError(null);

    try {
      const raw = await repository.getById(memberId); // Member | null 期待

      if (!raw) {
        setMember(null);
        return;
      }

      const item = raw as Member;

      const noFirst =
        item.firstName === null ||
        item.firstName === undefined ||
        item.firstName === "";
      const noLast =
        item.lastName === null ||
        item.lastName === undefined ||
        item.lastName === "";

      // ★ 名前が無い場合も、id を名前としては使わない（id はあくまで識別子）
      const normalized: Member = {
        ...item,
        id: item.id ?? memberId,
        firstName: noFirst ? null : item.firstName ?? null,
        lastName: noLast ? null : item.lastName ?? null,
      };

      setMember(normalized);
    } catch (e: any) {
      setError(e);
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

    const full = `${member.lastName ?? ""} ${member.firstName ?? ""}`
      .trim();

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
