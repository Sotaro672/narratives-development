// frontend/console/shell/src/features/transportation/presentation/hook/useTransportationFeeDetail.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import type {
  IslandCode,
  PrefectureCode,
  TransportationRegion,
} from "../../../../shared/types/transporation";

import {
  deleteTransportation,
  fetchTransportationVM,
  updateTransportation,
  type TransportationIslandRateVM,
  type TransportationRegionVM,
  type TransportationVM,
} from "../../application/transportationService";

export type TransportationDetailAmountInput = string | number | null;
export type TransportationDetailIslandAmountInput = string | number | null;

export type UseTransportationFeeDetailResult = {
  vm: {
    transportation: TransportationVM | null;
    regions: TransportationRegionVM[];
    islandRates: TransportationIslandRateVM[];
    loading: boolean;
    saving: boolean;
    deleting: boolean;
    exists: boolean;
    isDirty: boolean;
    error: string | null;
    successMessage: string | null;
  };
  handlers: {
    onBack: () => void;
    onReset: () => void;
    onSave: () => Promise<void>;
    onDelete: () => Promise<void>;
    onDismissError: () => void;
    onChangeName: (name: string) => void;
    onChangePrefectureAmount: (
      prefectureCode: PrefectureCode,
      amount: TransportationDetailAmountInput,
    ) => void;
    onChangeRegionAmount: (
      region: TransportationRegion,
      amount: TransportationDetailAmountInput,
    ) => void;
    onChangeIslandRateAmount: (
      islandCode: IslandCode,
      amount: TransportationDetailIslandAmountInput,
    ) => void;
  };
};

function toAmount(
  value: TransportationDetailAmountInput,
): number | null {
  if (value === null || value === "") {
    return null;
  }

  if (typeof value === "number") {
    return Number.isFinite(value)
      ? value
      : null;
  }

  const parsed = Number(value);

  return Number.isFinite(parsed)
    ? parsed
    : null;
}

function toIslandAmount(
  value: TransportationDetailIslandAmountInput,
): number | null {
  if (value === null || value === "") {
    return null;
  }

  if (typeof value === "number") {
    return Number.isFinite(value)
      ? value
      : null;
  }

  const parsed = Number(value);

  return Number.isFinite(parsed)
    ? parsed
    : null;
}

function cloneTransportationVM(
  value: TransportationVM,
): TransportationVM {
  return {
    ...value,
    regions: value.regions.map((region) => ({
      ...region,
      prefectures: region.prefectures.map(
        (prefecture) => ({
          ...prefecture,
        }),
      ),
    })),
    islandRates: value.islandRates.map(
      (islandRate) => ({
        ...islandRate,
      }),
    ),
  };
}

function transportationEquals(
  left: TransportationVM | null,
  right: TransportationVM | null,
): boolean {
  if (
    left === null ||
    right === null
  ) {
    return left === right;
  }

  return JSON.stringify({
    name: left.name,
    regions: left.regions,
    islandRates: left.islandRates,
  }) === JSON.stringify({
    name: right.name,
    regions: right.regions,
    islandRates: right.islandRates,
  });
}

function errorMessage(
  error: unknown,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return "配送料金設定の処理に失敗しました。";
}

export function useTransportationFeeDetail(
  transportationId?: string,
): UseTransportationFeeDetailResult {
  const navigate = useNavigate();

  const [
    transportation,
    setTransportation,
  ] = useState<TransportationVM | null>(
    null,
  );

  const [
    originalTransportation,
    setOriginalTransportation,
  ] = useState<TransportationVM | null>(
    null,
  );

  const [
    loading,
    setLoading,
  ] = useState(false);

  const [
    saving,
    setSaving,
  ] = useState(false);

  const [
    deleting,
    setDeleting,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<string | null>(
    null,
  );

  const [
    successMessage,
    setSuccessMessage,
  ] = useState<string | null>(
    null,
  );

  const reload = useCallback(
    async () => {
      setLoading(true);
      setError(null);
      setSuccessMessage(null);

      try {
        if (!transportationId) {
          throw new Error(
            "配送料金設定IDを取得できませんでした。",
          );
        }

        const loaded =
          await fetchTransportationVM(
            transportationId,
          );

        const next =
          cloneTransportationVM(
            loaded,
          );

        setTransportation(next);

        setOriginalTransportation(
          cloneTransportationVM(
            next,
          ),
        );
      } catch (
        loadError: unknown
      ) {
        setTransportation(null);
        setOriginalTransportation(null);
        setError(
          errorMessage(
            loadError,
          ),
        );
      } finally {
        setLoading(false);
      }
    },
    [
      transportationId,
    ],
  );

  useEffect(() => {
    void reload();
  }, [
    reload,
  ]);

  const regions =
    transportation?.regions ?? [];

  const islandRates =
    transportation?.islandRates ?? [];

  const exists =
    transportation !== null;

  const isDirty = useMemo(
    () =>
      !transportationEquals(
        transportation,
        originalTransportation,
      ),
    [
      transportation,
      originalTransportation,
    ],
  );

  const onBack =
    useCallback(() => {
      navigate(-1);
    }, [
      navigate,
    ]);

  const onDismissError =
    useCallback(() => {
      setError(null);
    }, []);

  const onReset =
    useCallback(() => {
      if (
        !originalTransportation
      ) {
        return;
      }

      setTransportation(
        cloneTransportationVM(
          originalTransportation,
        ),
      );

      setError(null);
      setSuccessMessage(null);
    }, [
      originalTransportation,
    ]);

  const onChangeName =
    useCallback(
      (
        name: string,
      ) => {
        setTransportation(
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              name,
            };
          },
        );

        setError(null);
        setSuccessMessage(null);
      },
      [],
    );

  const onChangePrefectureAmount =
    useCallback(
      (
        prefectureCode: PrefectureCode,
        amountInput: TransportationDetailAmountInput,
      ) => {
        const amount =
          toAmount(
            amountInput,
          );

        setTransportation(
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              regions:
                current.regions.map(
                  (region) => ({
                    ...region,
                    prefectures:
                      region.prefectures.map(
                        (
                          prefecture,
                        ) =>
                          prefecture.prefectureCode ===
                          prefectureCode
                            ? {
                                ...prefecture,
                                amount,
                              }
                            : prefecture,
                      ),
                  }),
                ),
            };
          },
        );

        setError(null);
        setSuccessMessage(null);
      },
      [],
    );

  const onChangeRegionAmount =
    useCallback(
      (
        targetRegion: TransportationRegion,
        amountInput: TransportationDetailAmountInput,
      ) => {
        const amount =
          toAmount(
            amountInput,
          );

        setTransportation(
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              regions:
                current.regions.map(
                  (region) => {
                    if (
                      region.region !==
                      targetRegion
                    ) {
                      return region;
                    }

                    return {
                      ...region,
                      prefectures:
                        region.prefectures.map(
                          (
                            prefecture,
                          ) => ({
                            ...prefecture,
                            amount,
                          }),
                        ),
                    };
                  },
                ),
            };
          },
        );

        setError(null);
        setSuccessMessage(null);
      },
      [],
    );

  const onChangeIslandRateAmount =
    useCallback(
      (
        islandCode: IslandCode,
        amountInput: TransportationDetailIslandAmountInput,
      ) => {
        const amount =
          toIslandAmount(
            amountInput,
          );

        setTransportation(
          (current) => {
            if (!current) {
              return current;
            }

            return {
              ...current,
              islandRates:
                current.islandRates.map(
                  (rate) =>
                    rate.islandCode ===
                    islandCode
                      ? {
                          ...rate,
                          amount,
                        }
                      : rate,
                ),
            };
          },
        );

        setError(null);
        setSuccessMessage(null);
      },
      [],
    );

  const onSave =
    useCallback(
      async () => {
        if (
          !transportation ||
          saving ||
          deleting
        ) {
          return;
        }

        const targetTransportationId =
          transportation.id ||
          transportationId;

        if (
          !targetTransportationId
        ) {
          setError(
            "配送料金設定IDを取得できませんでした。",
          );
          return;
        }

        setSaving(true);
        setError(null);
        setSuccessMessage(null);

        try {
          const saved =
            await updateTransportation(
              targetTransportationId,
              {
                name:
                  transportation.name,
                regions:
                  transportation.regions,
                islandRates:
                  transportation.islandRates,
              },
            );

          const next =
            cloneTransportationVM(
              saved,
            );

          setTransportation(next);

          setOriginalTransportation(
            cloneTransportationVM(
              next,
            ),
          );

          setSuccessMessage(
            "配送料金設定を更新しました。",
          );
        } catch (
          saveError: unknown
        ) {
          setError(
            errorMessage(
              saveError,
            ),
          );
        } finally {
          setSaving(false);
        }
      },
      [
        transportation,
        saving,
        deleting,
        transportationId,
      ],
    );

  const onDelete =
    useCallback(
      async () => {
        if (
          !transportation ||
          deleting ||
          saving
        ) {
          return;
        }

        const targetTransportationId =
          transportation.id ||
          transportationId;

        if (
          !targetTransportationId
        ) {
          setError(
            "配送料金設定IDを取得できませんでした。",
          );
          return;
        }

        setDeleting(true);
        setError(null);
        setSuccessMessage(null);

        try {
          await deleteTransportation(
            targetTransportationId,
          );

          navigate(
            "/transportationFee",
            {
              replace: true,
            },
          );
        } catch (
          deleteError: unknown
        ) {
          setError(
            errorMessage(
              deleteError,
            ),
          );
        } finally {
          setDeleting(false);
        }
      },
      [
        transportation,
        deleting,
        saving,
        transportationId,
        navigate,
      ],
    );

  return {
    vm: {
      transportation,
      regions,
      islandRates,
      loading,
      saving,
      deleting,
      exists,
      isDirty,
      error,
      successMessage,
    },
    handlers: {
      onBack,
      onReset,
      onSave,
      onDelete,
      onDismissError,
      onChangeName,
      onChangePrefectureAmount,
      onChangeRegionAmount,
      onChangeIslandRateAmount,
    },
  };
}