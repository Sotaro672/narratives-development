// frontend/console/shell/src/features/company/presentation/hook/useCompanyDetail.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

import type {
  Company,
} from "../../../../shared/types/company";

import type {
  ShippingAddress,
} from "../../../../shared/types/shippingAddress";

import {
  safeDateTimeLabelJa,
} from "../../../../shared/util/dateJa";

import {
  useAssigneeSelection,
} from "../../../admin/presentation/hook/useAssigneeSelection";

import {
  createCompanyShippingAddress,
  deleteCompanyShippingAddress,
  fetchCompanyDetail,
  listCompanyShippingAddresses,
  updateCompanyDetail,
  updateCompanyShippingAddress,
} from "../../application/companyDetailService";

export type CompanyShippingAddressFormValue = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

function getErrorMessage(
  error: unknown,
): string {
  if (error instanceof Error) {
    return error.message;
  }

  return String(error);
}

function toShippingAddressFormValue(
  value: CompanyShippingAddressFormValue,
): CompanyShippingAddressFormValue {
  return {
    zipCode: value.zipCode,
    state: value.state,
    city: value.city,
    street: value.street,
    street2: value.street2,
    country: value.country,
  };
}

export function useCompanyDetail() {
  const navigate = useNavigate();

  const {
    currentMember,
  } = useAuthContext();

  const companyId =
    currentMember?.companyId ?? "";

  const [
    company,
    setCompany,
  ] = useState<Company | null>(
    null,
  );

  const [
    companyName,
    setCompanyName,
  ] = useState("");

  const [
    shippingAddresses,
    setShippingAddresses,
  ] = useState<ShippingAddress[]>(
    [],
  );

  const [
    shippingAddressFormOpen,
    setShippingAddressFormOpen,
  ] = useState(false);

  const [
    editingShippingAddress,
    setEditingShippingAddress,
  ] = useState<ShippingAddress | null>(
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
    shippingAddressSaving,
    setShippingAddressSaving,
  ] = useState(false);

  const [
    error,
    setError,
  ] = useState<string | null>(
    null,
  );

  const {
    assigneeId: adminMemberId,
    assigneeName: adminMemberName,
    assigneeCandidates: adminCandidates,
    loadingMembers,
    handleSelectAssignee: handleSelectAdmin,
  } = useAssigneeSelection({
    initialAssigneeId:
      company?.admin ?? null,
    defaultToCurrentMember: false,
  });

  const loadCompany = useCallback(
    async () => {
      if (!companyId) {
        setCompany(null);
        setCompanyName("");
        return;
      }

      const response =
        await fetchCompanyDetail(
          companyId,
        );

      setCompany(
        response,
      );

      setCompanyName(
        response.name,
      );
    },
    [
      companyId,
    ],
  );

  const loadShippingAddresses =
    useCallback(
      async () => {
        if (!companyId) {
          setShippingAddresses(
            [],
          );
          return;
        }

        const response =
          await listCompanyShippingAddresses();

        setShippingAddresses(
          response,
        );
      },
      [
        companyId,
      ],
    );

  const reload =
    useCallback(
      async () => {
        if (!companyId) {
          setError(
            "companyId が取得できません。",
          );
          return;
        }

        try {
          setLoading(
            true,
          );

          setError(
            null,
          );

          await Promise.all([
            loadCompany(),
            loadShippingAddresses(),
          ]);
        } catch (
          loadError: unknown
        ) {
          setError(
            getErrorMessage(
              loadError,
            ),
          );
        } finally {
          setLoading(
            false,
          );
        }
      },
      [
        companyId,
        loadCompany,
        loadShippingAddresses,
      ],
    );

  useEffect(
    () => {
      if (!companyId) {
        return;
      }

      void reload();
    },
    [
      companyId,
      reload,
    ],
  );

  const createdByName =
    useMemo(
      () => {
        const createdBy =
          company?.createdBy ?? "";

        if (!createdBy) {
          return null;
        }

        const matched =
          adminCandidates.find(
            (candidate) =>
              candidate.id ===
              createdBy,
          );

        return (
          matched?.name ??
          createdBy
        );
      },
      [
        company?.createdBy,
        adminCandidates,
      ],
    );

  const updatedByName =
    useMemo(
      () => {
        const updatedBy =
          company?.updatedBy ?? "";

        if (!updatedBy) {
          return null;
        }

        const matched =
          adminCandidates.find(
            (candidate) =>
              candidate.id ===
              updatedBy,
          );

        return (
          matched?.name ??
          updatedBy
        );
      },
      [
        company?.updatedBy,
        adminCandidates,
      ],
    );

  const createdAt =
    useMemo(
      () => {
        if (!company?.createdAt) {
          return null;
        }

        return safeDateTimeLabelJa(
          company.createdAt,
          "",
        );
      },
      [
        company?.createdAt,
      ],
    );

  const updatedAt =
    useMemo(
      () => {
        if (!company?.updatedAt) {
          return null;
        }

        return safeDateTimeLabelJa(
          company.updatedAt,
          "",
        );
      },
      [
        company?.updatedAt,
      ],
    );

  const handleBack =
    useCallback(
      () => {
        navigate(
          -1,
        );
      },
      [
        navigate,
      ],
    );

  const handleSave =
    useCallback(
      async () => {
        if (
          !companyId ||
          !company ||
          saving
        ) {
          return;
        }

        if (!companyName) {
          setError(
            "会社名は必須です。",
          );
          return;
        }

        if (!adminMemberId) {
          setError(
            "管理者を選択してください。",
          );
          return;
        }

        try {
          setSaving(
            true,
          );

          setError(
            null,
          );

          const updated =
            await updateCompanyDetail(
              companyId,
              {
                name:
                  companyName,
                admin:
                  adminMemberId,
              },
            );

          setCompany(
            updated,
          );

          setCompanyName(
            updated.name,
          );
        } catch (
          saveError: unknown
        ) {
          setError(
            getErrorMessage(
              saveError,
            ),
          );
        } finally {
          setSaving(
            false,
          );
        }
      },
      [
        companyId,
        company,
        companyName,
        adminMemberId,
        saving,
      ],
    );

  const handleOpenCreateShippingAddress =
    useCallback(
      () => {
        if (
          shippingAddressSaving
        ) {
          return;
        }

        setError(
          null,
        );

        setEditingShippingAddress(
          null,
        );

        setShippingAddressFormOpen(
          true,
        );
      },
      [
        shippingAddressSaving,
      ],
    );

  const handleOpenEditShippingAddress =
    useCallback(
      (
        shippingAddressId: string,
      ) => {
        if (
          shippingAddressSaving
        ) {
          return;
        }

        const target =
          shippingAddresses.find(
            (address) =>
              address.id ===
              shippingAddressId,
          );

        if (!target) {
          setError(
            "編集対象の住所が見つかりません。",
          );
          return;
        }

        setError(
          null,
        );

        setEditingShippingAddress(
          target,
        );

        setShippingAddressFormOpen(
          true,
        );
      },
      [
        shippingAddresses,
        shippingAddressSaving,
      ],
    );

  const handleCloseShippingAddressForm =
    useCallback(
      () => {
        if (
          shippingAddressSaving
        ) {
          return;
        }

        setShippingAddressFormOpen(
          false,
        );

        setEditingShippingAddress(
          null,
        );
      },
      [
        shippingAddressSaving,
      ],
    );

  const handleSaveShippingAddress =
    useCallback(
      async (
        formValue:
          CompanyShippingAddressFormValue,
      ) => {
        if (
          shippingAddressSaving
        ) {
          return;
        }

        const input =
          toShippingAddressFormValue(
            formValue,
          );

        if (
          !input.zipCode ||
          !input.state ||
          !input.city ||
          !input.street
        ) {
          setError(
            "郵便番号、都道府県、市区町村、住所は必須です。",
          );
          return;
        }

        try {
          setShippingAddressSaving(
            true,
          );

          setError(
            null,
          );

          if (
            editingShippingAddress
          ) {
            const updated =
              await updateCompanyShippingAddress(
                editingShippingAddress.id,
                input,
              );

            setShippingAddresses(
              (
                current,
              ) =>
                current.map(
                  (
                    address,
                  ) =>
                    address.id ===
                    updated.id
                      ? updated
                      : address,
                ),
            );
          } else {
            const created =
              await createCompanyShippingAddress(
                input,
              );

            setShippingAddresses(
              (
                current,
              ) => [
                created,
                ...current,
              ],
            );
          }

          setShippingAddressFormOpen(
            false,
          );

          setEditingShippingAddress(
            null,
          );
        } catch (
          saveError: unknown
        ) {
          setError(
            getErrorMessage(
              saveError,
            ),
          );
        } finally {
          setShippingAddressSaving(
            false,
          );
        }
      },
      [
        editingShippingAddress,
        shippingAddressSaving,
      ],
    );

  const handleDeleteShippingAddress =
    useCallback(
      async (
        shippingAddressId: string,
      ) => {
        if (
          !shippingAddressId ||
          shippingAddressSaving
        ) {
          return;
        }

        const target =
          shippingAddresses.find(
            (address) =>
              address.id ===
              shippingAddressId,
          );

        if (!target) {
          setError(
            "削除対象の住所が見つかりません。",
          );
          return;
        }

        const confirmed =
          window.confirm(
            "この在庫保管場所を削除しますか？",
          );

        if (!confirmed) {
          return;
        }

        try {
          setShippingAddressSaving(
            true,
          );

          setError(
            null,
          );

          await deleteCompanyShippingAddress(
            shippingAddressId,
          );

          setShippingAddresses(
            (
              current,
            ) =>
              current.filter(
                (address) =>
                  address.id !==
                  shippingAddressId,
              ),
          );

          if (
            editingShippingAddress?.id ===
            shippingAddressId
          ) {
            setEditingShippingAddress(
              null,
            );

            setShippingAddressFormOpen(
              false,
            );
          }
        } catch (
          deleteError: unknown
        ) {
          setError(
            getErrorMessage(
              deleteError,
            ),
          );
        } finally {
          setShippingAddressSaving(
            false,
          );
        }
      },
      [
        shippingAddresses,
        shippingAddressSaving,
        editingShippingAddress,
      ],
    );

  return {
    company,
    companyId,
    companyName,
    setCompanyName,

    adminMemberId,
    adminMemberName,
    adminCandidates,
    loadingMembers,
    handleSelectAdmin,

    shippingAddresses,
    shippingAddressFormOpen,
    editingShippingAddress,

    loading,
    saving,
    shippingAddressSaving,
    error,

    createdByName,
    createdAt,
    updatedByName,
    updatedAt,

    reload,
    handleBack,
    handleSave,

    handleOpenCreateShippingAddress,
    handleOpenEditShippingAddress,
    handleCloseShippingAddressForm,
    handleSaveShippingAddress,
    handleDeleteShippingAddress,
  };
}

export default useCompanyDetail;