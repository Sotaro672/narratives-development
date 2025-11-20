// frontend/console/member/src/application/memberService.ts
import type { Member } from "../domain/entity/member";
import type { MemberFilter } from "../domain/repository/memberRepository";
import type { Page } from "../../../shell/src/shared/types/common/common";

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// Permission の型
import type {
  Permission,
  PermissionCategory,
} from "../../../shell/src/shared/types/permission";

// 権限一覧を backend (/permissions) から取得する HTTP リポジトリ
import { PermissionRepositoryHTTP } from "../../../permission/src/infrastructure/http/permissionRepositoryHTTP";

// ★ Brand ドメイン／HTTP リポジトリ
import type { Brand } from "../../../brand/src/domain/entity/brand";
import { BrandRepositoryHTTP } from "../../../brand/src/infrastructure/http/brandRepositoryHTTP";

// ★ Brand 名整形ヘルパ（backend brand.Service.FormatName と揃える）
import { formatBrandName } from "../../../brand/src/application/brandService";

// API 呼び出しロジック（既存の infrastructure/query を利用）
import {
  fetchMemberListWithToken,
  fetchMemberByIdWithToken,
  formatLastFirst,
} from "../infrastructure/query/memberQuery";

export type MemberListResult = {
  members: Member[];
  /** ID -> 「姓 名」表示名 */
  nameMap: Record<string, string>;
};

// ─────────────────────────────────────────────
// Backend base URL（useMemberList / useMemberDetail と同じ構成）
// ─────────────────────────────────────────────
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

// 最終的に使うベース URL
export const API_BASE = ENV_BASE || FALLBACK_BASE;

// Permission 一覧を取得するリポジトリ（シングルトン的に使う）
const permissionRepo = new PermissionRepositoryHTTP();

// Brand 一覧を取得するリポジトリ（シングルトン的に使う）
const brandRepo = new BrandRepositoryHTTP();

// カンマ区切り文字列 → string[]
export const parseCommaSeparated = (s: string): string[] =>
  s
    .split(",")
    .map((x) => x.trim())
    .filter(Boolean);

// ─────────────────────────────────────────────
// Permission 関連サービス
// ─────────────────────────────────────────────

/**
 * backend/internal/domain/permission/catalog.go 経由で
 * /permissions から取得した一覧を返す
 */
export async function fetchAllPermissions(): Promise<Permission[]> {
  const pageResult = await permissionRepo.list(); // GET /permissions
  return pageResult.items;
}

/**
 * Permission の配列をカテゴリごとにグルーピング
 * （hook 側で useMemo してもよいが、ロジック自体は共通化しておく）
 */
export function groupPermissionsByCategory(
  allPermissions: Permission[],
): Record<PermissionCategory, Permission[]> {
  const map: Record<string, Permission[]> = {};
  for (const p of allPermissions) {
    const cat = (p.category || "brand") as PermissionCategory;
    if (!map[cat]) map[cat] = [];
    map[cat].push(p);
  }
  return map as Record<PermissionCategory, Permission[]>;
}

// ─────────────────────────────────────────────
// Member / currentMember 関連サービス
// ─────────────────────────────────────────────

/**
 * FirebaseAuth.currentUser.uid に対応する Member を取得する
 * - /members/{uid} を叩いて currentMember を取得
 * - 取得できない場合は null
 */
export async function fetchCurrentMember(): Promise<Member | null> {
  const currentUser = auth.currentUser;
  if (!currentUser) {
    console.warn("[memberService] fetchCurrentMember: no auth.currentUser");
    return null;
  }

  const uid = currentUser.uid;
  const token = await currentUser.getIdToken();

  // ログ出力
  // 例: [memberService] fetchCurrentMember uid: 06GeY... GET https://.../members/06GeY...
  // eslint-disable-next-line no-console
  console.log(
    "[memberService] fetchCurrentMember uid:",
    uid,
    "GET",
    `${API_BASE}/members/${encodeURIComponent(uid)}`,
  );

  try {
    const member = await fetchMemberByIdWithToken(token, uid);
    if (!member) {
      console.warn("[memberService] fetchCurrentMember: member not found");
      return null;
    }
    return member;
  } catch (e) {
    console.error("[memberService] fetchCurrentMember error:", e);
    return null;
  }
}

// ─────────────────────────────────────────────
// Brand 関連サービス
// ─────────────────────────────────────────────

/**
 * currentMember の companyId と同じブランドのみを取得する
 *
 * - currentMember を backend /members/{uid} から取得
 * - そこから companyId を取り出し、BrandRepositoryHTTP 経由で /brands を叩く
 */
export async function fetchBrandsForCurrentMember(): Promise<Brand[]> {
  const current = await fetchCurrentMember();
  const companyId = (current?.companyId ?? "").trim();

  if (!companyId) {
    console.warn(
      "[memberService] fetchBrandsForCurrentMember: currentMember.companyId is empty",
    );
    return [];
  }

  return fetchBrandsByCompany(companyId);
}

/**
 * 指定した companyId のブランドのみを取得する
 *
 * - BrandRepositoryHTTP 経由で /brands を叩き、companyId でサーバ側フィルタ
 * - 念のためフロント側でも companyId でフィルタ（安全側）
 */
export async function fetchBrandsByCompany(
  companyId: string | null,
): Promise<Brand[]> {
  if (!companyId) {
    return [];
  }

  // eslint-disable-next-line no-console
  console.log("[memberService] fetchBrandsByCompany companyId =", companyId);

  const pageResult = await brandRepo.list({
    filter: {
      companyId,
      deleted: false,
      isActive: true,
    },
    sort: {
      column: "created_at",
      order: "desc",
    },
    page: 1,
    perPage: 200,
  });

  const items = (pageResult.items ?? []) as Brand[];

  // 念のためフロント側でも companyId で絞り込み
  const filtered = items.filter((b) => (b.companyId ?? "") === companyId);

  // eslint-disable-next-line no-console
  console.log(
    "[memberService] fetchBrandsByCompany brands =",
    filtered.map((b) => ({ id: b.id, name: b.name, companyId: b.companyId })),
  );

  return filtered;
}

// ─────────────────────────────────────────────
// Member 一覧
// ─────────────────────────────────────────────

/**
 * メンバー一覧取得（バックエンド API 経由）
 * - companyId はサーバ側で認証情報からスコープ
 * - 姓・名が両方未設定なら firstName に「招待中」を設定
 * - ★ assignedBrands に入っている brandId をブランド名（FormatName）に変換
 *   → memberManagement の「所属ブランド」列にそのまま表示できるようにする
 */
export async function fetchMemberList(
  page: Page,
  filter: MemberFilter,
): Promise<MemberListResult> {
  // Firebase Auth から ID トークンを取得
  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error(
      "未認証のためメンバー一覧を取得できません。（currentUser が null）",
    );
  }

  const token = await currentUser.getIdToken();
  // eslint-disable-next-line no-console
  console.log("[memberService.fetchMemberList] currentUser.uid:", currentUser.uid);

  const { items } = await fetchMemberListWithToken(token, page, filter);

  // ---------------------------------------------------
  //  ブランド一覧を取得して brandId -> brandName マップ作成
  // ---------------------------------------------------
  let brandNameMap: Record<string, string> = {};
  try {
    const brands = await fetchBrandsForCurrentMember();
    const map: Record<string, string> = {};
    for (const b of brands) {
      const id = (b.id ?? "").trim();
      if (!id) continue;
      map[id] = formatBrandName(b.name ?? "");
    }
    brandNameMap = map;
    // eslint-disable-next-line no-console
    console.log("[memberService.fetchMemberList] brandNameMap =", brandNameMap);
  } catch (e) {
    console.error(
      "[memberService.fetchMemberList] failed to load brands for current member",
      e,
    );
    brandNameMap = {};
  }

  // 姓・名と所属ブランド名を正規化
  const normalized: Member[] = items.map((m) => {
    const noFirst = !String(m.firstName ?? "").trim();
    const noLast = !String(m.lastName ?? "").trim();

    // assignedBrands: brandId[] -> brandName[]
    let assignedBrandNames: string[] | null = null;
    if (Array.isArray(m.assignedBrands) && m.assignedBrands.length > 0) {
      const names = m.assignedBrands
        .map((id) => {
          const key = String(id ?? "").trim();
          if (!key) return "";
          // brandNameMap にあればブランド名、なければ ID でフォールバック
          const name = brandNameMap[key] ?? key;
          return formatBrandName(name);
        })
        .filter((label) => label.length > 0);

      assignedBrandNames = names.length > 0 ? names : null;
    } else {
      assignedBrandNames = null;
    }

    // 姓名「招待中」補正
    const base: Member =
      noFirst && noLast
        ? ({ ...m, firstName: "招待中" } as Member)
        : (m as Member);

    // 所属ブランドをブランド名配列として上書き
    return {
      ...base,
      assignedBrands: assignedBrandNames,
    };
  });

  // 表示名マップを作成
  const nameMap: Record<string, string> = {};
  for (const m of normalized) {
    const disp = formatLastFirst(m.lastName as any, m.firstName as any);
    if (disp) {
      nameMap[m.id] = disp;
    }
  }

  return { members: normalized, nameMap };
}

/**
 * 単一メンバーの表示名「姓 名」を取得
 * - ID が不正・未認証などの場合は空文字を返す
 */
export async function fetchMemberNameLastFirstById(
  memberId: string,
): Promise<string> {
  const id = String(memberId ?? "").trim();
  if (!id) return "";

  const currentUser = auth.currentUser;
  if (!currentUser) return "";

  const token = await currentUser.getIdToken();
  const member = await fetchMemberByIdWithToken(token, id);
  if (!member) return "";

  const disp = formatLastFirst(
    member.lastName as any,
    member.firstName as any,
  );
  return disp ?? "";
}

/**
 * メンバー詳細取得
 * - /members/:id を叩いて Member を取得
 * - 姓名が空の場合も id にはフォールバックせず、firstName/lastName は null に正規化
 */
export async function fetchMemberDetail(
  memberId: string,
): Promise<Member | null> {
  const id = String(memberId ?? "").trim();
  if (!id) return null;

  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error("未認証のためメンバー情報を取得できません。");
  }

  const token = await currentUser.getIdToken();
  // eslint-disable-next-line no-console
  console.log("[memberService.fetchMemberDetail] currentUser.uid:", currentUser.uid);

  const raw = await fetchMemberByIdWithToken(token, id);
  if (!raw) return null;

  // 姓名正規化（ID にはフォールバックしない）
  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";
  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  const normalized: Member = {
    ...raw,
    id: raw.id ?? id,
    firstName: noFirst ? null : raw.firstName ?? null,
    lastName: noLast ? null : raw.lastName ?? null,
  };

  return normalized;
}

// ─────────────────────────────────────────────
// Member 作成
// ─────────────────────────────────────────────

export type CreateMemberParams = {
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email: string;
  permissionsText: string;
  brandsText: string;
  authCompanyId: string | null;
  currentMemberId: string | null;
};

/**
 * メンバー作成 & 招待メール送信
 * - POST /members
 * - POST /members/{memberId}/invitation
 */
export async function createMember(params: CreateMemberParams): Promise<Member> {
  const {
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email,
    permissionsText,
    brandsText,
    authCompanyId,
    currentMemberId,
  } = params;

  const id = crypto.randomUUID();
  const now = new Date().toISOString();

  const perms = parseCommaSeparated(permissionsText);
  const brands = parseCommaSeparated(brandsText);

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
  console.log("[memberService.createMember] POST", url, body);

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
      "[memberService.createMember] POST (invitation)",
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
          `[memberService.createMember] 招待メール送信に失敗しました (status ${inviteRes.status}) ${inviteText}`,
        );
      } else {
        // eslint-disable-next-line no-console
        console.log(
          "[memberService.createMember] 招待メール送信リクエスト成功",
        );
      }
    } catch (invErr) {
      // eslint-disable-next-line no-console
      console.error(
        "[memberService.createMember] 招待メール送信中にエラーが発生しました",
        invErr,
      );
    }
  }

  return created;
}
