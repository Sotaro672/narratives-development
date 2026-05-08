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
} from "../../application/memberListService";

// メンバー作成 & 招待メール送信
import { createMember, parseCommaSeparated } from "../../application/memberCreateService";
import { sendMemberInvitation } from "../../application/invitationService";

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

  // 任意：テキスト入力でも permissions / brands を指定できるよう残しておく
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

  /**
   * メンバー作成 + 招待メール送信
   * overrides で permissions / assignedBrandIds を画面側から上書き可能
   */
  const handleSubmit = useCallback(
    async (
      e?: React.FormEvent,
      overrides?: {
        permissions?: string[];
        assignedBrandIds?: string[];
      },
    ) => {
      e?.preventDefault?.();
      setError(null);
      setSubmitting(true);
      try {
        // 役割カテゴリ由来の権限
        const permissionsFromCategory =
          permissionsByCategory[category]?.map((p) => (p as any).name) ?? [];

        // テキスト入力由来の権限
        const permissionsFromText = permissionsText
          ? parseCommaSeparated(permissionsText)
          : [];

        // マージ & 重複除去
        const mergedPermissions = Array.from(
          new Set([...permissionsFromCategory, ...permissionsFromText]),
        );

        // 画面からの上書きがあればそちらを優先
        const finalPermissions =
          overrides?.permissions && overrides.permissions.length > 0
            ? overrides.permissions
            : mergedPermissions;

        // brandsText からのフォールバック
        const fallbackBrandIds = brandsText
          ? parseCommaSeparated(brandsText)
          : [];

        const finalAssignedBrandIds =
          overrides?.assignedBrandIds && overrides.assignedBrandIds.length > 0
            ? overrides.assignedBrandIds
            : fallbackBrandIds;

        // デバッグログ
        // eslint-disable-next-line no-console
        console.log("[useMemberCreate] submit payload (frontend)", {
          permissionsFromCategory,
          permissionsFromText,
          mergedPermissions,
          finalPermissions,
          fallbackBrandIds,
          finalAssignedBrandIds,
        });

        // 1. メンバー作成
        const created = await createMember({
          firstName,
          lastName,
          firstNameKana,
          lastNameKana,
          email,
          permissions: finalPermissions,
          assignedBrandIds: finalAssignedBrandIds,
          authCompanyId,
          currentMemberId,
        });

        // 2. 招待メール送信（失敗してもフォームエラーにはしない）
        try {
          await sendMemberInvitation(created.id, created.email ?? null);
        } catch (inviteErr) {
          // eslint-disable-next-line no-console
          console.error(
            "[useMemberCreate] 招待メール送信中にエラーが発生しました",
            inviteErr,
          );
        }

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
      permissionsByCategory,
      category,
      permissionsText,
      brandsText,
      authCompanyId,
      currentMemberId,
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
