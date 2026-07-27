// frontend/console/shell/src/features/member/presentation/hooks/useMemberDetail.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  fetchMemberDetailByUid,
  type MemberDetail,
} from "../../application/memberDetailService";

import {
  listBrands,
  type BrandRow,
} from "../../../brand/application/brandService";

import type { PermissionCategory } from "../../../../shared/types/permission";

import { groupPermissionsByCategory } from "../../../permission/application/permissionCatalog";

/**
 * メンバー詳細hook
 *
 * IMPORTANT:
 * - memberUidにはFirebase Authentication UIDを渡す
 * - Firestore MemberのdocIdを渡してはいけない
 *
 * Backend:
 * - GET /members/{uid}はFirebase UID専用
 * - PATCH /members/{docId}はFirestore MemberのdocId専用
 */
export function useMemberDetail(
  memberUid?: string,
) {
  const [member, setMember] =
    useState<MemberDetail | null>(null);

  const [brandRows, setBrandRows] =
    useState<BrandRow[]>([]);

  const [loading, setLoading] =
    useState(false);

  const [error, setError] =
    useState<Error | null>(null);

  const normalizedMemberUid = useMemo(
    () => String(memberUid ?? "").trim(),
    [memberUid],
  );

  const load = useCallback(async () => {
    if (!normalizedMemberUid) {
      setMember(null);
      setError(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const normalized =
        await fetchMemberDetailByUid(
          normalizedMemberUid,
        );

      setMember(normalized);
    } catch (loadError: unknown) {
      setError(
        loadError instanceof Error
          ? loadError
          : new Error(String(loadError)),
      );

      setMember(null);
    } finally {
      setLoading(false);
    }
  }, [normalizedMemberUid]);

  useEffect(() => {
    void load();
  }, [load]);

  // Member取得後、認証中の会社に所属するブランド一覧を取得する。
  // companyIdによるスコープはBackend側で行うため、
  // listBrandsにはcompanyIdを渡さない。
  useEffect(() => {
    if (!member) {
      setBrandRows([]);
      return;
    }

    let cancelled = false;

    async function loadBrands() {
      try {
        const rows = await listBrands();

        if (!cancelled) {
          setBrandRows(rows);
        }
      } catch {
        if (!cancelled) {
          setBrandRows([]);
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, [member]);

  // PageHeader用の表示名。
  // Backend responseのdisplayNameを正として扱う。
  const memberName = useMemo(() => {
    if (!member) {
      return "不明なメンバー";
    }

    const displayName =
      member.displayName.trim();

    const nameFromParts = [
      member.lastName,
      member.firstName,
    ]
      .filter((value) => value.length > 0)
      .join(" ");

    return (
      displayName ||
      nameFromParts ||
      member.email ||
      "招待中"
    );
  }, [member]);

  // 所属ブランドID一覧。
  // nullの場合は空配列として扱う。
  const assignedBrands =
    useMemo<string[]>(() => {
      return member?.assignedBrands ?? [];
    }, [member?.assignedBrands]);

  // 権限一覧。
  const permissions =
    useMemo<string[]>(() => {
      return member?.permissions ?? [];
    }, [member?.permissions]);

  // Backendへの追加取得は行わず同期計算だけなのでfalse固定。
  const permissionsLoading = false;

  const groupedPermissionsByCategory =
    useMemo(() => {
      if (permissions.length === 0) {
        return {} as Partial<
          Record<
            PermissionCategory,
            string[]
          >
        >;
      }

      return groupPermissionsByCategory(
        permissions,
      ) as Partial<
        Record<
          PermissionCategory,
          string[]
        >
      >;
    }, [permissions]);

  const hasGroupedPermissions =
    Object.keys(
      groupedPermissionsByCategory,
    ).length > 0 &&
    permissions.length > 0;

  return {
    member,
    memberName,
    assignedBrands,
    permissions,
    brandRows,
    loading,
    error,
    reload: load,

    permissionsLoading,
    groupedPermissionsByCategory,
    hasGroupedPermissions,
  };
}