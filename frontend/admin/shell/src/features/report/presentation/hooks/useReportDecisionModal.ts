// frontend/admin/shell/src/features/report/presentation/hooks/useReportDecisionModal.ts

import { useCallback, useState } from "react";

type DecisionAction = (
  decisionReason: string,
) => Promise<unknown | null>;

type UseReportDecisionModalParams = {
  canDecide: boolean;
  deciding: boolean;
  keep: DecisionAction;
  remove: DecisionAction;
};

export function useReportDecisionModal({
  canDecide,
  deciding,
  keep,
  remove,
}: UseReportDecisionModalParams) {
  const [decisionReason, setDecisionReason] = useState("");
  const [decisionModalOpen, setDecisionModalOpen] = useState(false);
  const [decisionAttempted, setDecisionAttempted] = useState(false);

  const resetDecisionState = useCallback(() => {
    setDecisionReason("");
    setDecisionAttempted(false);
  }, []);

  const openDecisionModal = useCallback(() => {
    if (!canDecide) {
      return;
    }

    resetDecisionState();
    setDecisionModalOpen(true);
  }, [canDecide, resetDecisionState]);

  const closeDecisionModal = useCallback(() => {
    if (deciding) {
      return;
    }

    setDecisionModalOpen(false);
    resetDecisionState();
  }, [deciding, resetDecisionState]);

  const handleDecisionSuccess = useCallback(() => {
    setDecisionModalOpen(false);
    resetDecisionState();
  }, [resetDecisionState]);

  const handleKeep = useCallback(async () => {
    setDecisionAttempted(true);

    const result = await keep(decisionReason);

    if (result) {
      handleDecisionSuccess();
    }
  }, [
    decisionReason,
    handleDecisionSuccess,
    keep,
  ]);

  const handleRemove = useCallback(async () => {
    setDecisionAttempted(true);

    const result = await remove(decisionReason);

    if (result) {
      handleDecisionSuccess();
    }
  }, [
    decisionReason,
    handleDecisionSuccess,
    remove,
  ]);

  return {
    decisionReason,
    decisionModalOpen,
    decisionAttempted,
    setDecisionReason,
    openDecisionModal,
    closeDecisionModal,
    handleKeep,
    handleRemove,
  };
}