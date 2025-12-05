// frontend/console/admin/src/presentation/hook/useAdminCard.tsx
import { useEffect, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ ID → 「姓 名」を backend の /members/{id}/display-name で解決するフック
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

// ★ Admin 用アプリケーションサービス
//   - ListMembersByCompanyID → displayName 付きメンバー一覧を返す想定
import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeId: string | null; // ← そのまま保持（重要）
  assigneeName: string; // 表示用のみ（getDisplayName or fallback）
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;

  openAssigneePopover: boolean;
  setOpenAssigneePopover: (v: boolean) => void;

  onSelectAssignee: (id: string) => void;
};

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();
  const { getNameLastFirstByID } = useMemberList(); // ★ backend の getDisplayName を叩く

  // ★ assigneeId — 選択 ID のルートは従来のまま完全維持
  const [assigneeId, setAssigneeId] = useState<string | null>(
    currentMember?.id ?? null,
  );

  // 一覧取得時のローカルキャッシュ（id → name）
  const [assigneeNameMap, setAssigneeNameMap] =
    useState<Record<string, string>>({});

  // 画面表示専用 — 名前文字列
  const [assigneeName, setAssigneeName] = useState<string>("未設定");

  const [openAssigneePopover, setOpenAssigneePopover] = useState(false);

  const [loadingMembers, setLoadingMembers] = useState(false);
  const [assigneeCandidates, setAssigneeCandidates] = useState<
    AssigneeCandidate[]
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
      setLoadingMembers(true);
      try {
        const { candidates, nameMap } =
          await fetchAssigneeCandidatesForCurrentCompany();

        // ★ backend(ListMembersByCompanyID → displayName 付き)の結果をそのまま UI 用 state に格納
        setAssigneeCandidates(candidates);
        setAssigneeNameMap(nameMap);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, []);

  // ★ assigneeId → assigneeName の解決（UI 表示専用）
  //   1. backend の /members/{id}/display-name を叩く（getNameLastFirstByID）
  //   2. 失敗した場合のみ一覧キャッシュ（assigneeNameMap）を見る
  //   3. それも無ければ currentMember からの fallback
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
        // 1. backend の getDisplayName を叩く（useMemberList → /members/{id}/display-name）
        const resolved = await getNameLastFirstByID(id);
        if (!cancelled && resolved) {
          setAssigneeName(resolved);
          return;
        }
      } catch {
        // 無視して fallback ルートへ
      }

      // 2. 一覧キャッシュから得られれば使う
      if (!cancelled && assigneeNameMap[id]) {
        setAssigneeName(assigneeNameMap[id]);
        return;
      }

      // 3. fallback
      if (!cancelled) {
        setAssigneeName(fallback);
      }
    };

    void resolveName();
    return () => {
      cancelled = true;
    };
  }, [assigneeId, assigneeNameMap, currentMember, getNameLastFirstByID]);

  // ★ AdminCard から assigneeId を更新する
  const onSelectAssignee = useCallback((id: string) => {
    setAssigneeId(id); // ← assigneeId ルートは維持
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
