// frontend/console/member/src/domain/entity/member.ts

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

  /** backend の displayName（lastName + firstName） */
  displayName?: string | null;

  /** 空文字 or undefined の場合は「未設定」扱い */
  email?: string | null;

  /** Permission.Name の配列（backend: Permissions） */
  permissions: string[];

  /** 割当ブランドIDの配列（backend: AssignedBrands） */
  assignedBrands?: string[] | null;

  /** 所属会社ID */
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
 * - backend response の displayName を優先
 * - 無ければ lastName + firstName
 * - どちらも無ければ空文字
 */
export function getMemberFullName(member: Member): string {
  const displayName = (member.displayName ?? "").trim();
  const ln = (member.lastName ?? "").trim();
  const fn = (member.firstName ?? "").trim();
  const composed = `${ln} ${fn}`.trim();

  return displayName || composed || "";
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

/** Permission カタログ用の最小型（backend: permdom.Permission の Name フィールド対応） */
export interface PermissionCatalogItem {
  name: string;
}

/**
 * Member生成用ヘルパ
 * - backend の New / NewFromStringsTime の簡略版
 * - createdAt/updatedAt は ISO8601 文字列想定
 * - validation は backend の責任とする
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
  return {
    id: params.id,
    uid:
      params.uid !== undefined && params.uid !== null && params.uid !== ""
        ? params.uid
        : null,
    firstName:
      params.firstName !== undefined && params.firstName !== null
        ? params.firstName
        : null,
    lastName:
      params.lastName !== undefined && params.lastName !== null
        ? params.lastName
        : null,
    firstNameKana:
      params.firstNameKana !== undefined && params.firstNameKana !== null
        ? params.firstNameKana
        : null,
    lastNameKana:
      params.lastNameKana !== undefined && params.lastNameKana !== null
        ? params.lastNameKana
        : null,
    email:
      params.email !== undefined && params.email !== null ? params.email : null,
    permissions: dedup(params.permissions ?? []),
    assignedBrands:
      params.assignedBrands && params.assignedBrands.length > 0
        ? dedup(params.assignedBrands)
        : null,
    companyId:
      params.companyId !== undefined && params.companyId !== null
        ? params.companyId
        : null,
    status:
      params.status !== undefined && params.status !== null
        ? params.status
        : null,
    createdAt: params.createdAt,
    updatedAt: params.updatedAt ?? null,
    updatedBy: params.updatedBy ?? null,
    deletedAt: params.deletedAt ?? null,
    deletedBy: params.deletedBy ?? null,
    displayName:
      params.displayName !== undefined && params.displayName !== null
        ? params.displayName
        : `${params.lastName ?? ""} ${params.firstName ?? ""}`.trim() || null,
  };
}

/**
 * Permission カタログに基づいて Permissions を設定（backend: SetPermissionsByName 相当）
 * - validation は backend の責任
 * - frontend では catalog に存在する候補だけを UI 都合で整形する
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