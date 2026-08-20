// frontend/console/shell/src/features/company/presentation/hook/useLocationDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import type { CompanyShippingAddressReadModel } from "../../../../shared/types/shippingAddress";
import {
  deleteCompanyShippingAddress,
  fetchCompanyShippingAddress,
  updateCompanyShippingAddress,
} from "../../application/locationManagementService";

type LocationFieldErrors = {
  name: string | null;
  zipCode: string | null;
  state: string | null;
  city: string | null;
  street: string | null;
};

export type UseLocationDetailResult = {
  vm: {
    location: CompanyShippingAddressReadModel | null;
    loading: boolean;
    saving: boolean;
    deleting: boolean;
    exists: boolean;
    isDirty: boolean;
    nameError: string | null;
    zipCodeError: string | null;
    stateError: string | null;
    cityError: string | null;
    streetError: string | null;
    error: string | null;
  };
  handlers: {
    onChangeName: (value: string) => void;
    onChangeZipCode: (value: string) => void;
    onChangeState: (value: string) => void;
    onChangeCity: (value: string) => void;
    onChangeStreet: (value: string) => void;
    onChangeStreet2: (value: string) => void;
    onBack: () => void;
    onReset: () => void;
    onSave: () => Promise<boolean>;
    onDelete: () => Promise<void>;
  };
};

const emptyFieldErrors: LocationFieldErrors = {
  name: null,
  zipCode: null,
  state: null,
  city: null,
  street: null,
};

function cloneLocation(
  location: CompanyShippingAddressReadModel,
): CompanyShippingAddressReadModel {
  return { ...location };
}

function locationEquals(
  left: CompanyShippingAddressReadModel | null,
  right: CompanyShippingAddressReadModel | null,
): boolean {
  if (left === null || right === null) return left === right;

  return (
    left.name === right.name &&
    left.zipCode === right.zipCode &&
    left.state === right.state &&
    left.city === right.city &&
    left.street === right.street &&
    left.street2 === right.street2 &&
    left.country === right.country
  );
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  return "在庫保管場所の処理に失敗しました。";
}

function validateLocation(
  location: CompanyShippingAddressReadModel,
): LocationFieldErrors {
  const errors: LocationFieldErrors = { ...emptyFieldErrors };

  if (!location.name) errors.name = "保管場所名を入力してください。";

  if (!location.zipCode) {
    errors.zipCode = "郵便番号を入力してください。";
  } else if (
    location.country === "JP" &&
    !/^[0-9]{3}-?[0-9]{4}$/.test(location.zipCode)
  ) {
    errors.zipCode = "郵便番号は123-4567または1234567の形式で入力してください。";
  }

  if (!location.state) errors.state = "都道府県を入力してください。";
  if (!location.city) errors.city = "市区町村を入力してください。";
  if (!location.street) errors.street = "住所を入力してください。";

  return errors;
}

function hasFieldError(errors: LocationFieldErrors): boolean {
  return Object.values(errors).some((value) => value !== null);
}

export function useLocationDetail(
  locationId?: string,
): UseLocationDetailResult {
  const navigate = useNavigate();

  const [location, setLocation] =
    useState<CompanyShippingAddressReadModel | null>(null);
  const [originalLocation, setOriginalLocation] =
    useState<CompanyShippingAddressReadModel | null>(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [fieldErrors, setFieldErrors] =
    useState<LocationFieldErrors>({ ...emptyFieldErrors });
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);
    setFieldErrors({ ...emptyFieldErrors });

    try {
      if (!locationId) {
        throw new Error("在庫保管場所IDを取得できませんでした。");
      }

      const loaded = await fetchCompanyShippingAddress(locationId);
      const next = cloneLocation(loaded);

      setLocation(next);
      setOriginalLocation(cloneLocation(next));
    } catch (loadError: unknown) {
      setLocation(null);
      setOriginalLocation(null);
      setError(getErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [locationId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const exists = location !== null;

  const isDirty = useMemo(
    () => !locationEquals(location, originalLocation),
    [location, originalLocation],
  );

  const clearFieldError = useCallback(
    (field: keyof LocationFieldErrors) => {
      setFieldErrors((current) => ({
        ...current,
        [field]: null,
      }));
    },
    [],
  );

  const onChangeName = useCallback(
    (value: string) => {
      setLocation((current) =>
        current
          ? {
              ...current,
              name: value,
            }
          : current,
      );

      clearFieldError("name");
      setError(null);
    },
    [clearFieldError],
  );

  const onChangeZipCode = useCallback(
    (value: string) => {
      setLocation((current) =>
        current
          ? {
              ...current,
              zipCode: value,
            }
          : current,
      );

      clearFieldError("zipCode");
      setError(null);
    },
    [clearFieldError],
  );

  const onChangeState = useCallback(
    (value: string) => {
      setLocation((current) =>
        current
          ? {
              ...current,
              state: value,
            }
          : current,
      );

      clearFieldError("state");
      setError(null);
    },
    [clearFieldError],
  );

  const onChangeCity = useCallback(
    (value: string) => {
      setLocation((current) =>
        current
          ? {
              ...current,
              city: value,
            }
          : current,
      );

      clearFieldError("city");
      setError(null);
    },
    [clearFieldError],
  );

  const onChangeStreet = useCallback(
    (value: string) => {
      setLocation((current) =>
        current
          ? {
              ...current,
              street: value,
            }
          : current,
      );

      clearFieldError("street");
      setError(null);
    },
    [clearFieldError],
  );

  const onChangeStreet2 = useCallback((value: string) => {
    setLocation((current) =>
      current
        ? {
            ...current,
            street2: value,
          }
        : current,
    );

    setError(null);
  }, []);

  const onBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const onReset = useCallback(() => {
    if (!originalLocation) return;

    setLocation(cloneLocation(originalLocation));
    setFieldErrors({ ...emptyFieldErrors });
    setError(null);
  }, [originalLocation]);

  const onSave = useCallback(async (): Promise<boolean> => {
    if (!location || saving || deleting) return false;

    const targetLocationId = location.id || locationId;

    if (!targetLocationId) {
      setError("在庫保管場所IDを取得できませんでした。");
      return false;
    }

    const nextFieldErrors = validateLocation(location);
    setFieldErrors(nextFieldErrors);

    if (hasFieldError(nextFieldErrors)) {
      setError("入力内容を確認してください。");
      return false;
    }

    setSaving(true);
    setError(null);

    try {
      const saved = await updateCompanyShippingAddress(
        targetLocationId,
        {
          name: location.name,
          zipCode: location.zipCode,
          state: location.state,
          city: location.city,
          street: location.street,
          street2: location.street2,
          country: location.country || "JP",
        },
      );

      const next = cloneLocation(saved);

      setLocation(next);
      setOriginalLocation(cloneLocation(next));
      setFieldErrors({ ...emptyFieldErrors });

      return true;
    } catch (saveError: unknown) {
      setError(getErrorMessage(saveError));
      return false;
    } finally {
      setSaving(false);
    }
  }, [location, locationId, saving, deleting]);

  const onDelete = useCallback(async (): Promise<void> => {
    if (!location || saving || deleting) return;

    const targetLocationId = location.id || locationId;

    if (!targetLocationId) {
      setError("在庫保管場所IDを取得できませんでした。");
      return;
    }

    const confirmed = window.confirm(
      `「${location.name}」を削除しますか？`,
    );

    if (!confirmed) return;

    setDeleting(true);
    setError(null);

    try {
      await deleteCompanyShippingAddress(targetLocationId);

      navigate("/stockLocation", {
        replace: true,
      });
    } catch (deleteError: unknown) {
      setError(getErrorMessage(deleteError));
    } finally {
      setDeleting(false);
    }
  }, [location, locationId, saving, deleting, navigate]);

  return {
    vm: {
      location,
      loading,
      saving,
      deleting,
      exists,
      isDirty,
      nameError: fieldErrors.name,
      zipCodeError: fieldErrors.zipCode,
      stateError: fieldErrors.state,
      cityError: fieldErrors.city,
      streetError: fieldErrors.street,
      error,
    },
    handlers: {
      onChangeName,
      onChangeZipCode,
      onChangeState,
      onChangeCity,
      onChangeStreet,
      onChangeStreet2,
      onBack,
      onReset,
      onSave,
      onDelete,
    },
  };
}

export default useLocationDetail;