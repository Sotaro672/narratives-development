// frontend/console/inventory/src/presentation/hook/listCreate/useAssignee.ts
import * as React from "react";

// ★ Admin 用 hook（担当者候補の取得・選択）
import { useAdminCard as useAdminCardHook } from "../../../../../admin/src/presentation/hook/useAdminCard";

export function useAssignee(): {
  assigneeName: string;
  assigneeCandidates: Array<{ id: string; name: string }>;
  loadingMembers: boolean;
  assigneeId: string | undefined;
  handleSelectAssignee: (id: string) => void;
} {
  const { assigneeName, assigneeCandidates, loadingMembers, onSelectAssignee } =
    useAdminCardHook();

  const [assigneeId, setAssigneeId] = React.useState<string | undefined>(undefined);

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      setAssigneeId(id || undefined);
      onSelectAssignee(id);
    },
    [onSelectAssignee],
  );

  return {
    assigneeName,
    assigneeCandidates: (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    loadingMembers: Boolean(loadingMembers),
    assigneeId,
    handleSelectAssignee,
  };
}
