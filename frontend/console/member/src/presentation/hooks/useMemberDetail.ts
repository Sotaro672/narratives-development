// frontend/member/src/hooks/useMemberDetail.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import type { Member } from "../../domain/entity/member";

// ★ メンバー詳細取得サービス
// NOTE:
// fetchMemberDetailByUid は backend の GET /members/{uid} を使う。
// そのため、Firestore member docId ではなく Firebase Auth UID を渡すこと。
import { fetchMemberDetailByUid } from "../../application/memberDetailService";

// ブランド一覧取得用（id → name 変換用）
import {
  listBrands,
  type BrandRow,
} from "../../../../brand/src/application/brandService";

// PermissionCategory 型（backend の PermissionCategory と対応）
import type { PermissionCategory } from "../../../../shell/src/shared/types/permission";

// ★ 権限名 → カテゴリ別グルーピング（TS 版カタログヘルパ）
import { groupPermissionsByCategory } from "../../../../permission/src/application/permissionCatalog";

/**
 * メンバー詳細 hook
 *
 * IMPORTANT:
 * - memberUid には Firebase Auth UID を渡す
 * - Firestore member docId を渡してはいけない
 *
 * backend:
 * - GET /members/{uid} は Firebase UID 専用
 * - PATCH /members/{docId} は Firestore member docId 専用
 */
export function useMemberDetail(memberUid?: string) {
  const [member, setMember] = useState<Member | null>(null);
  const [brandRows, setBrandRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

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
      const normalized = await fetchMemberDetailByUid(normalizedMemberUid);
      setMember(normalized);
    } catch (e: unknown) {
      setError(e instanceof Error ? e : new Error(String(e)));
      setMember(null);
    } finally {
      setLoading(false);
    }
  }, [normalizedMemberUid]);

  useEffect(() => {
    void load();
  }, [load]);

  // member.companyId からブランド一覧を取得して brandRows にセット
  useEffect(() => {
    const companyId = String(member?.companyId ?? "").trim();
    if (!companyId) {
      setBrandRows([]);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const rows = await listBrands(companyId);
        if (!cancelled) {
          setBrandRows(rows);
        }
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useMemberDetail] failed to load brands", e);
        if (!cancelled) {
          setBrandRows([]);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [member?.companyId]);

  // PageHeader 用の表示名
  const memberName = useMemo(() => {
    if (!member) return "不明なメンバー";

    const full = `${member.lastName ?? ""} ${member.firstName ?? ""}`.trim();
    const displayName = String(member.displayName ?? "").trim();
    const fullName = String(member.fullName ?? "").trim();

    return full || displayName || fullName || "招待中";
  }, [member]);

  // 所属ブランドID一覧（存在しない場合は空配列）
  const assignedBrands: string[] = useMemo(() => {
    return (member?.assignedBrands as string[] | null | undefined) ?? [];
  }, [member?.assignedBrands]);

  // 権限一覧（存在しない場合は空配列）
  const permissions: string[] = useMemo(() => {
    return member?.permissions ?? [];
  }, [member?.permissions]);

  // ─────────────────────────────────────
  // 権限名 → Category ごとにグルーピング
  // Firestore には "wallet.view" 等しか入っていないため、
  // TS 側の permissionCatalog の groupPermissionsByCategory を利用する。
  // ─────────────────────────────────────

  // PermissionCard 用のローディングフラグ
  // ※ バックエンドへの追加フェッチは行わず、同期計算だけなので false 固定
  const permissionsLoading = false;

  const groupedPermissionsByCategory = useMemo(() => {
    if (permissions.length === 0) {
      return {} as Partial<Record<PermissionCategory, string[]>>;
    }

    return groupPermissionsByCategory(
      permissions,
    ) as Partial<Record<PermissionCategory, string[]>>;
  }, [permissions]);

  const hasGroupedPermissions =
    Object.keys(groupedPermissionsByCategory).length > 0 &&
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

    // ★ PermissionCard 用
    permissionsLoading,
    groupedPermissionsByCategory,
    hasGroupedPermissions,
  };
}