// frontend/member/src/hooks/useMemberDetail.ts

import { useCallback, useEffect, useState } from "react";
import type { Member } from "../../domain/entity/member";

// ★ 正しいサービスを import
import { fetchMemberDetail } from "../../application/memberDetailService";

// ブランド一覧取得用（id → name 変換用）
import { listBrands, type BrandRow } from "../../../../brand/src/application/brandService";

export function useMemberDetail(memberId?: string) {
  const [member, setMember] = useState<Member | null>(null);
  const [brandRows, setBrandRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const load = useCallback(async () => {
    if (!memberId) return;

    setLoading(true);
    setError(null);

    try {
      const normalized = await fetchMemberDetail(memberId);
      setMember(normalized);
    } catch (e: any) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setLoading(false);
    }
  }, [memberId]);

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

    (async () => {
      try {
        const rows = await listBrands(companyId);
        setBrandRows(rows);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useMemberDetail] failed to load brands", e);
        setBrandRows([]);
      }
    })();
  }, [member?.companyId]);

  // PageHeader 用の表示名
  const memberName = (() => {
    if (!member) return "不明なメンバー";
    const full = `${member.lastName ?? ""} ${member.firstName ?? ""}`.trim();
    return full || "招待中";
  })();

  // 所属ブランドID一覧（存在しない場合は空配列）
  const assignedBrands: string[] =
    (member?.assignedBrands as string[] | null | undefined) ?? [];

  // 権限一覧（存在しない場合は空配列）
  const permissions: string[] = member?.permissions ?? [];

  return {
    member,
    memberName,
    assignedBrands,
    permissions,
    brandRows,
    loading,
    error,
    reload: load,
  };
}
