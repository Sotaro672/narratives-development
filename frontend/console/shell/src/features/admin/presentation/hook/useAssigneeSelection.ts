// frontend/console/shell/src/features/admin/presentation/hook/useAssigneeSelection.ts

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import { useAdminCard } from "./useAdminCard";

import type { AssigneeCandidate } from "../../application/AdminService";

export type UseAssigneeSelectionArgs = {
  initialAssigneeId?: string | null;
  initialAssigneeName?: string | null;
  defaultToCurrentMember?: boolean;
};

export type UseAssigneeSelectionResult = {
  assigneeId: string;
  assigneeName: string;
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;
  handleSelectAssignee: (id: string) => void;
  clearAssignee: () => void;
  resetAssignee: () => void;
};

export function useAssigneeSelection(
  args: UseAssigneeSelectionArgs = {},
): UseAssigneeSelectionResult {
  const {
    initialAssigneeId = null,
    initialAssigneeName = null,
    defaultToCurrentMember = true,
  } = args;

  const { currentMember } = useAuthContext();
  const { assigneeCandidates, loadingMembers } = useAdminCard();

  const [assigneeId, setAssigneeId] = useState(initialAssigneeId ?? "");
  const previousInitialAssigneeIdRef = useRef<string | null>(initialAssigneeId);

  /**
   * Backend再取得・保存後のresponseなどによってinitialAssigneeIdが変化した場合、
   * 現在のassigneeIdも最新値へ同期する。
   *
   * 同一initialAssigneeIdでの再renderでは同期しないため、
   * ユーザーが画面上で選択したassigneeIdを不要に上書きしない。
   */
  useEffect(() => {
    const previousInitialAssigneeId = previousInitialAssigneeIdRef.current;

    if (previousInitialAssigneeId === initialAssigneeId) {
      return;
    }

    previousInitialAssigneeIdRef.current = initialAssigneeId;
    setAssigneeId(initialAssigneeId ?? "");
  }, [initialAssigneeId]);

  /**
   * Create画面などinitialAssigneeIdが存在しない場合のみ、
   * currentMemberを初期担当者として設定する。
   */
  useEffect(() => {
    if (assigneeId || !defaultToCurrentMember || !currentMember) {
      return;
    }

    if (!currentMember.id) {
      return;
    }

    setAssigneeId(currentMember.id);
  }, [assigneeId, currentMember, defaultToCurrentMember]);

  const assigneeName = useMemo(() => {
    if (!assigneeId) {
      return "未設定";
    }

    const matched = assigneeCandidates.find(
      (candidate) => candidate.id === assigneeId,
    );

    if (matched) {
      return matched.name;
    }

    if (currentMember?.id === assigneeId) {
      return currentMember.displayName || "未設定";
    }

    if (initialAssigneeId === assigneeId && initialAssigneeName) {
      return initialAssigneeName;
    }

    return "未設定";
  }, [
    assigneeId,
    assigneeCandidates,
    currentMember,
    initialAssigneeId,
    initialAssigneeName,
  ]);

  const handleSelectAssignee = useCallback(
    (id: string) => {
      if (!id) {
        return;
      }

      const isCandidate = assigneeCandidates.some(
        (candidate) => candidate.id === id,
      );
      const isCurrentMember = currentMember?.id === id;

      if (!isCandidate && !isCurrentMember) {
        return;
      }

      setAssigneeId(id);
    },
    [assigneeCandidates, currentMember],
  );

  const clearAssignee = useCallback(() => {
    setAssigneeId("");
  }, []);

  const resetAssignee = useCallback(() => {
    if (initialAssigneeId) {
      setAssigneeId(initialAssigneeId);
      return;
    }

    if (defaultToCurrentMember && currentMember?.id) {
      setAssigneeId(currentMember.id);
      return;
    }

    setAssigneeId("");
  }, [initialAssigneeId, defaultToCurrentMember, currentMember]);

  return {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
    clearAssignee,
    resetAssignee,
  };
}