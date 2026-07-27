// frontend/console/shell/src/features/member/presentation/hooks/useMemberCreate.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type FormEvent,
} from "react";

import type { Member } from "../../../../shared/types/member";

import type {
  Permission,
  PermissionCategory,
} from "../../../../shared/types/permission";

import type { Brand } from "../../../../shared/types/brand";

import {
  fetchAllPermissions,
  fetchBrandsForCurrentMember,
  groupPermissionsByCategory,
} from "../../application/memberListService";

import {
  createMember,
  parseCommaSeparated,
} from "../../application/memberCreateService";

import { sendMemberInvitation } from "../../application/invitationService";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  registeredAt: string;
};

export type UseMemberCreateOptions = {
  /**
   * 作成成功時に呼び出す。
   * 呼び出し元で画面遷移などを行う。
   */
  onSuccess?: (created: Member) => void;
};

function formatDateYmd(
  iso?: string | null,
): string {
  if (!iso) {
    return "";
  }

  const date = new Date(iso);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  const year = date.getFullYear();
  const month = String(
    date.getMonth() + 1,
  ).padStart(2, "0");
  const day = String(
    date.getDate(),
  ).padStart(2, "0");

  return `${year}/${month}/${day}`;
}

function toBrandRows(
  brands: Brand[],
): BrandRow[] {
  return brands.map((brand) => ({
    id: brand.id,
    name: String(brand.name ?? "").trim(),
    isActive: Boolean(
      brand.isActive ?? true,
    ),
    registeredAt: formatDateYmd(
      brand.createdAt,
    ),
  }));
}

function getErrorMessage(
  error: unknown,
): string {
  return error instanceof Error
    ? error.message
    : String(error);
}

export function useMemberCreate(
  options?: UseMemberCreateOptions,
) {
  const [firstName, setFirstName] =
    useState("");

  const [lastName, setLastName] =
    useState("");

  const [
    firstNameKana,
    setFirstNameKana,
  ] = useState("");

  const [
    lastNameKana,
    setLastNameKana,
  ] = useState("");

  const [email, setEmail] =
    useState("");

  const [category, setCategory] =
    useState<PermissionCategory>("brand");

  const [
    permissionsText,
    setPermissionsText,
  ] = useState("");

  const [
    brandsText,
    setBrandsText,
  ] = useState("");

  const [
    submitting,
    setSubmitting,
  ] = useState(false);

  const [error, setError] =
    useState<string | null>(null);

  const [
    allPermissions,
    setAllPermissions,
  ] = useState<Permission[]>([]);

  useEffect(() => {
    let cancelled = false;

    async function loadPermissions() {
      try {
        const items =
          await fetchAllPermissions();

        if (!cancelled) {
          setAllPermissions(items);
        }
      } catch {
        if (!cancelled) {
          setAllPermissions([]);
        }
      }
    }

    void loadPermissions();

    return () => {
      cancelled = true;
    };
  }, []);

  const permissionsByCategory: Record<
    PermissionCategory,
    Permission[]
  > = useMemo(
    () =>
      groupPermissionsByCategory(
        allPermissions,
      ),
    [allPermissions],
  );

  const permissionCategories =
    useMemo(
      () =>
        (
          Object.keys(
            permissionsByCategory,
          ) as PermissionCategory[]
        ).map((permissionCategory) => ({
          key: permissionCategory,
          count:
            permissionsByCategory[
              permissionCategory
            ]?.length ?? 0,
          permissions:
            permissionsByCategory[
              permissionCategory
            ] ?? [],
        })),
      [permissionsByCategory],
    );

  const permissionCategoryList =
    useMemo(
      () =>
        Object.keys(
          permissionsByCategory,
        ) as PermissionCategory[],
      [permissionsByCategory],
    );

  const [allBrands, setAllBrands] =
    useState<Brand[]>([]);

  const [brandRows, setBrandRows] =
    useState<BrandRow[]>([]);

  useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      try {
        const brands =
          await fetchBrandsForCurrentMember();

        if (!cancelled) {
          setAllBrands(brands);
          setBrandRows(
            toBrandRows(brands),
          );
        }
      } catch {
        if (!cancelled) {
          setAllBrands([]);
          setBrandRows([]);
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, []);

  /**
   * Memberを作成し、招待メールを送信する。
   *
   * overridesでpermissionsとassignedBrandIdsを
   * 画面側から上書きできる。
   */
  const handleSubmit = useCallback(
    async (
      event?: FormEvent,
      overrides?: {
        permissions?: string[];
        assignedBrandIds?: string[];
      },
    ) => {
      event?.preventDefault();

      setError(null);
      setSubmitting(true);

      try {
        const permissionsFromCategory =
          permissionsByCategory[
            category
          ]?.map(
            (permission) =>
              permission.name,
          ) ?? [];

        const permissionsFromText =
          permissionsText
            ? parseCommaSeparated(
                permissionsText,
              )
            : [];

        const mergedPermissions =
          Array.from(
            new Set([
              ...permissionsFromCategory,
              ...permissionsFromText,
            ]),
          );

        const finalPermissions =
          overrides?.permissions &&
          overrides.permissions.length > 0
            ? overrides.permissions
            : mergedPermissions;

        const fallbackBrandIds =
          brandsText
            ? parseCommaSeparated(
                brandsText,
              )
            : [];

        const finalAssignedBrandIds =
          overrides?.assignedBrandIds &&
          overrides.assignedBrandIds
            .length > 0
            ? overrides.assignedBrandIds
            : fallbackBrandIds;

        const created =
          await createMember({
            firstName,
            lastName,
            firstNameKana,
            lastNameKana,
            email,
            permissions:
              finalPermissions,
            assignedBrandIds:
              finalAssignedBrandIds,
          });

        try {
          await sendMemberInvitation(
            created.id,
            created.email,
          );
        } catch {
          // Member作成自体は成功しているため、
          // 招待メール送信失敗をフォームエラーにはしない。
        }

        options?.onSuccess?.(
          created,
        );
      } catch (submitError: unknown) {
        setError(
          getErrorMessage(
            submitError,
          ),
        );
      } finally {
        setSubmitting(false);
      }
    },
    [
      firstName,
      lastName,
      firstNameKana,
      lastNameKana,
      email,
      permissionsByCategory,
      category,
      permissionsText,
      brandsText,
      options,
    ],
  );

  return {
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email,
    category,
    permissionsText,
    brandsText,
    submitting,
    error,

    allPermissions,
    permissionsByCategory,
    permissionCategories,
    permissionCategoryList,

    allBrands,
    brandRows,

    setFirstName,
    setLastName,
    setFirstNameKana,
    setLastNameKana,
    setEmail,
    setCategory,
    setPermissionsText,
    setBrandsText,
    setError,

    handleSubmit,
  };
}