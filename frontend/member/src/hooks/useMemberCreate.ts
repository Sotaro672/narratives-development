// frontend/member/src/hooks/useMemberCreate.ts
import { useCallback, useMemo, useState } from "react";
import type { Member } from "../domain/entity/member";
import { MemberRepositoryFS } from "../infrastructure/firestore/memberRepositoryFS";

// 権限モックデータの取り込み
import {
  ALL_PERMISSIONS,
  groupPermissionsByCategory,
} from "../../../permission/src/infrastructure/mockdata/mockdata";

// Permission のカテゴリ型（＝新しい「役割」概念）
import type {
  Permission,
  PermissionCategory,
} from "../../../shell/src/shared/types/permission";

// ブランドのモックデータをインポート（UIでの選択/表示に備える）
import {
  ALL_BRANDS,
  toBrandRows,
} from "../../../brand/src/infrastructure/mockdata/mockdata";
import type { BrandRow } from "../../../brand/src/infrastructure/mockdata/mockdata";

export type UseMemberCreateOptions = {
  /** 作成成功時に呼ばれます（呼び出し元で navigate などを実施） */
  onSuccess?: (created: Member) => void;
};

export function useMemberCreate(options?: UseMemberCreateOptions) {
  const repo = useMemo(() => new MemberRepositoryFS(), []);

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
  const allPermissions: Permission[] = ALL_PERMISSIONS;

  const permissionsByCategory = useMemo(
    () => groupPermissionsByCategory(ALL_PERMISSIONS),
    []
  );

  // UIで扱いやすい配列形式（カテゴリ名・件数・配列）
  const permissionCategories = useMemo(
    () =>
      (Object.keys(permissionsByCategory) as PermissionCategory[]).map((cat) => ({
        key: cat,
        count: permissionsByCategory[cat]?.length ?? 0,
        permissions: permissionsByCategory[cat] ?? [],
      })),
    [permissionsByCategory]
  );

  // 選択肢としてのカテゴリ一覧
  const permissionCategoryList = useMemo(
    () => Object.keys(permissionsByCategory) as PermissionCategory[],
    [permissionsByCategory]
  );

  // ===== ブランド（UI 連携用に公開しておく） =====
  const allBrands = ALL_BRANDS;
  const brandRows: BrandRow[] = useMemo(() => toBrandRows(ALL_BRANDS), []);

  const toArray = (s: string) =>
    s
      .split(",")
      .map((x) => x.trim())
      .filter(Boolean);

  const handleSubmit = useCallback(
    async (e?: React.FormEvent) => {
      e?.preventDefault?.();
      setError(null);
      setSubmitting(true);
      try {
        const id = crypto.randomUUID();
        const now = new Date().toISOString();

        const member: Member = {
          id,
          firstName: firstName.trim() || undefined,
          lastName: lastName.trim() || undefined,
          firstNameKana: firstNameKana.trim() || undefined,
          lastNameKana: lastNameKana.trim() || undefined,
          email: email.trim() || undefined,
          permissions: toArray(permissionsText),
          assignedBrands: (() => {
            const arr = toArray(brandsText);
            return arr.length ? arr : undefined;
          })(),
          createdAt: now,
          updatedAt: now,
          updatedBy: "console",
          deletedAt: null,
          deletedBy: null,
        };

        const created = await repo.create(member);
        options?.onSuccess?.(created);
      } catch (err: any) {
        setError(err?.message ?? String(err));
      } finally {
        setSubmitting(false);
      }
    },
    [
      repo,
      firstName,
      lastName,
      firstNameKana,
      lastNameKana,
      email,
      permissionsText,
      brandsText,
      options,
    ]
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

    // 動作
    handleSubmit,
  };
}
