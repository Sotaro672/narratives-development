// frontend/console/member/src/presentation/hooks/useMemberCreate.ts
import { useCallback, useEffect, useMemo, useState } from "react";
import type { Member } from "../../domain/entity/member";

// ★ ログイン情報（companyId）は AuthContext から取得
import { useAuthContext } from "../../../../shell/src/auth/application/AuthContext";

// Permission のカテゴリ型（＝新しい「役割」概念）
import type {
  Permission,
  PermissionCategory,
} from "../../../../shell/src/shared/types/permission";

// Brand ドメイン型
import type { Brand } from "../../../../brand/src/domain/entity/brand";

// アプリケーションサービス（API 呼び出しロジックなど）
import {
  fetchAllPermissions,
  fetchBrandsForCurrentMember,
  groupPermissionsByCategory,
} from "../../application/memberService";
import { createMember } from "../../application/invitationService";

// UI 用 BrandRow 型（テーブル表示用）
export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  registeredAt: string; // YYYY/MM/DD
};

export type UseMemberCreateOptions = {
  /** 作成成功時に呼ばれます（呼び出し元で navigate などを実施） */
  onSuccess?: (created: Member) => void;
};

// ISO → YYYY/MM/DD
function formatDateYmd(iso?: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
}

// Brand[] → BrandRow[]
function toBrandRows(brands: Brand[]): BrandRow[] {
  return brands.map((b) => ({
    id: b.id,
    name: String(b.name ?? "").trim(),
    isActive: Boolean(b.isActive ?? true),
    registeredAt: formatDateYmd(b.createdAt),
  }));
}

export function useMemberCreate(options?: UseMemberCreateOptions) {
  // 認証中ユーザ（companyId / uid は AuthContext からも使える）
  const { user } = useAuthContext();
  const authCompanyId = user?.companyId ?? null;
  const currentMemberId = user?.uid ?? null;

  // ---- フォーム状態 ----
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [firstNameKana, setFirstNameKana] = useState("");
  const [lastNameKana, setLastNameKana] = useState("");
  const [email, setEmail] = useState("");

  // 新: PermissionCategory が「役割」相当
  const [category, setCategory] = useState<PermissionCategory>("brand");

  const [permissionsText, setPermissionsText] = useState(""); // カンマ区切り
  const [brandsText, setBrandsText] = useState(""); // カンマ区切り

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // ===== 権限カテゴリ情報（カテゴリ選択の Popover で利用） =====
  const [allPermissions, setAllPermissions] = useState<Permission[]>([]);

  // 初回マウント時に backend から権限一覧を取得
  useEffect(() => {
    (async () => {
      try {
        const items = await fetchAllPermissions(); // Service 経由
        setAllPermissions(items);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useMemberCreate] failed to load permissions", e);
        setAllPermissions([]);
      }
    })();
  }, []);

  // カテゴリごとにグルーピング
  const permissionsByCategory: Record<PermissionCategory, Permission[]> =
    useMemo(
      () => groupPermissionsByCategory(allPermissions),
      [allPermissions],
    );

  // UIで扱いやすい配列形式（カテゴリ名・件数・配列）
  const permissionCategories = useMemo(
    () =>
      (Object.keys(permissionsByCategory) as PermissionCategory[]).map(
        (cat) => ({
          key: cat,
          count: permissionsByCategory[cat]?.length ?? 0,
          permissions: permissionsByCategory[cat] ?? [],
        }),
      ),
    [permissionsByCategory],
  );

  // 選択肢としてのカテゴリ一覧
  const permissionCategoryList = useMemo(
    () => Object.keys(permissionsByCategory) as PermissionCategory[],
    [permissionsByCategory],
  );

  // ===== ブランド（currentMember.companyId ベースで取得） =====
  const [allBrands, setAllBrands] = useState<Brand[]>([]);
  const [brandRows, setBrandRows] = useState<BrandRow[]>([]);

  // ✅ 選択されたブランドID一覧
  const [selectedBrandIds, setSelectedBrandIds] = useState<string[]>([]);

  useEffect(() => {
    (async () => {
      try {
        // ★ ここで memberService 経由で currentMember → companyId → brands を取得
        const brands = await fetchBrandsForCurrentMember();
        setAllBrands(brands);
        setBrandRows(toBrandRows(brands));
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error(
          "[useMemberCreate] failed to load brands via memberService",
          e,
        );
        setAllBrands([]);
        setBrandRows([]);
      }
    })();
  }, []);

  // ✅ ブランド選択のトグル
  const toggleBrandSelection = useCallback((brandId: string) => {
    setSelectedBrandIds((prev) =>
      prev.includes(brandId)
        ? prev.filter((id) => id !== brandId)
        : [...prev, brandId],
    );
  }, []);

  const handleSubmit = useCallback(
    async (e?: React.FormEvent) => {
      e?.preventDefault?.();
      setError(null);
      setSubmitting(true);
      try {
        const created = await createMember({
          firstName,
          lastName,
          firstNameKana,
          lastNameKana,
          email,
          permissionsText,
          brandsText,
          authCompanyId,
          currentMemberId,
          // ✅ ここで選択されたブランドIDを渡す
          assignedBrandIds: selectedBrandIds,
        });

        // 呼び出し元へ通知
        options?.onSuccess?.(created);
      } catch (err: any) {
        setError(err?.message ?? String(err));
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
      permissionsText,
      brandsText,
      authCompanyId,
      currentMemberId,
      selectedBrandIds,
      options,
    ],
  );

  // ─────────────────────────────────────────────────────────────
  // 戻り値
  // ─────────────────────────────────────────────────────────────
  return {
    // 値
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

    // 権限データ（カテゴリ表示用）
    allPermissions,
    permissionsByCategory,
    permissionCategories,
    permissionCategoryList,

    // ブランド（UI での表示・選択に利用可能）
    allBrands,
    brandRows,
    selectedBrandIds,

    // セッター
    setFirstName,
    setLastName,
    setFirstNameKana,
    setLastNameKana,
    setEmail,
    setCategory,
    setPermissionsText,
    setBrandsText,
    setError,
    setSelectedBrandIds,

    // 動作
    toggleBrandSelection,
    handleSubmit,
  };
}
