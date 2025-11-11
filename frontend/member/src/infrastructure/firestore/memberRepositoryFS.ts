// frontend/member/src/infrastructure/firestore/memberRepositoryFS.ts

import {
  collection,
  doc,
  getDoc,
  getDocs,
  query,
  where,
  orderBy,
  limit as fsLimit,
  startAfter,
  setDoc,
  updateDoc,
  deleteDoc,
  type DocumentData,
  type QueryDocumentSnapshot,
  type DocumentSnapshot,
} from "firebase/firestore";

import { firestore } from "../../adapter/outbound/firestoreClient";

import type {
  MemberRepository,
  MemberFilter,
  MemberSort,
} from "../../domain/repository/memberRepository";
import type { Member, MemberPatch } from "../../domain/entity/member";
import type {
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../../../shell/src/shared/types/common/common";
import { DEFAULT_PAGE_LIMIT } from "../../../../shell/src/shared/types/common/common";

/**
 * Firestore implementation of MemberRepository.
 *
 * 実運用では backend API 経由が本命で、これは POC / 開発用の簡易実装想定。
 */
export class MemberRepositoryFS implements MemberRepository {
  private readonly col = collection(firestore, "members");

  // ======================
  // CRUD / List
  // ======================

  async getById(id: string): Promise<Member | null> {
    if (!id) return null;
    const snap = await getDoc(doc(this.col, id));
    if (!snap.exists()) return null;
    return this.docToDomain(snap);
  }

  async list(page: Page, filter?: MemberFilter): Promise<PageResult<Member>> {
    // Page: { limit, offset }
    const perPage =
      page && typeof page.limit === "number" && page.limit > 0
        ? page.limit
        : DEFAULT_PAGE_LIMIT;
    const offset =
      page && typeof page.offset === "number" && page.offset >= 0
        ? page.offset
        : 0;

    // Firestore クエリ（代表的な条件のみ反映）
    let q = query(this.col, orderBy("createdAt", "desc"));

    if (filter?.roleIds && filter.roleIds.length === 1) {
      q = query(q, where("role", "==", filter.roleIds[0]));
    }
    if (filter?.companyId) {
      q = query(q, where("companyId", "==", filter.companyId));
    }

    // offset 対応のため offset+limit 件取得してクライアント側で slice
    const fetchSize = offset + perPage;
    const snap = await getDocs(query(q, fsLimit(fetchSize)));

    let items = snap.docs.map((d) => this.docToDomain(d));

    // searchQuery, permissions, brandIds などはクライアント側フィルタ
    if (filter) {
      items = this.applyPostFilter(items, filter);
    }

    const pagedItems = items.slice(offset, offset + perPage);

    const resultPage: Page = {
      limit: perPage,
      offset,
    };

    const result: PageResult<Member> = {
      items: pagedItems,
      totalCount: items.length,
      page: resultPage,
    };

    return result;
  }

  async create(member: Member, _opts?: SaveOptions): Promise<Member> {
    const id = member.id || crypto.randomUUID();
    const nowIso = new Date().toISOString();

    const data: Member = {
      ...member,
      id,
      createdAt: member.createdAt || nowIso,
      updatedAt: member.updatedAt ?? nowIso,
    };

    await setDoc(doc(this.col, id), data);
    return data;
  }

  async update(
    id: string,
    patch: MemberPatch,
    _opts?: SaveOptions
  ): Promise<Member> {
    const current = await this.getById(id);
    if (!current) {
      throw new Error("member not found");
    }

    const merged = {
      ...current,
      ...this.applyPatch(current, patch),
      updatedAt: new Date().toISOString(),
    } as Member;

    await updateDoc(doc(this.col, id), merged as any);
    return merged;
  }

  async delete(id: string): Promise<void> {
    if (!id) return;
    await deleteDoc(doc(this.col, id));
  }

  // ======================
  // Extra APIs
  // ======================

  async listByCursor(
    filter: MemberFilter,
    sort: MemberSort,
    cursorPage: CursorPage
  ): Promise<CursorPageResult<Member>> {
    const limit =
      cursorPage && cursorPage.limit > 0
        ? cursorPage.limit
        : DEFAULT_PAGE_LIMIT;

    const orderField =
      sort?.column === "updatedAt"
        ? "updatedAt"
        : sort?.column === "name" || sort?.column === "email"
        ? "createdAt" // name/email の専用 index が無ければ createdAt で代用
        : "createdAt";
    const orderDir = sort?.order === "asc" ? "asc" : "desc";

    let q = query(this.col, orderBy(orderField, orderDir));

    if (filter?.roleIds && filter.roleIds.length === 1) {
      q = query(q, where("role", "==", filter.roleIds[0]));
    }

    // cursorPage.cursor は前回の nextCursor (docId) を前提
    if (cursorPage?.cursor) {
      const cursorSnap = await getDoc(doc(this.col, cursorPage.cursor));
      if (cursorSnap.exists()) {
        q = query(q, startAfter(cursorSnap));
      }
    }

    const snap = await getDocs(query(q, fsLimit(limit + 1)));

    let docs = snap.docs;
    let hasNext = false;
    if (docs.length > limit) {
      hasNext = true;
      docs = docs.slice(0, limit);
    }

    let items = docs.map((d) => this.docToDomain(d));
    if (filter) {
      items = this.applyPostFilter(items, filter);
    }

    const last = docs[docs.length - 1];
    const nextCursor = hasNext && last ? last.id : null;

    const result: CursorPageResult<Member> = {
      items,
      nextCursor,
      prevCursor: undefined, // 今回は next のみサポートする簡易実装
      hasNext,
      hasPrev: Boolean(cursorPage?.cursor), // cursor があれば「前がある」扱い（簡易）
    };

    return result;
  }

  async getByEmail(email: string): Promise<Member | null> {
    const normalized = email.trim().toLowerCase();
    if (!normalized) return null;

    const q = query(this.col, where("email", "==", normalized), fsLimit(1));
    const snap = await getDocs(q);
    if (snap.empty) return null;
    return this.docToDomain(snap.docs[0]);
  }

  async exists(id: string): Promise<boolean> {
    if (!id) return false;
    const snap = await getDoc(doc(this.col, id));
    return snap.exists();
  }

  async count(filter: MemberFilter): Promise<number> {
    let q = query(this.col);
    if (filter?.roleIds && filter.roleIds.length === 1) {
      q = query(q, where("role", "==", filter.roleIds[0]));
    }
    const snap = await getDocs(q);
    let items = snap.docs.map((d) => this.docToDomain(d));
    items = this.applyPostFilter(items, filter);
    return items.length;
  }

  async save(member: Member, opts?: SaveOptions): Promise<Member> {
    const exists = member.id ? await this.exists(member.id) : false;
    if (!exists) {
      if (opts?.mode === "update" || opts?.ifExists) {
        throw new Error("member does not exist");
      }
      return this.create(member, opts);
    }
    if (opts?.mode === "create" || opts?.ifNotExists) {
      throw new Error("member already exists");
    }
    return this.update(member.id, member as MemberPatch, opts);
  }

  async reset(): Promise<void> {
    const snap = await getDocs(this.col);
    await Promise.all(snap.docs.map((d) => deleteDoc(d.ref)));
  }

  // ======================
  // Helpers
  // ======================

  private docToDomain(
    snap:
      | QueryDocumentSnapshot<DocumentData>
      | DocumentSnapshot<DocumentData>
  ): Member {
    const data = snap.data() || {};
    const id = snap.id;

    const member: Member = {
      id,
      firstName: (data.firstName ?? "").trim() || undefined,
      lastName: (data.lastName ?? "").trim() || undefined,
      firstNameKana: (data.firstNameKana ?? "").trim() || undefined,
      lastNameKana: (data.lastNameKana ?? "").trim() || undefined,
      email: (data.email ?? "").trim() || undefined,
      role: data.role,
      permissions: Array.isArray(data.permissions)
        ? data.permissions.map((p: string) => String(p))
        : [],
      assignedBrands: Array.isArray(data.assignedBrands)
        ? data.assignedBrands.map((b: string) => String(b))
        : undefined,
      createdAt: data.createdAt ?? new Date().toISOString(),
      updatedAt:
        typeof data.updatedAt === "string"
          ? data.updatedAt
          : data.updatedAt ?? null,
      updatedBy:
        typeof data.updatedBy === "string"
          ? data.updatedBy
          : data.updatedBy ?? null,
      deletedAt:
        typeof data.deletedAt === "string"
          ? data.deletedAt
          : data.deletedAt ?? null,
      deletedBy:
        typeof data.deletedBy === "string"
          ? data.deletedBy
          : data.deletedBy ?? null,
    };

    return member;
  }

  private applyPatch(current: Member, patch: MemberPatch): Partial<Member> {
    const next: Partial<Member> = {};

    if ("firstName" in patch) next.firstName = patch.firstName ?? undefined;
    if ("lastName" in patch) next.lastName = patch.lastName ?? undefined;
    if ("firstNameKana" in patch)
      next.firstNameKana = patch.firstNameKana ?? undefined;
    if ("lastNameKana" in patch)
      next.lastNameKana = patch.lastNameKana ?? undefined;
    if ("email" in patch) next.email = patch.email ?? undefined;
    if ("role" in patch && patch.role) next.role = patch.role;
    if ("permissions" in patch)
      next.permissions = patch.permissions ?? current.permissions;
    if ("assignedBrands" in patch)
      next.assignedBrands = patch.assignedBrands ?? current.assignedBrands;

    if ("createdAt" in patch)
      next.createdAt = patch.createdAt ?? current.createdAt;
    if ("updatedAt" in patch)
      next.updatedAt = patch.updatedAt ?? current.updatedAt ?? null;
    if ("updatedBy" in patch)
      next.updatedBy = patch.updatedBy ?? current.updatedBy ?? null;
    if ("deletedAt" in patch)
      next.deletedAt = patch.deletedAt ?? current.deletedAt ?? null;
    if ("deletedBy" in patch)
      next.deletedBy = patch.deletedBy ?? current.deletedBy ?? null;

    return next;
  }

  /** Firestore で表現しきれない filter をクライアント側で適用 */
  private applyPostFilter(items: Member[], filter?: MemberFilter): Member[] {
    if (!filter) return items;
    let result = items;

    if (filter.searchQuery) {
      const q = filter.searchQuery.trim().toLowerCase();
      if (q) {
        result = result.filter((m) => {
          const fullName = `${m.lastName ?? ""}${m.firstName ?? ""}`
            .toLowerCase();
          const email = (m.email ?? "").toLowerCase();
          return (
            fullName.includes(q) ||
            email.includes(q) ||
            m.id.toLowerCase().includes(q)
          );
        });
      }
    }

    if (filter.permissions && filter.permissions.length > 0) {
      const wanted = filter.permissions.map((p) => p.toLowerCase());
      result = result.filter((m) =>
        wanted.every((w) =>
          m.permissions.some((p) => p.toLowerCase() === w),
        ),
      );
    }

    if (filter.brandIds && filter.brandIds.length > 0) {
      const set = new Set(filter.brandIds);
      result = result.filter((m) =>
        (m.assignedBrands ?? []).some((b) => set.has(b)),
      );
    }

    return result;
  }
}
