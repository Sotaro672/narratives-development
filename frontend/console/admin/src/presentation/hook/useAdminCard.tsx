// frontend/console/admin/src/presentation/hook/useAdminCard.tsx
import { useEffect, useMemo, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import { fetchMemberListWithToken } from "../../../../member/src/infrastructure/query/memberQuery";
import type { MemberFilter } from "../../../../member/src/domain/repository/memberRepository";
import type { Page } from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE } from "../../../../shell/src/shared/types/common/common";

export type UseAdminCardResult = {
  /** 表示中の担当者名（必ず string / fallback 付き） */
  assigneeName: string;

  /** currentMember と同じ companyId のメンバー候補一覧 */
  assigneeCandidates: { id: string; name: string }[];

  /** メンバー一覧取得中フラグ */
  loadingMembers: boolean;

  /** 担当者ポップオーバー開閉状態 */
  openAssigneePopover: boolean;
  setOpenAssigneePopover: (v: boolean) => void;

  /** 一覧から担当者を選択したときのハンドラ */
  onSelectAssignee: (id: string) => void;
};

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();

  // 選択中の担当者 ID（初期値は currentMember.id）
  const [assigneeId, setAssigneeId] = useState<string | null>(
    currentMember?.id ?? null,
  );

  // メンバー ID → 表示名
  const [assigneeNameMap, setAssigneeNameMap] = useState<
    Record<string, string>
  >({});

  // ポップオーバー制御
  const [openAssigneePopover, setOpenAssigneePopover] = useState(false);

  // 一覧取得状態
  const [loadingMembers, setLoadingMembers] = useState(false);
  const [assigneeCandidates, setAssigneeCandidates] = useState<
    { id: string; name: string }[]
  >([]);

  // currentMember が後から取れた場合に初期 assignee を補正
  useEffect(() => {
    if (currentMember?.id && !assigneeId) {
      setAssigneeId(currentMember.id);
    }
  }, [currentMember?.id, assigneeId]);

  // currentMember と同じ companyId の member 一覧を backend から取得
  useEffect(() => {
    (async () => {
      const fbUser = auth.currentUser;
      if (!fbUser) return;

      const token = await fbUser.getIdToken();

      // ★ companyId はここでは送らない：
      //    backend の MemberUsecase.List が ctx から companyId を強制適用する前提
      const page: Page = { ...DEFAULT_PAGE, number: 1, perPage: 200 };
      const filter: MemberFilter = {
        // 必要なら status: "active" などを追加
      };

      setLoadingMembers(true);
      try {
        const { items } = await fetchMemberListWithToken(token, page, filter);

        const candidates = items.map((m) => {
          const full =
            `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim() ||
            (m.email ?? "") ||
            m.id;
          return {
            id: m.id,
            name: full,
          };
        });

        const nameMap: Record<string, string> = {};
        candidates.forEach((c) => {
          nameMap[c.id] = c.name;
        });

        setAssigneeCandidates(candidates);
        setAssigneeNameMap(nameMap);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, []);

  // 表示用の安全な担当者名（必ず string を返す）
  const assigneeName = useMemo(() => {
    if (assigneeId && assigneeNameMap[assigneeId]) {
      return assigneeNameMap[assigneeId];
    }

    // fallback: currentMember の氏名 or email
    const base =
      `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`.trim() ||
      currentMember?.email?.trim();

    return base || "未設定";
  }, [assigneeId, assigneeNameMap, currentMember]);

  // ポップオーバーの一覧から担当者を選択
  const onSelectAssignee = useCallback((id: string) => {
    setAssigneeId(id);
    setOpenAssigneePopover(false);
  }, []);

  return {
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    openAssigneePopover,
    setOpenAssigneePopover,
    onSelectAssignee,
  };
}
