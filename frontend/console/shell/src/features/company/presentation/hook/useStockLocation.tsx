// frontend/console/shell/src/features/company/presentation/hook/useStockLocation.tsx

import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import type { ShippingAddress } from "../../../../shared/types/shippingAddress";
import type { ShippingAddressFormValue } from "../components/ShippingAddressFormModal";

import {
  createCompanyShippingAddress,
  deleteCompanyShippingAddress,
  listCompanyShippingAddresses,
  updateCompanyShippingAddress,
} from "../../application/companyDetailService";

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return String(error);
}

export function useStockLocation() {
  const navigate = useNavigate();

  const [shippingAddresses, setShippingAddresses] = useState<ShippingAddress[]>([]);
  const [shippingAddressFormOpen, setShippingAddressFormOpen] = useState(false);
  const [editingShippingAddress, setEditingShippingAddress] = useState<ShippingAddress | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadShippingAddresses = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await listCompanyShippingAddresses();
      setShippingAddresses(response);
    } catch (loadError: unknown) {
      setError(getErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadShippingAddresses();
  }, [loadShippingAddresses]);

  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleOpenCreateShippingAddress = useCallback(() => {
    if (saving) {
      return;
    }

    setError(null);
    setEditingShippingAddress(null);
    setShippingAddressFormOpen(true);
  }, [saving]);

  const handleOpenEditShippingAddress = useCallback(
    (shippingAddressId: string) => {
      if (saving) {
        return;
      }

      const target = shippingAddresses.find((address) => address.id === shippingAddressId);
      if (!target) {
        setError("編集対象の在庫保管場所が見つかりません。");
        return;
      }

      setError(null);
      setEditingShippingAddress(target);
      setShippingAddressFormOpen(true);
    },
    [shippingAddresses, saving],
  );

  const handleCloseShippingAddressForm = useCallback(() => {
    if (saving) {
      return;
    }

    setShippingAddressFormOpen(false);
    setEditingShippingAddress(null);
  }, [saving]);

  const handleSaveShippingAddress = useCallback(
    async (value: ShippingAddressFormValue) => {
      if (saving) {
        return;
      }

      if (!value.name || !value.zipCode || !value.state || !value.city || !value.street) {
        setError("保管場所名、郵便番号、都道府県、市区町村、住所は必須です。");
        return;
      }

      try {
        setSaving(true);
        setError(null);

        if (editingShippingAddress) {
          const updated = await updateCompanyShippingAddress(
            editingShippingAddress.id,
            value,
          );

          setShippingAddresses((current) =>
            current.map((address) =>
              address.id === updated.id ? updated : address,
            ),
          );
        } else {
          const created = await createCompanyShippingAddress(value);
          setShippingAddresses((current) => [created, ...current]);
        }

        setShippingAddressFormOpen(false);
        setEditingShippingAddress(null);
      } catch (saveError: unknown) {
        setError(getErrorMessage(saveError));
      } finally {
        setSaving(false);
      }
    },
    [editingShippingAddress, saving],
  );

  const handleDeleteShippingAddress = useCallback(
    async (shippingAddressId: string) => {
      if (!shippingAddressId || saving) {
        return;
      }

      const target = shippingAddresses.find((address) => address.id === shippingAddressId);
      if (!target) {
        setError("削除対象の在庫保管場所が見つかりません。");
        return;
      }

      const confirmed = window.confirm(
        `「${target.name}」を削除しますか？`,
      );
      if (!confirmed) {
        return;
      }

      try {
        setSaving(true);
        setError(null);

        await deleteCompanyShippingAddress(shippingAddressId);

        setShippingAddresses((current) =>
          current.filter((address) => address.id !== shippingAddressId),
        );

        if (editingShippingAddress?.id === shippingAddressId) {
          setEditingShippingAddress(null);
          setShippingAddressFormOpen(false);
        }
      } catch (deleteError: unknown) {
        setError(getErrorMessage(deleteError));
      } finally {
        setSaving(false);
      }
    },
    [shippingAddresses, editingShippingAddress, saving],
  );

  return {
    shippingAddresses,
    shippingAddressFormOpen,
    editingShippingAddress,
    loading,
    saving,
    error,
    reload: loadShippingAddresses,
    handleBack,
    handleOpenCreateShippingAddress,
    handleOpenEditShippingAddress,
    handleCloseShippingAddressForm,
    handleSaveShippingAddress,
    handleDeleteShippingAddress,
  };
}

export default useStockLocation;