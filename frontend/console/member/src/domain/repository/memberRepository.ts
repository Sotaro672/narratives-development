// frontend/console/member/src/domain/repository/memberRepository.ts

import type { Member, MemberPatch } from "../entity/member";
import type {
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../../../shell/src/shared/types/common/common";

/**
 * MemberFilter
 * backend/internal/domain/member/repository_port.go の Filter に対応。
 *
 * - 日付は ISO8601 文字列
 * - undefined は「条件指定なし」
 * - backend 側で CurrentMember の companyId にスコープされる。
 */
export interface MemberFilter {
  /** 名前 / フリガナ / メール等の部分一致検索 */
  searchQuery?: string;

  /**
   * Firebase Auth UID。
   * backend の GET /members?uid=... や内部 filter と対応。
   */
  uid?: string;

  /** 割当ブランドID */
  brandIds?: string[];

  /** 所属企業IDフィルタ */
  companyId?: string;

  /** active / inactive など */
  status?: string;

  /** 作成日時範囲 */
  createdFrom?: string;
  createdTo?: string;

  /** 更新日時範囲 */
  updatedFrom?: string;
  updatedTo?: string;

  /** 権限名 */
  permissions?: string[];
}

/**
 * 呼び出し側で companyId を強制付与するためのユーティリティ。
 */
export function scopedFilterByCompanyId(
  companyId: string,
  base: MemberFilter = {},
): MemberFilter {
  const id = (companyId ?? "").trim();
  if (!id) {
    throw new Error("scopedFilterByCompanyId: companyId is required");
  }

  return { ...base, companyId: id };
}

/**
 * SortOrder
 * backend の SortOrder に対応
 */
export type MemberSortOrder = "asc" | "desc";

/**
 * MemberSort
 * backend の Sort に対応
 */
export interface MemberSort {
  column: string;
  order: MemberSortOrder;
}

/**
 * MemberRepository
 *
 * IMPORTANT:
 * - Member.id は Firestore member document ID
 * - Member.uid は Firebase Auth UID
 * - backend の GET /members/{uid} は Firebase UID 専用
 * - backend の PATCH /members/{docId} は Firestore docId 専用
 *
 * 後方互換用の getById / exists は廃止。
 */
export interface MemberRepository {
  // ===== 取得 =====

  /**
   * Firebase UID で member を取得する。
   *
   * backend:
   * GET /members/{uid}
   */
  getByUid(uid: string): Promise<Member | null>;

  /**
   * Email から member を取得する。
   */
  getByEmail(email: string): Promise<Member | null>;

  // ===== 一覧 =====

  /**
   * 一覧取得。
   */
  list(page: Page, filter?: MemberFilter): Promise<PageResult<Member>>;

  /**
   * カーソルベース一覧取得。
   */
  listByCursor(
    filter: MemberFilter,
    sort: MemberSort,
    cursorPage: CursorPage,
  ): Promise<CursorPageResult<Member>>;

  // ===== 作成 / 更新 / 削除 =====

  /**
   * 作成。
   *
   * 通常の console member 作成では uid / id を request body から送らない。
   * backend 側で招待前 member として uid 空で作成される。
   */
  create(member: Member, opts?: SaveOptions): Promise<Member>;

  /**
   * Firestore member docId による更新。
   *
   * backend:
   * PATCH /members/{docId}
   */
  update(
    docId: string,
    patch: MemberPatch,
    opts?: SaveOptions,
  ): Promise<Member>;

  /**
   * 削除。
   */
  delete(docId: string): Promise<void>;

  // ===== 補助 =====

  /**
   * Firebase UID による存在確認。
   */
  existsByUid(uid: string): Promise<boolean>;

  /**
   * 件数カウント。
   */
  count(filter: MemberFilter): Promise<number>;

  /**
   * Save。
   *
   * 現状 backend API では create のみ対応。
   */
  save(member: Member, opts?: SaveOptions): Promise<Member>;
}