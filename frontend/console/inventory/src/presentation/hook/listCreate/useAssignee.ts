// frontend/console/inventory/src/presentation/hook/listCreate/useAssignee.ts
import * as React from "react";

import { useAdminCard as useAdminCardHook } from "../../../../../admin/src/presentation/hook/useAdminCard";
import { useAuth } from "../../../../../shell/src/auth/presentation/hook/useCurrentMember";

type AssigneeCandidate = {
  id: string;
  name: string;
};

function getMemberUid(member: unknown): string {
  const m = member as any;

  return String(m?.uid ?? "");
}

function getMemberDisplayName(member: unknown): string {
  const m = member as any;

  const displayName = String(m?.displayName ?? "");
  if (displayName) return displayName;

  const nameParts = [m?.lastName, m?.firstName]
    .map((v) => String(v ?? ""))
    .filter(Boolean);

  const joinedName = nameParts.join(" ");
  if (joinedName) return joinedName;

  const email = String(m?.email ?? "");
  if (email) return email;

  const uid = getMemberUid(member);
  if (uid) return uid;

  return String(m?.id ?? "");
}

function normalizeAssigneeCandidates(rawCandidates: unknown): AssigneeCandidate[] {
  const rows = Array.isArray(rawCandidates) ? rawCandidates : [];

  return rows
    .map((raw) => {
      const c = raw as any;

      const id = String(c?.uid ?? c?.id ?? "");
      if (!id) return null;

      const displayName = String(c?.displayName ?? "");

      const nameParts = [c?.lastName, c?.firstName]
        .map((v) => String(v ?? ""))
        .filter(Boolean);

      const joinedName = nameParts.join(" ");
      const name = displayName || joinedName || String(c?.email ?? "") || id;

      return {
        id,
        name,
      };
    })
    .filter(Boolean) as AssigneeCandidate[];
}

export function useAssignee(): {
  assigneeName: string;
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;
  assigneeId: string | undefined;
  handleSelectAssignee: (id: string) => void;
} {
  const { assigneeCandidates: rawAssigneeCandidates, loadingMembers } =
    useAdminCardHook();
  const { currentMember } = useAuth();

  const normalizedCandidates = React.useMemo(
    () => normalizeAssigneeCandidates(rawAssigneeCandidates),
    [rawAssigneeCandidates],
  );

  const [assigneeId, setAssigneeId] = React.useState<string | undefined>(
    undefined,
  );
  const [assigneeName, setAssigneeName] = React.useState("");

  React.useEffect(() => {
    if (assigneeId) return;
    if (!currentMember) return;

    const currentMemberUid = getMemberUid(currentMember);
    if (!currentMemberUid) return;

    setAssigneeId(currentMemberUid);
    setAssigneeName(getMemberDisplayName(currentMember));
  }, [currentMember, assigneeId]);

  React.useEffect(() => {
    if (!assigneeId) return;

    const matched = normalizedCandidates.find((c) => c.id === assigneeId);
    if (matched) {
      setAssigneeName(matched.name);
      return;
    }

    if (getMemberUid(currentMember) === assigneeId) {
      setAssigneeName(getMemberDisplayName(currentMember));
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

      if (getMemberUid(currentMember) === nextId) {
        setAssigneeName(getMemberDisplayName(currentMember));
        return;
      }

      setAssigneeName(nextId);
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