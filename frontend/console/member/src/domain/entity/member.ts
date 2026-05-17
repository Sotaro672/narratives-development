// frontend/console/member/src/domain/entity/member.ts

/** Email バリデーション（backend の emailRe 相当） */
const emailRe = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/**
 * Member
 * backend/internal/domain/member/entity.go の Member に対応。
 *
 * - id は Firestore member document ID
 * - uid は Firebase Auth UID
 * - GET /members/{uid} には uid を使う
 * - PATCH /members/{docId} には id を使う
 * - 日付は ISO8601 文字列（例: "2025-01-10T00:00:00Z"）を想定
 * - Firestore/GraphQL とのやり取りを考慮し、文字列系フィールドは string | null を許容
 * - 役割（role）は廃止。権限は permissions で表現。
 */
export interface Member {
  /** Firestore member document ID */
  id: string;

  /** Firebase Auth UID */
  uid?: string | null;

  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;

  /** 姓＋名を結合したフルネーム（lastName → firstName） */
  fullName?: string | null;

  /** backend の displayName（lastName + firstName） */
  displayName?: string | null;

  /** 空文字 or undefined の場合は「未設定」扱い（backend と同様の解釈） */
  email?: string | null;

  /** Permission.Name の配列（backend: Permissions） */
  permissions: string[];

  /** 割当ブランドIDの配列（backend: AssignedBrands） */
  assignedBrands?: string[] | null;

  /** 所属会社ID（backend と同期：存在しない/未設定なら null） */
  companyId?: string | null;

  /** member status（例: active / invited など） */
  status?: string | null;

  createdAt: string; // ISO8601
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;

  /**
   * Detail UI 用の権限グルーピング。
   * application 層で groupPermissionsByCategory から付与される。
   */
  permissionGroups?: Record<string, string[]>;

  /**
   * Detail UI 用の権限カテゴリ一覧。
   * application 層で groupPermissionsByCategory から付与される。
   */
  permissionCategories?: string[];
}

/**
 * Member から表示用フルネームを取得
 * - lastName + firstName を優先
 * - 無ければ displayName
 * - 無ければ fullName
 * - どちらも無ければ空文字
 */
export function getMemberFullName(member: Member): string {
  const ln = (member.lastName ?? "").trim();
  const fn = (member.firstName ?? "").trim();
  const composed = `${ln} ${fn}`.trim();
  const displayName = (member.displayName ?? "").trim();
  const fullField = (member.fullName ?? "").trim();

  return composed || displayName || fullField || "";
}

/**
 * MemberPatch
 * backend/internal/domain/member/entity.go の MemberPatch に対応。
 * - usecase / repository レイヤで部分更新時に利用
 * - undefined は「この項目は更新しない」、null は「null に更新する」意図
 * - 役割（role）は廃止済み
 *
 * NOTE:
 * - uid は PATCH /members/{docId} では更新しない
 * - uid bind は /members/{docId}/bind-firebase-uid 側で扱う
 */
export interface MemberPatch {
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  permissions?: string[] | null;
  assignedBrands?: string[] | null;

  /** 所属会社IDの部分更新 */
  companyId?: string | null;

  /** member status の部分更新 */
  status?: string | null;

  createdAt?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/** Domain 相当のエラー種別（必要に応じて Error クラス化してもよい） */
export const MemberError = {
  InvalidID: "member: invalid id",
  InvalidEmail: "member: invalid email",
  InvalidCreatedAt: "member: invalid createdAt",
  InvalidUpdatedAt: "member: invalid updatedAt",
  InvalidUpdatedBy: "member: invalid updatedBy",
  InvalidDeletedAt: "member: invalid deletedAt",
  InvalidDeletedBy: "member: invalid deletedBy",
  NotFound: "member: not found",
  Conflict: "member: conflict",
  PreconditionFailed: "member: precondition failed",
} as const;

export type MemberErrorCode = (typeof MemberError)[keyof typeof MemberError];

/** Permission カタログ用の最小型（backend: permdom.Permission の Name フィールド対応） */
export interface PermissionCatalogItem {
  name: string;
}

/**
 * Member生成用ヘルパ
 * - backend の New / NewFromStringsTime の簡略版
 * - createdAt/updatedAt は ISO8601 文字列想定
 * - 「undefined は使わず、必要に応じて null を使う」方針
 */
export function createMember(params: {
  id: string;

  /** Firebase Auth UID。招待前 member では null / 空の可能性あり */
  uid?: string | null;

  createdAt: string;
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  permissions?: string[];
  assignedBrands?: string[] | null;

  /** 所属会社ID（未設定は null） */
  companyId?: string | null;

  /** member status */
  status?: string | null;

  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
  displayName?: string | null;
}): Member {
  const member: Member = {
    id: params.id,
    uid:
      params.uid !== undefined && params.uid !== null && params.uid !== ""
        ? params.uid
        : null,
    createdAt: params.createdAt,
    permissions: dedup(params.permissions ?? []),
    updatedAt: params.updatedAt ?? null,
    updatedBy: params.updatedBy ?? null,
    deletedAt: params.deletedAt ?? null,
    deletedBy: params.deletedBy ?? null,
    companyId:
      params.companyId !== undefined && params.companyId !== null
        ? params.companyId
        : null,
    status:
      params.status !== undefined && params.status !== null
        ? params.status
        : null,
  };

  member.firstName =
    params.firstName !== undefined && params.firstName !== null
      ? params.firstName
      : null;

  member.lastName =
    params.lastName !== undefined && params.lastName !== null
      ? params.lastName
      : null;

  member.firstNameKana =
    params.firstNameKana !== undefined && params.firstNameKana !== null
      ? params.firstNameKana
      : null;

  member.lastNameKana =
    params.lastNameKana !== undefined && params.lastNameKana !== null
      ? params.lastNameKana
      : null;

  member.email =
    params.email !== undefined && params.email !== null ? params.email : null;

  if (params.assignedBrands && params.assignedBrands.length > 0) {
    member.assignedBrands = dedup(params.assignedBrands);
  } else {
    member.assignedBrands = null;
  }

  // fullName / displayName を lastName → firstName で組み立てる
  {
    const ln = (member.lastName ?? "").trim();
    const fn = (member.firstName ?? "").trim();
    const full = `${ln} ${fn}`.trim();

    member.fullName = full !== "" ? full : null;

    member.displayName =
      params.displayName !== undefined && params.displayName !== null
        ? params.displayName
        : member.fullName;
  }

  const error = validateMember(member);
  if (error) {
    throw new Error(error);
  }

  return member;
}

/**
 * Member の妥当性検証
 * - 問題なければ null を返す
 * - 問題があれば MemberErrorCode を返す
 * - 役割（role）検証は削除済み
 */
export function validateMember(member: Member): MemberErrorCode | null {
  if (!member.id) {
    return MemberError.InvalidID;
  }

  if (member.email && !emailRe.test(member.email)) {
    return MemberError.InvalidEmail;
  }

  if (!member.createdAt) {
    return MemberError.InvalidCreatedAt;
  }

  // updatedBy / deletedBy の簡易チェック
  if (member.updatedBy !== undefined && member.updatedBy === "") {
    return MemberError.InvalidUpdatedBy;
  }

  if (member.deletedBy !== undefined && member.deletedBy === "") {
    return MemberError.InvalidDeletedBy;
  }

  return null;
}

/**
 * Permission カタログに基づいて Permissions を設定（backend: SetPermissionsByName 相当）
 * - 存在しない Permission 名は無視
 * - 重複排除 & ソート
 */
export function setPermissionsByName(
  member: Member,
  names: string[],
  catalog: PermissionCatalogItem[],
): Member {
  const allow = new Set(
    catalog
      .map((p) => p.name.trim())
      .filter((n) => n.length > 0),
  );

  const seen = new Set<string>();
  const out: string[] = [];

  for (const raw of names) {
    const n = raw.trim();
    if (!n || !allow.has(n) || seen.has(n)) continue;
    seen.add(n);
    out.push(n);
  }

  out.sort();

  return {
    ...member,
    permissions: out,
  };
}

/** 現在の Permissions がカタログに含まれるか検証（backend: ValidatePermissions 相当） */
export function validatePermissionsWithCatalog(
  member: Member,
  catalog: PermissionCatalogItem[],
): boolean {
  const allow = new Set(
    catalog
      .map((p) => p.name.trim())
      .filter((n) => n.length > 0),
  );

  return member.permissions.every((raw) => allow.has(raw.trim()));
}

/** 指定 Permission.Name を保持しているか（backend: HasPermission 相当） */
export function hasPermission(member: Member, name: string): boolean {
  const target = name.trim().toLowerCase();
  if (!target) return false;

  return member.permissions.some(
    (p) => p.trim().toLowerCase() === target,
  );
}

/** 配列の重複排除 + 空文字除去 */
function dedup(xs: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];

  for (const raw of xs) {
    const v = raw.trim();
    if (!v || seen.has(v)) continue;
    seen.add(v);
    out.push(v);
  }

  return out;
}