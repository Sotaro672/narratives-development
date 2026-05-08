// frontend\console\company\src\domain\repository\companyRepository.ts
// frontend/console/company/src/domain/repository/companyRepository.ts

import type { Company } from "../entity/company";

/**
 * Patch（部分更新）
 * - undefined のフィールドは「更新しない」
 * - 値を null にしたい場合は、必要に応じて別途ルールを定義
 */
export interface CompanyPatch {
  name?: string;
  admin?: string;
  isActive?: boolean;

  updatedAt?: string; // ISO8601
  updatedBy?: string;
  deletedAt?: string | null; // 論理削除
  deletedBy?: string | null;
}

/**
 * 検索用フィルタ
 * - Go の Filter 構造体に対応
 */
export interface CompanyFilter {
  /** フリーテキスト（name 等に対して部分一致など、実装側で解釈） */
  searchQuery?: string;

  /** 絞り込み条件 */
  ids?: string[];
  name?: string; // 完全一致 or LIKE は実装側で解釈
  admin?: string;
  isActive?: boolean;

  createdBy?: string;
  updatedBy?: string;
  deletedBy?: string;

  /** 日付レンジ（ISO8601 文字列） */
  createdFrom?: string;
  createdTo?: string;
  updatedFrom?: string;
  updatedTo?: string;
  deletedFrom?: string;
  deletedTo?: string;

  /**
   * 論理削除の tri-state
   * - undefined: 全件
   * - true: 削除済のみ
   * - false: 未削除のみ
   */
  deleted?: boolean;
}

/** ソート順 */
export type SortOrder = "asc" | "desc";

/** ソート指定（Go の common.Sort に相当） */
export interface Sort {
  /** ソート対象フィールド名（例: "createdAt", "name" など） */
  field: string;
  order: SortOrder;
}

/** ページネーション指定（ページ番号 + サイズ） */
export interface Page {
  page: number; // 1-based
  size: number;
}

/** ページネーション結果（ページ番号ベース） */
export interface PageResult<T> {
  items: T[];
  totalCount: number;
  page: number;
  size: number;
}

/** カーソルベースのページング入力 */
export interface CursorPage {
  cursor?: string | null;
  size: number;
}

/** カーソルベースのページング結果 */
export interface CursorPageResult<T> {
  items: T[];
  nextCursor?: string | null;
}

/** SaveOptions の簡易版（必要に応じて拡張） */
export interface SaveOptions {
  /** 存在しない場合に作成を許可するか（Upsert 用） */
  upsert?: boolean;
}

/**
 * CompanyRepository ポート（契約）
 * - Go の Repository インターフェースに対応
 * - すべて Promise ベースで非同期
 */
export interface CompanyRepository {
  /** 一覧取得（ページ番号ベース） */
  list(
    filter: CompanyFilter,
    sort: Sort,
    page: Page
  ): Promise<PageResult<Company>>;

  /** 一覧取得（カーソルベース） */
  listByCursor(
    filter: CompanyFilter,
    sort: Sort,
    cpage: CursorPage
  ): Promise<CursorPageResult<Company>>;

  /** 単一取得 */
  getById(id: string): Promise<Company | null>;

  /** 存在チェック */
  exists(id: string): Promise<boolean>;

  /** 件数取得 */
  count(filter: CompanyFilter): Promise<number>;

  /** 作成 */
  create(c: Company): Promise<Company>;

  /** 部分更新（Patch） */
  update(id: string, patch: CompanyPatch): Promise<Company>;

  /** 論理削除 or 物理削除（実装側ルールに依存） */
  delete(id: string): Promise<void>;

  /** Save / Upsert 等の拡張オペレーション */
  save(c: Company, opts?: SaveOptions): Promise<Company>;
}
