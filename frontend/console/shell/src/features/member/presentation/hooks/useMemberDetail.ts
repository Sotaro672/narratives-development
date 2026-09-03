// frontend/console/shell/src/features/member/presentation/hooks/useMemberDetail.ts

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  fetchMemberDetailById,
  type MemberDetail,
} from "../../application/memberDetailService";

import {
  listBrands,
  type BrandRow,
} from "../../../brand/application/brandService";

/**
 * メンバー詳細hook
 *
 * IMPORTANT:
 * - memberIdにはFirestore Member document IDを渡す
 * - Firebase Authentication UIDは詳細取得キーとして使用しない
 *
 * Backend:
 * - GET /members/by-id/{memberId}はFirestore Member docId専用
 */
export function useMemberDetail(memberId?: string) {
  const [member, setMember] = useState<MemberDetail | null>(null);
  const [brandRows, setBrandRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(async () => {
    if (!memberId) {
      setMember(null);
      setError(null);
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await fetchMemberDetailById(memberId);
      setMember(result);
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
  }, [memberId]);

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * Member取得後、認証中の会社に所属するブランド一覧を取得する。
   * companyIdによるスコープはBackend側で行うため、listBrandsにはcompanyIdを渡さない。
   */
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

  /**
   * PageHeader用の表示名。
   * Backend responseのdisplayNameを正として扱う。
   * 招待前Memberでは氏名が空の場合があるためemailを表示する。
   */
  const memberName = useMemo(() => {
    if (!member) {
      return "不明なメンバー";
    }

    if (member.displayName) {
      return member.displayName;
    }

    return member.email;
  }, [member]);

  const assignedBrands = useMemo<string[]>(
    () => member?.assignedBrands ?? [],
    [member?.assignedBrands],
  );

  const permissions = useMemo<string[]>(
    () => member?.permissions ?? [],
    [member?.permissions],
  );

  const groupedPermissionsByCategory = useMemo(
    () => member?.permissionGroups ?? {},
    [member?.permissionGroups],
  );

  const hasGroupedPermissions =
    permissions.length > 0 &&
    Object.keys(groupedPermissionsByCategory).length > 0;

  /**
   * Firebase UIDが未設定のMemberを招待中として扱う。
   * nullのMemberは招待中とは判定しない。
   */
  const isInvitationPending = member !== null && member.uid === "";

  const permissionsLoading = false;

  return {
    member,
    memberName,
    assignedBrands,
    permissions,
    brandRows,
    loading,
    error,
    reload: load,
    isInvitationPending,
    permissionsLoading,
    groupedPermissionsByCategory,
    hasGroupedPermissions,
  };
}