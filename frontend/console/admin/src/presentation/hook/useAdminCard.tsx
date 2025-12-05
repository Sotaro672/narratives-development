// frontend/console/admin/src/presentation/hook/useAdminCard.tsx
import { useEffect, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ID → displayName 取得
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

// AdminService
import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeId: string | null;
  assigneeName: string;
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;

  openAssigneePopover: boolean;
  setOpenAssigneePopover: (v: boolean) => void;

  onSelectAssignee: (id: string) => void;
};

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();
  const { getNameLastFirstByID } = useMemberList();

  // ★ 初期値は null（未設定）
  const [assigneeId, setAssigneeId] = useState<string | null>(null);

  const [assigneeNameMap, setAssigneeNameMap] =
    useState<Record<string, string>>({});

  const [assigneeName, setAssigneeName] = useState<string>("未設定");

  const [openAssigneePopover, setOpenAssigneePopover] = useState(false);
  const [loadingMembers, setLoadingMembers] = useState(false);

  const [assigneeCandidates, setAssigneeCandidates] = useState<
    AssigneeCandidate[]
  >([]);

  // ★ 初期 assignee を currentMember.id に合わせて自動設定しない（削除済み）

  // メンバー一覧をロード
  useEffect(() => {
    (async () => {
      setLoadingMembers(true);
      try {
        const { candidates, nameMap } =
          await fetchAssigneeCandidatesForCurrentCompany();

        setAssigneeCandidates(candidates);
        setAssigneeNameMap(nameMap);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, []);

  // assigneeId → 名前の解決
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
        if (!cancelled) setAssigneeName("未設定");
        return;
      }

      try {
        const resolved = await getNameLastFirstByID(id);
        if (!cancelled && resolved) {
          setAssigneeName(resolved);
          return;
        }
      } catch {}

      if (!cancelled && assigneeNameMap[id]) {
        setAssigneeName(assigneeNameMap[id]);
        return;
      }

      if (!cancelled) setAssigneeName(fallback);
    };

    void resolveName();
    return () => {
      cancelled = true;
    };
  }, [assigneeId, assigneeNameMap, currentMember, getNameLastFirstByID]);

  // 選択された assigneeId の更新
  const onSelectAssignee = useCallback((id: string) => {
    setAssigneeId(id);
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
