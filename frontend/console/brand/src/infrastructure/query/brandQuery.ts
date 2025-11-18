// frontend/console/brand/src/infrastructure/query/brandQuery.ts
import {
  collection,
  doc,
  getDoc,
  getDocs,
  addDoc,
  setDoc,
  updateDoc,
  deleteDoc,
  query,
  where,
  serverTimestamp,
  Timestamp,
  DocumentData,
  QueryConstraint,
} from "firebase/firestore";

import { getFirestoreClient } from "../../adapter/outbound/firestoreClient";
import type { Brand, BrandPatch } from "../../domain/entity/brand";

/** Firestore collection name */
const COL = "brands";

/** util: TS Brand <- Firestore doc */
function fromDoc(id: string, data: DocumentData): Brand {
  const ts = (v?: any): string | undefined => {
    if (!v) return undefined;
    if (v instanceof Timestamp) return v.toDate().toISOString();
    // ISO 文字列が入っているケースも許容
    if (typeof v === "string") return v;
    return undefined;
  };

  return {
    id,
    companyId: (data.companyId ?? "").trim(),
    name: (data.name ?? "").trim(),
    description: data.description ?? null,
    websiteUrl: data.websiteUrl ?? null,
    isActive: Boolean(data.isActive),
    managerId: data.managerId ?? null,
    walletAddress: (data.walletAddress ?? "").trim(),

    createdAt: ts(data.createdAt) ?? new Date(0).toISOString(),
    createdBy: data.createdBy ?? null,
    updatedAt: ts(data.updatedAt) ?? null,
    updatedBy: data.updatedBy ?? null,
    deletedAt: ts(data.deletedAt) ?? null,
    deletedBy: data.deletedBy ?? null,
  };
}

/** util: Firestore doc <- TS Brand / BrandPatch */
function toDoc(input: Partial<Brand> | BrandPatch): Record<string, any> {
  const trimOrNull = (v: any) =>
    v == null ? null : String(v).trim() === "" ? null : String(v).trim();

  const out: Record<string, any> = {};

  if ("companyId" in input) out.companyId = trimOrNull((input as any).companyId);
  if ("name" in input) out.name = trimOrNull((input as any).name);
  if ("description" in input) out.description = (input as any).description ?? null;
  if ("websiteUrl" in input) out.websiteUrl = (input as any).websiteUrl ?? null;
  if ("isActive" in input) out.isActive = (input as any).isActive;
  if ("managerId" in input) out.managerId = (input as any).managerId ?? null;
  if ("walletAddress" in input) out.walletAddress = trimOrNull((input as any).walletAddress);

  if ("createdBy" in input) out.createdBy = (input as any).createdBy ?? null;
  if ("updatedBy" in input) out.updatedBy = (input as any).updatedBy ?? null;
  if ("deletedBy" in input) out.deletedBy = (input as any).deletedBy ?? null;

  if ("createdAt" in input && (input as any).createdAt) {
    out.createdAt =
      typeof (input as any).createdAt === "string"
        ? new Date((input as any).createdAt)
        : (input as any).createdAt;
  }
  if ("updatedAt" in input && (input as any).updatedAt) {
    out.updatedAt =
      typeof (input as any).updatedAt === "string"
        ? new Date((input as any).updatedAt)
        : (input as any).updatedAt;
  }
  if ("deletedAt" in input && (input as any).deletedAt) {
    out.deletedAt =
      typeof (input as any).deletedAt === "string"
        ? new Date((input as any).deletedAt)
        : (input as any).deletedAt;
  }

  return out;
}

/* ============================================================================
 * Create
 *  - バックエンドを経由せず、そのまま Firestore に反映します
 *  - walletAddress はバックエンドで自動付与していた想定ですが、ここでは空文字のまま保持します
 * ========================================================================== */
export async function createBrandDirect(payload: {
  companyId: string;
  name: string;
  description?: string | null;
  websiteUrl?: string | null;
  managerId: string | null;
}): Promise<Brand> {
  const db = getFirestoreClient();

  const base = {
    companyId: payload.companyId.trim(),
    name: payload.name.trim(),
    description: payload.description ?? null,
    websiteUrl: payload.websiteUrl ?? null,
    isActive: true, // 仕様: 作成時は常に true
    managerId: payload.managerId ?? null,
    walletAddress: "",

    createdAt: serverTimestamp(),
    createdBy: null,
    updatedAt: serverTimestamp(),
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
  };

  const ref = await addDoc(collection(db, COL), base);
  const snap = await getDoc(ref);
  return fromDoc(ref.id, snap.data() || base);
}

/* ============================================================================
 * Get
 * ========================================================================== */
export async function getBrandByIdDirect(id: string): Promise<Brand | null> {
  const db = getFirestoreClient();
  const ref = doc(db, COL, id.trim());
  const snap = await getDoc(ref);
  if (!snap.exists()) return null;
  return fromDoc(snap.id, snap.data()!);
}

/* ============================================================================
 * List (シンプル)
 *  - フィルタ: companyId のみ（orderBy なし） → 複合インデックス不要
 *  - さらに並べ替えや複数条件を付けたい場合は Firebase コンソールで
 *    要求された複合インデックスを作成してください
 * ========================================================================== */
export async function listBrandsDirect(options?: {
  companyId?: string;
}): Promise<Brand[]> {
  const db = getFirestoreClient();
  const constraints: QueryConstraint[] = [];

  if (options?.companyId) {
    constraints.push(where("companyId", "==", options.companyId.trim()));
  }

  const qRef = constraints.length
    ? query(collection(db, COL), ...constraints)
    : query(collection(db, COL));

  const snap = await getDocs(qRef);
  return snap.docs.map((d) => fromDoc(d.id, d.data()));
}

/* ============================================================================
 * Update (部分更新)
 * ========================================================================== */
export async function updateBrandDirect(
  id: string,
  patch: BrandPatch,
): Promise<Brand> {
  const db = getFirestoreClient();
  const ref = doc(db, COL, id.trim());

  const data = toDoc(patch);
  data.updatedAt = serverTimestamp();

  // setDoc(merge) でも良いが、ここは updateDoc を使用
  await updateDoc(ref, data as any);

  const snap = await getDoc(ref);
  return fromDoc(ref.id, snap.data()!);
}

/* ============================================================================
 * Delete (完全削除)
 *  - 仕様に合わせて論理削除にしたい場合は、deletedAt/ deletedBy を更新してください
 * ========================================================================== */
export async function deleteBrandDirect(id: string): Promise<void> {
  const db = getFirestoreClient();
  await deleteDoc(doc(db, COL, id.trim()));
}

/* ============================================================================
 * Save (Upsert)
 *  - id が存在しない場合は新規作成、ある場合は置換（merge）
 * ========================================================================== */
export async function saveBrandDirect(brand: Brand): Promise<Brand> {
  const db = getFirestoreClient();

  if (!brand.id || brand.id.trim() === "") {
    // create
    return createBrandDirect({
      companyId: brand.companyId,
      name: brand.name,
      description: brand.description ?? null,
      websiteUrl: brand.websiteUrl ?? null,
      managerId: brand.managerId ?? null,
    });
  }

  const ref = doc(db, COL, brand.id.trim());
  const data = {
    ...toDoc(brand),
    updatedAt: serverTimestamp(),
  };

  await setDoc(ref, data, { merge: true });
  const snap = await getDoc(ref);
  return fromDoc(ref.id, snap.data()!);
}
