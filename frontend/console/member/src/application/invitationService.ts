// frontend/console/member/src/application/invitationService.ts
import type { Member } from "../domain/entity/member";

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// memberService から API_BASE を利用（同じバックエンドURLを共有）
import { API_BASE } from "./memberService";

// ─────────────────────────────────────────────
// Utility
// ─────────────────────────────────────────────

// カンマ区切り文字列 → string[]
export const parseCommaSeparated = (s: string): string[] =>
  s
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);

// ─────────────────────────────────────────────
// Member 作成 & 招待メール送信
// ─────────────────────────────────────────────

export type CreateMemberParams = {
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email: string;

  // UI 上のテキスト入力（従来どおり残しておく）
  permissionsText: string;
  brandsText: string;

  // ✅ 新: チェックボックスなどで選択されたブランドID一覧
  assignedBrandIds?: string[];

  authCompanyId: string | null;
  currentMemberId: string | null;
};

/**
 * メンバー作成 & 招待メール送信
 * - POST /members
 * - POST /members/{memberId}/invitation
 */
export async function createMember(
  params: CreateMemberParams,
): Promise<Member> {
  const {
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email,
    permissionsText,
    brandsText,
    assignedBrandIds,
    authCompanyId,
    currentMemberId,
  } = params;

  const id = crypto.randomUUID();
  const now = new Date().toISOString();

  const perms = parseCommaSeparated(permissionsText);

  // ✅ ここで assignedBrandIds を優先し、なければカンマ区切り文字列から生成
  const brands =
    Array.isArray(assignedBrandIds) && assignedBrandIds.length > 0
      ? assignedBrandIds
      : parseCommaSeparated(brandsText);

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
    ...(authCompanyId ? { companyId: authCompanyId } : {}),
    status: "active",
  };

  // 認証トークン取得
  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("未認証のためメンバーを作成できません。");
  }

  const url = `${API_BASE}/members`;
  // eslint-disable-next-line no-console
  console.log("[invitationService.createMember] POST", url, body);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `メンバー作成に失敗しました (status ${res.status}) ${text || ""}`,
    );
  }

  // HTML が返ってきていないかチェック（env ミス検出用）
  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `サーバーから JSON ではないレスポンスが返却されました (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または API_BASE=${API_BASE} を確認してください。\n` +
        text.slice(0, 200),
    );
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
    permissions: Array.isArray(apiMember.permissions)
      ? apiMember.permissions
      : [],
    assignedBrands: Array.isArray(apiMember.assignedBrands)
      ? apiMember.assignedBrands
      : null,
    ...(apiMember.companyId
      ? { companyId: apiMember.companyId }
      : authCompanyId
        ? { companyId: authCompanyId }
        : {}),
    createdAt: apiMember.createdAt ?? now,
    createdBy: apiMember.createdBy ?? currentMemberId ?? null,
    updatedAt: apiMember.updatedAt ?? now,
    updatedBy: apiMember.updatedBy ?? currentMemberId ?? null,
    deletedAt: apiMember.deletedAt ?? null,
    deletedBy: apiMember.deletedBy ?? null,
  } as Member;

  // ─────────────────────────────────────
  // 招待メール送信トリガー
  // ─────────────────────────────────────
  if (created.email) {
    const inviteUrl = `${API_BASE}/members/${encodeURIComponent(
      created.id,
    )}/invitation`;

    // eslint-disable-next-line no-console
    console.log(
      "[invitationService.createMember] POST (invitation)",
      inviteUrl,
    );

    try {
      const inviteRes = await fetch(inviteUrl, {
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({}),
      });

      if (!inviteRes.ok) {
        const inviteText = await inviteRes.text().catch(() => "");
        // eslint-disable-next-line no-console
        console.error(
          `[invitationService.createMember] 招待メール送信に失敗しました (status ${inviteRes.status}) ${inviteText}`,
        );
      } else {
        // eslint-disable-next-line no-console
        console.log(
          "[invitationService.createMember] 招待メール送信リクエスト成功",
        );
      }
    } catch (invErr) {
      // eslint-disable-next-line no-console
      console.error(
        "[invitationService.createMember] 招待メール送信中にエラーが発生しました",
        invErr,
      );
    }
  }

  return created;
}
