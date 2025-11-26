// frontend/console/admin/src/presentation/hook/useAdminCard.tsx
import { useEffect, useMemo, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { fetchMemberListWithToken } from "../../../../member/src/infrastructure/query/memberQuery";
import type { MemberFilter } from "../../../../member/src/domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE } from "../../../../shell/src/shared/types/common/common";

// ★ ID → 「姓 名」を解決するフック
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

export type UseAdminCardResult = {
  assigneeId: string | null;            // ← そのまま保持（重要）
  assigneeName: string;                  // 表示用のみ
  assigneeCandidates: { id: string; name: string }[];
  loadingMembers: boolean;

  openAssigneePopover: boolean;
  setOpenAssigneePopover: (v: boolean) => void;

  onSelectAssignee: (id: string) => void;
};

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();

  const { getNameLastFirstByID } = useMemberList();

  // ★ assigneeId — 選択 ID のルートは従来のまま完全維持
  const [assigneeId, setAssigneeId] = useState<string | null>(
    currentMember?.id ?? null
  );

  // 一覧取得時のローカルキャッシュ
  const [assigneeNameMap, setAssigneeNameMap] = useState<Record<string, string>>(
    {}
  );

  // 画面表示専用 — getNameLastFirstByID の結果
  const [assigneeName, setAssigneeName] = useState<string>("未設定");

  const [openAssigneePopover, setOpenAssigneePopover] = useState(false);

  const [loadingMembers, setLoadingMembers] = useState(false);
  const [assigneeCandidates, setAssigneeCandidates] = useState<
    { id: string; name: string }[]
  >([]);

  // currentMember の id が後から来るケースに備える
  useEffect(() => {
    if (currentMember?.id && !assigneeId) {
      setAssigneeId(currentMember.id);
    }
  }, [currentMember?.id, assigneeId]);

  // メンバー一覧取得（companyId は backend が ctx から付与）
  useEffect(() => {
    (async () => {
      const fbUser = auth.currentUser;
      if (!fbUser) return;

      const token = await fbUser.getIdToken();
      const page: Page = { ...DEFAULT_PAGE, number: 1, perPage: 200 };
      const filter: MemberFilter = {};

      setLoadingMembers(true);
      try {
        const { items } = await fetchMemberListWithToken(token, page, filter);

        const candidates = items.map((m) => {
          const full =
            `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim() ||
            m.email ||
            m.id;
          return { id: m.id, name: full };
        });

        const nameMap: Record<string, string> = {};
        candidates.forEach((c) => (nameMap[c.id] = c.name));

        setAssigneeCandidates(candidates);
        setAssigneeNameMap(nameMap);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, []);

  // ★ assigneeId → assigneeName の解決（UI 表示専用）
  useEffect(() => {
    let cancelled = false;

    const resolveName = async () => {
      const id = (assigneeId ?? "").trim();
      const fallback =
        `${currentMember?.lastName ?? ""} ${
          currentMember?.firstName ?? ""
        }`.trim() ||
        currentMember?.email ||
        "未設定";

      if (!id) {
        if (!cancelled) setAssigneeName(fallback);
        return;
      }

      try {
        // 1. getNameLastFirstByID（非同期）
        const resolved = await getNameLastFirstByID(id);
        if (!cancelled && resolved) {
          setAssigneeName(resolved);
          return;
        }
      } catch (_) {
        /* 無視して fallback に進む */
      }

      // 2. 一覧キャッシュから得られれば使う
      if (!cancelled && assigneeNameMap[id]) {
        setAssigneeName(assigneeNameMap[id]);
        return;
      }

      // 3. fallback
      if (!cancelled) setAssigneeName(fallback);
    };

    void resolveName();
    return () => {
      cancelled = true;
    };
  }, [assigneeId, assigneeNameMap, currentMember, getNameLastFirstByID]);

  // ★ AdminCard から assigneeId を更新する
  const onSelectAssignee = useCallback((id: string) => {
    setAssigneeId(id);                // ← assigneeId ルートは維持
    setOpenAssigneePopover(false);
  }, []);

  return {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    openAssigneePopover,
    setOpenAssigneePopover,
    onSelectAssignee,
  };
}
