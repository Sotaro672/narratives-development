//frontend\console\company\src\infrastructure\firestore\companyRepositoryFS.ts
import {
  collection,
  doc,
  getDoc,
  getDocs,
  query,
  where,
  orderBy,
  limit,
  startAfter,
  addDoc,
  updateDoc,
  serverTimestamp,
  type QueryConstraint,
  type DocumentSnapshot,
  type DocumentData,
} from "firebase/firestore";

import { db } from "../../adapter/outbound/firestoreClient";

import type { Company } from "../../domain/entity/company";
import type {
  CompanyRepository,
  CompanyFilter,
  CompanyPatch,
  Sort,
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
} from "../../domain/repository/companyRepository";

/**
 * Firestore 実装の CompanyRepository
 * - コレクション: "companies"
 * - ドキュメント: { id, name, admin, isActive, createdAt, updatedAt, ... }
 */
export class CompanyRepositoryFS implements CompanyRepository {
  private colRef = collection(db, "companies");

  // =========================================================
  // 共通ヘルパ
  // =========================================================

  /** DocumentSnapshot → Company への変換（id は doc.id を優先） */
  private fromSnap(snap: DocumentSnapshot<DocumentData>): Company {
    const data = snap.data() || {};
    return {
      id: snap.id,
      ...(data as Omit<Company, "id">),
    };
  }

  /** Filter / Sort から Firestore の QueryConstraint 配列を生成 */
  private buildQueryConstraints(
    filter?: CompanyFilter,
    sort?: Sort,
  ): QueryConstraint[] {
    const constraints: QueryConstraint[] = [];

    if (filter) {
      // name / admin / isActive など完全一致系
      if (filter.name) constraints.push(where("name", "==", filter.name));
      if (filter.admin) constraints.push(where("admin", "==", filter.admin));
      if (typeof filter.isActive === "boolean") {
        constraints.push(where("isActive", "==", filter.isActive));
      }

      // 論理削除フラグ（tri-state）
      if (typeof filter.deleted === "boolean") {
        if (filter.deleted) {
          // 削除済のみ
          constraints.push(where("deletedAt", "!=", null));
        } else {
          // 未削除のみ
          constraints.push(where("deletedAt", "==", null));
        }
      }

      // ※ createdFrom/To など日付レンジは必要に応じて追加実装（インデックス注意）
    }

    // ソート
    if (sort) {
      constraints.push(orderBy(sort.field, sort.order));
    } else {
      constraints.push(orderBy("createdAt", "desc"));
    }

    return constraints;
  }

  // =========================================================
  // list（ページ番号ベース）
  // =========================================================

  async list(
    filter: CompanyFilter,
    sort: Sort,
    page: Page,
  ): Promise<PageResult<Company>> {
    const constraints = this.buildQueryConstraints(filter, sort);

    // page.page（0-based 前提）ぶんスキップするため、多めに取得して slice
    const take = (page.page + 1) * page.size;
    constraints.push(limit(take));

    const q = query(this.colRef, ...constraints);
    const snap = await getDocs(q);

    const allItems = snap.docs.map((d) => this.fromSnap(d));

    const start = page.page * page.size;
    const items = allItems.slice(start, start + page.size);

    return {
      items,
      totalCount: allItems.length,
      page: page.page,
      size: page.size,
    };
  }

  // =========================================================
  // listByCursor（カーソルベース）
  // =========================================================

  async listByCursor(
    filter: CompanyFilter,
    sort: Sort,
    cpage: CursorPage,
  ): Promise<CursorPageResult<Company>> {
    const constraints = this.buildQueryConstraints(filter, sort);

    // カーソルがあればその doc 以降を取得
    if (cpage.cursor) {
      const cursorDocRef = doc(this.colRef, cpage.cursor);
      const cursorSnap = await getDoc(cursorDocRef);
      if (cursorSnap.exists()) {
        constraints.push(startAfter(cursorSnap));
      }
    }

    constraints.push(limit(cpage.size));

    const q = query(this.colRef, ...constraints);
    const snap = await getDocs(q);

    const items = snap.docs.map((d) => this.fromSnap(d));

    const lastDoc = snap.docs[snap.docs.length - 1];
    const nextCursor =
      snap.docs.length === cpage.size && lastDoc ? lastDoc.id : null;

    return {
      items,
      nextCursor,
    };
  }

  // =========================================================
  // Get / Exists / Count
  // =========================================================

  async getById(id: string): Promise<Company | null> {
    const ref = doc(this.colRef, id);
    const snap = await getDoc(ref);
    if (!snap.exists()) return null;
    return this.fromSnap(snap);
  }

  async exists(id: string): Promise<boolean> {
    const ref = doc(this.colRef, id);
    const snap = await getDoc(ref);
    return snap.exists();
  }

  async count(filter: CompanyFilter): Promise<number> {
    const constraints = this.buildQueryConstraints(filter);
    const q = query(this.colRef, ...constraints);
    const snap = await getDocs(q);
    return snap.docs.length;
  }

  // =========================================================
  // Create / Update / Delete / Save
  // =========================================================

  async create(c: Company): Promise<Company> {
    const now = serverTimestamp();

    // いったん id を空で入れて、作成後に docRef.id を id として返す
    const docRef = await addDoc(this.colRef, {
      ...c,
      id: c.id ?? "",
      createdAt: now,
      updatedAt: now,
    });

    // Firestore 側の id フィールドも合わせたい場合はここで更新してもよい
    await updateDoc(docRef, { id: docRef.id });

    return {
      ...c,
      id: docRef.id,
    };
  }

  async update(id: string, patch: CompanyPatch): Promise<Company> {
    const ref = doc(this.colRef, id);

    // --- ここがポイント ---
    // ドメインの CompanyPatch 型（updatedAt: string）とは切り離し、
    // Firestore に渡す更新オブジェクトを別 DTO として組み立てる
    const patchForFs: Record<string, any> = {
      updatedAt: serverTimestamp(),
    };

    if (typeof patch.name !== "undefined") patchForFs.name = patch.name;
    if (typeof patch.admin !== "undefined") patchForFs.admin = patch.admin;
    if (typeof patch.isActive !== "undefined") patchForFs.isActive = patch.isActive;
    if (typeof patch.updatedBy !== "undefined") patchForFs.updatedBy = patch.updatedBy;
    if (typeof patch.deletedAt !== "undefined") patchForFs.deletedAt = patch.deletedAt;
    if (typeof patch.deletedBy !== "undefined") patchForFs.deletedBy = patch.deletedBy;

    await updateDoc(ref, patchForFs);

    const snap = await getDoc(ref);
    if (!snap.exists()) {
      // 更新後に存在しないケースは想定外なのでエラーにしておく
      throw new Error("company: not found after update");
    }
    return this.fromSnap(snap);
  }

  async delete(id: string): Promise<void> {
    const ref = doc(this.colRef, id);

    // 論理削除として deletedAt を更新
    await updateDoc(ref, {
      deletedAt: serverTimestamp(),
    });
  }

  async save(c: Company): Promise<Company> {
    if (c.id) {
      return this.update(c.id, c as CompanyPatch);
    }
    return this.create(c);
  }
}

export default CompanyRepositoryFS;
