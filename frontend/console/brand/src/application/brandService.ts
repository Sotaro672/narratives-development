// frontend/console/brand/src/application/brandService.ts

import { brandRepositoryHTTP } from "../infrastructure/http/brandRepositoryHTTP";

export type BrandRow = {
  id: string;
  name: string;
  isActive: boolean;
  managerId?: string | null;
  registeredAt: string; // YYYY/MM/DD
};

type Brand = {
  id: string;
  name?: string | null;
  isActive?: boolean | null;
  managerId?: string | null;
  createdAt?: string | null;
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

// ===========================
// ブランド一覧取得（owner 削除版）
// ===========================
export async function listBrands(companyId: string): Promise<BrandRow[]> {
  if (!companyId) return [];

  // ① brandRepository からブランド取得
  const result = await brandRepositoryHTTP.list({
    filter: { companyId },
    sort: { column: "created_at", order: "desc" },
    page: 1,
    perPage: 200,
  });

  const brands = (result.items ?? []) as Brand[];

  // ② Brand → BrandRow（owner 要素は削除）
  const rows: BrandRow[] = brands.map((b) => ({
    id: b.id,
    name: String(b.name ?? "").trim(),
    isActive: !!b.isActive,
    managerId: (b.managerId ?? "").trim() || null,
    registeredAt: formatDateYmd(b.createdAt),
  }));

  return rows;
}
