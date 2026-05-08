// frontend/console/inventory/src/presentation/hook/listCreate/useAssignee.ts
import * as React from "react";

// ★ Admin 用 hook（担当者候補の取得）
import { useAdminCard as useAdminCardHook } from "../../../../../admin/src/presentation/hook/useAdminCard";
import { useAuth } from "../../../../../shell/src/auth/presentation/hook/useCurrentMember";

export function useAssignee(): {
  assigneeName: string;
  assigneeCandidates: Array<{ id: string; name: string }>;
  loadingMembers: boolean;
  assigneeId: string | undefined;
  handleSelectAssignee: (id: string) => void;
} {
  const { assigneeCandidates, loadingMembers } = useAdminCardHook();
  const { currentMember } = useAuth();

  const normalizedCandidates = React.useMemo(
    () => (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    [assigneeCandidates],
  );

  const [assigneeId, setAssigneeId] = React.useState<string | undefined>(undefined);
  const [assigneeName, setAssigneeName] = React.useState("");

  React.useEffect(() => {
    if (assigneeId) return;
    if (!currentMember) return;

    const currentMemberId = String(currentMember.id ?? "");
    const currentMemberName =
      currentMember.fullName || currentMember.email || currentMemberId;

    setAssigneeId(currentMemberId);
    setAssigneeName(currentMemberName);
  }, [currentMember, assigneeId]);

  React.useEffect(() => {
    if (!assigneeId) return;

    const matched = normalizedCandidates.find((c) => c.id === assigneeId);
    if (matched) {
      setAssigneeName(matched.name);
      return;
    }

    if (currentMember?.id === assigneeId) {
      const fallbackName =
        currentMember.fullName || currentMember.email || currentMember.id;
      setAssigneeName(fallbackName);
    }
  }, [normalizedCandidates, assigneeId, currentMember]);

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "");
      if (!nextId) return;

      setAssigneeId(nextId);

      const matched = normalizedCandidates.find((c) => c.id === nextId);
      if (matched) {
        setAssigneeName(matched.name);
        return;
      }

      if (currentMember?.id === nextId) {
        const fallbackName =
          currentMember.fullName || currentMember.email || currentMember.id;
        setAssigneeName(fallbackName);
        return;
      }

      setAssigneeName("");
    },
    [normalizedCandidates, currentMember],
  );

  return {
    assigneeName,
    assigneeCandidates: normalizedCandidates,
    loadingMembers: Boolean(loadingMembers),
    assigneeId,
    handleSelectAssignee,
  };
}