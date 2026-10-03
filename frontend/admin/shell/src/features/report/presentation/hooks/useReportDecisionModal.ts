// frontend/admin/shell/src/features/report/presentation/hooks/useReportDecisionModal.ts

import { useCallback, useState } from "react";

type KeepDecisionAction = (
  decisionReason: string,
  continueTrade?: boolean,
) => Promise<unknown | null>;

type RemoveDecisionAction = (
  decisionReason: string,
) => Promise<unknown | null>;

type UseReportDecisionModalParams = {
  canDecide: boolean;
  deciding: boolean;
  keep: KeepDecisionAction;
  remove: RemoveDecisionAction;
};

export function useReportDecisionModal({
  canDecide,
  deciding,
  keep,
  remove,
}: UseReportDecisionModalParams) {
  const [decisionReason, setDecisionReason] = useState("");
  const [continueTrade, setContinueTrade] = useState(false);
  const [decisionModalOpen, setDecisionModalOpen] = useState(false);
  const [decisionAttempted, setDecisionAttempted] = useState(false);

  const resetDecisionState = useCallback(() => {
    setDecisionReason("");
    setContinueTrade(false);
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

    const result = await keep(
      decisionReason,
      continueTrade,
    );

    if (result) {
      handleDecisionSuccess();
    }
  }, [
    continueTrade,
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
    continueTrade,
    decisionModalOpen,
    decisionAttempted,
    setDecisionReason,
    setContinueTrade,
    openDecisionModal,
    closeDecisionModal,
    handleKeep,
    handleRemove,
  };
}