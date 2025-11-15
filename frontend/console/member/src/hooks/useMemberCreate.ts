// frontend/member/src/hooks/useMemberCreate.ts
import { useCallback, useMemo, useState } from "react";
import type { Member } from "../domain/entity/member";

// ★ バックエンド呼び出し用：Firebase Auth & ログイン情報（companyId）
import { auth } from "../../../shell/src/auth/config/firebaseClient";
import { useAuthContext } from "../../../shell/src/auth/application/AuthContext";

// 権限モックデータ
import {
  ALL_PERMISSIONS,
  groupPermissionsByCategory,
} from "../../../permission/src/infrastructure/mockdata/mockdata";

// Permission のカテゴリ型（＝新しい「役割」概念）
import type {
  Permission,
  PermissionCategory,
} from "../../../shell/src/shared/types/permission";

// ブランドのモックデータ（UI 用）
import {
  ALL_BRANDS,
  toBrandRows,
} from "../../../brand/src/infrastructure/mockdata/mockdata";
import type { BrandRow } from "../../../brand/src/infrastructure/mockdata/mockdata";

// バックエンドのベースURL（末尾スラッシュ除去）
const API_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/,
    ""
  ) ?? "";

export type UseMemberCreateOptions = {
  /** 作成成功時に呼ばれます（呼び出し元で navigate などを実施） */
  onSuccess?: (created: Member) => void;
};

export function useMemberCreate(options?: UseMemberCreateOptions) {
  // 認証中ユーザ（companyId をフロントでも把握しておく）
  const { user } = useAuthContext();
  const authCompanyId = user?.companyId ?? null;

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

        const perms = toArray(permissionsText);
        const brands = toArray(brandsText);

        // API へ送るリクエストボディ（handler の memberCreateRequest に対応）
        const body = {
          id,
          firstName: firstName.trim() || "",
          lastName: lastName.trim() || "",
          firstNameKana: firstNameKana.trim() || "",
          lastNameKana: lastNameKana.trim() || "",
          email: email.trim() || "",
          permissions: perms,
          assignedBrands: brands,
          // ★ companyId はサーバ側で context から上書き適用される想定だが、
          //    クライアントでも把握できている場合は一緒に送っておく（冪等）
          ...(authCompanyId ? { companyId: authCompanyId } : {}),
          status: "active",
        };

        // 認証トークン取得
        const token = await auth.currentUser?.getIdToken();
        if (!token) {
          throw new Error("未認証のためメンバーを作成できません。");
        }

        const res = await fetch(`${API_BASE}/members`, {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(body),
        });

        if (!res.ok) {
          const text = await res.text().catch(() => "");
          throw new Error(`メンバー作成に失敗しました (status ${res.status}) ${text || ""}`);
        }

        // バックエンド（usecase/repo）から返る Member をフロントの Member 型に整形
        const apiMember = (await res.json()) as any;

        const created: Member = {
          id: apiMember.id ?? id,
          firstName: apiMember.firstName ?? null,
          lastName: apiMember.lastName ?? null,
          firstNameKana: apiMember.firstNameKana ?? null,
          lastNameKana: apiMember.lastNameKana ?? null,
          email: apiMember.email ?? null,
          permissions: Array.isArray(apiMember.permissions) ? apiMember.permissions : [],
          assignedBrands: Array.isArray(apiMember.assignedBrands)
            ? apiMember.assignedBrands
            : null,
          // ISO8601 を期待
          createdAt: apiMember.createdAt ?? now,
          updatedAt: apiMember.updatedAt ?? now,
          updatedBy: apiMember.updatedBy ?? null,
          deletedAt: apiMember.deletedAt ?? null,
          deletedBy: apiMember.deletedBy ?? null,
          // companyId を返してくれる場合は受け取り、無ければ auth の値を反映
          ...(apiMember.companyId ? { companyId: apiMember.companyId } : authCompanyId ? { companyId: authCompanyId } : {}),
        } as Member;

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
