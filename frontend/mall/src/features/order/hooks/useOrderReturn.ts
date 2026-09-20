// frontend/mall/src/features/order/hooks/useOrderReturn.ts

import { useCallback, useState } from "react";

import type { ReturnPackageState } from "../components/ReturnRequestModal";

type ReturnItem = (
  itemIndex: number,
  packageState: ReturnPackageState,
  reason: string,
) => boolean | Promise<boolean>;

type UseOrderReturnParams = {
  returningItemIndex: number | null;
  returnItem: ReturnItem;
};

export function useOrderReturn({
  returningItemIndex,
  returnItem,
}: UseOrderReturnParams) {
  const [returnTargetIndex, setReturnTargetIndex] = useState<number | null>(null);
  const [packageState, setPackageState] = useState<ReturnPackageState | null>(null);
  const [reason, setReason] = useState("");

  const resetReturnForm = useCallback(() => {
    setReturnTargetIndex(null);
    setPackageState(null);
    setReason("");
  }, []);

  const openReturnModal = useCallback((itemIndex: number) => {
    if (!Number.isInteger(itemIndex) || itemIndex < 0) {
      return;
    }

    setReturnTargetIndex(itemIndex);
    setPackageState(null);
    setReason("");
  }, []);

  const closeReturnModal = useCallback(() => {
    if (returningItemIndex !== null) {
      return;
    }

    resetReturnForm();
  }, [resetReturnForm, returningItemIndex]);

  const changePackageState = useCallback((value: ReturnPackageState | null) => {
    setPackageState(value);
    setReason("");
  }, []);

  const submitReturn = useCallback(async () => {
    if (returnTargetIndex === null || packageState === null) {
      return false;
    }

    const normalizedReason = reason.trim();

    if (!normalizedReason) {
      return false;
    }

    const succeeded = await returnItem(
      returnTargetIndex,
      packageState,
      normalizedReason,
    );

    if (!succeeded) {
      return false;
    }

    resetReturnForm();
    return true;
  }, [
    packageState,
    reason,
    resetReturnForm,
    returnItem,
    returnTargetIndex,
  ]);

  return {
    returnTargetIndex,
    packageState,
    reason,
    setReason,
    setPackageState: changePackageState,
    openReturnModal,
    closeReturnModal,
    submitReturn,
  };
}