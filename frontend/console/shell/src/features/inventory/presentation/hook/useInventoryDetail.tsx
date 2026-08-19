// frontend/console/shell/src/features/inventory/presentation/hook/useInventoryDetail.tsx

import * as React from "react";

import type {
  InventoryDetailRowDTO,
  InventoryDetailViewModel,
  InventoryShippingAddressDTO,
} from "../../../../shared/types/inventory";

import {
  loadInventoryDetailViewModel,
  saveInventoryShippingAddress,
} from "../../application/inventoryDetailService";

import { fetchListsByInventoryIdHTTP } from "../../../list/infrastructure/repository";

export type InventoryListItem = {
  id: string;
  readableId: string;
};

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryDetailRowDTO[];
  loading: boolean;
  error: string | null;

  selectedShippingAddressId: string;
  shippingAddressOptions: InventoryShippingAddressDTO[];
  shippingAddressSaving: boolean;
  shippingAddressError: string | null;

  listItems: InventoryListItem[];
  listLoading: boolean;
  listError: string | null;

  handleSelectShippingAddress: (shippingAddressId: string) => void;
  handleSaveShippingAddress: () => Promise<void>;
};

export function useInventoryDetail(
  inventoryId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [selectedShippingAddressId, setSelectedShippingAddressId] = React.useState("");
  const [shippingAddressSaving, setShippingAddressSaving] = React.useState(false);
  const [shippingAddressError, setShippingAddressError] = React.useState<string | null>(null);

  const [listItems, setListItems] = React.useState<InventoryListItem[]>([]);
  const [listLoading, setListLoading] = React.useState(false);
  const [listError, setListError] = React.useState<string | null>(null);

  const invId = React.useMemo(() => inventoryId ?? "", [inventoryId]);

  React.useEffect(() => {
    if (!invId) {
      setVm(null);
      setError(null);
      setLoading(false);
      setSelectedShippingAddressId("");
      setShippingAddressSaving(false);
      setShippingAddressError(null);
      return;
    }

    let cancelled = false;

    async function load() {
      try {
        setLoading(true);
        setError(null);
        setShippingAddressError(null);

        const nextVm = await loadInventoryDetailViewModel(invId);
        if (cancelled) {
          return;
        }

        setVm(nextVm);
        setSelectedShippingAddressId(nextVm.shippingAddressId);
      } catch (error) {
        if (cancelled) {
          return;
        }

        setError(
          error instanceof Error
            ? error.message
            : String(error),
        );
        setVm(null);
        setSelectedShippingAddressId("");
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [invId]);

  React.useEffect(() => {
    if (!invId) {
      setListItems([]);
      setListLoading(false);
      setListError(null);
      return;
    }

    let cancelled = false;

    async function loadLists() {
      try {
        setListLoading(true);
        setListError(null);

        const items = await fetchListsByInventoryIdHTTP(invId);
        if (cancelled) {
          return;
        }

        setListItems(
          items
            .filter((item) => item.inventoryId === invId)
            .map((item) => ({
              id: item.id,
              readableId: item.readableId,
            })),
        );
      } catch (error) {
        if (cancelled) {
          return;
        }

        setListItems([]);
        setListError(
          error instanceof Error
            ? error.message
            : String(error),
        );
      } finally {
        if (!cancelled) {
          setListLoading(false);
        }
      }
    }

    void loadLists();

    return () => {
      cancelled = true;
    };
  }, [invId]);

  const rows = React.useMemo<InventoryDetailRowDTO[]>(
    () => vm?.rows ?? [],
    [vm],
  );

  const shippingAddressOptions = React.useMemo<InventoryShippingAddressDTO[]>(
    () => vm?.shippingAddressOptions ?? [],
    [vm],
  );

  const handleSelectShippingAddress = React.useCallback(
    (shippingAddressId: string) => {
      if (!shippingAddressId) {
        return;
      }

      setSelectedShippingAddressId(shippingAddressId);
      setShippingAddressError(null);
    },
    [],
  );

  const handleSaveShippingAddress = React.useCallback(
    async () => {
      if (!invId) {
        setShippingAddressError("inventoryId is empty");
        return;
      }

      if (!selectedShippingAddressId) {
        setShippingAddressError("在庫保管場所を選択してください。");
        return;
      }

      if (vm?.shippingAddressId === selectedShippingAddressId) {
        setShippingAddressError(null);
        return;
      }

      try {
        setShippingAddressSaving(true);
        setShippingAddressError(null);

        const nextVm = await saveInventoryShippingAddress(
          invId,
          selectedShippingAddressId,
        );

        setVm(nextVm);
        setSelectedShippingAddressId(nextVm.shippingAddressId);
      } catch (error) {
        setShippingAddressError(
          error instanceof Error
            ? error.message
            : String(error),
        );
      } finally {
        setShippingAddressSaving(false);
      }
    },
    [
      invId,
      selectedShippingAddressId,
      vm?.shippingAddressId,
    ],
  );

  return {
    vm,
    rows,
    loading,
    error,

    selectedShippingAddressId,
    shippingAddressOptions,
    shippingAddressSaving,
    shippingAddressError,

    listItems,
    listLoading,
    listError,

    handleSelectShippingAddress,
    handleSaveShippingAddress,
  };
}