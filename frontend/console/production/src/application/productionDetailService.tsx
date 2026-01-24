// frontend/console/production/src/application/productionDetailService.tsx
import type {
  Production,
  ProductionStatus,
} from "../../../shell/src/shared/types/production";
import {
  ProductionRepositoryHTTP,
  API_BASE as BACKEND_API_BASE,
} from "../infrastructure/http/productionRepositoryHTTP";
import { listProductionsHTTP } from "../infrastructure/query/productionQuery";
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// model module
import {
  listModelVariationsByProductBlueprintId,
  type ModelVariationResponse,
} from "../../../model/src/infrastructure/repository/modelRepositoryHTTP";

/**
 * 詳細表示用型（Production）
 * - createdAt/updatedAt/printedAt は Date として保持する
 */
export type ProductionDetail = Omit<
  Production,
  "createdAt" | "updatedAt" | "printedAt"
> & {
  totalQuantity: number;
  assigneeName?: string;
  productName?: string;
  brandName?: string;

  printedAt: Date | null;
  createdAt: Date | null;
  updatedAt: Date | null;
};

export type ModelVariationSummary = {
  id: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
};

export type ProductionQuantityRow = {
  id: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number | string | null;
  quantity: number;
};

/**
 * ProductBlueprint 詳細用
 * ※ このファイルの要件対象外ですが、こちらも Date に寄せたい場合は同様に toDate を使って置き換え可能です。
 */
export type ProductBlueprintDetail = {
  id: string;

  productName: string;
  companyId: string;
  brandId: string;
  itemType: string;
  fit: string;
  material: string;
  weight: number;

  qualityAssurance: string[];
  productIdTag: string;
  assigneeId: string;

  createdBy?: string | null;
  createdAt: string;
  updatedBy?: string | null;
  updatedAt: string;

  deletedBy?: string | null;
  deletedAt?: string | null;

  expireAt?: string | null;
};

async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("未ログインです");
  return user.getIdToken();
}

// ---------- helpers ----------
const asString = (v: any): string => (typeof v === "string" ? v : "");
const asNonEmptyString = (v: any): string =>
  typeof v === "string" && v.trim() ? v.trim() : "";

/**
 * API から来る日時を Date に正規化する
 * - string (ISO) / number (ms) / Date / Firestore Timestamp っぽいもの を許容
 * - 変換できなければ null
 */
const toDate = (v: any): Date | null => {
  if (!v) return null;

  // already Date
  if (v instanceof Date) {
    return Number.isNaN(v.getTime()) ? null : v;
  }

  // Firestore Timestamp shape (best-effort)
  // { seconds: number, nanoseconds: number } or { _seconds, _nanoseconds }
  const seconds =
    typeof v?.seconds === "number"
      ? v.seconds
      : typeof v?._seconds === "number"
        ? v._seconds
        : null;
  const nanos =
    typeof v?.nanoseconds === "number"
      ? v.nanoseconds
      : typeof v?._nanoseconds === "number"
        ? v._nanoseconds
        : 0;

  if (typeof seconds === "number") {
    const ms = seconds * 1000 + Math.floor((nanos ?? 0) / 1e6);
    const d = new Date(ms);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  // number (ms)
  if (typeof v === "number") {
    const d = new Date(v);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  // string
  if (typeof v === "string") {
    const d = new Date(v);
    return Number.isNaN(d.getTime()) ? null : d;
  }

  return null;
};

/* ---------------------------------------------------------
 * Production 詳細取得
 * --------------------------------------------------------- */
export async function loadProductionDetail(
  productionId: string,
): Promise<ProductionDetail | null> {
  if (!productionId) return null;

  const repo = new ProductionRepositoryHTTP();
  const raw = (await repo.getById(productionId)) as any;
  if (!raw) return null;

  const rawModels = Array.isArray(raw.models)
    ? raw.models
    : Array.isArray(raw.Models)
      ? raw.Models
      : [];

  const totalQuantity = rawModels.reduce(
    (sum: number, m: any) => sum + (m?.quantity ?? m?.Quantity ?? 0),
    0,
  );

  const blueprintId = asNonEmptyString(
    raw.productBlueprintId ?? raw.ProductBlueprintID ?? "",
  );

  const resolvedProductName =
    asNonEmptyString(raw.productName ?? raw.ProductName) || blueprintId;

  let detail: ProductionDetail = {
    ...(raw as Production),

    id: asNonEmptyString(raw.id ?? raw.ID ?? ""),
    productBlueprintId: blueprintId,

    productName: resolvedProductName,
    brandName: asString(raw.brandName ?? raw.BrandName ?? raw.brand ?? raw.Brand ?? ""),

    assigneeId: asString(raw.assigneeId ?? raw.AssigneeID ?? ""),
    assigneeName: asString(raw.assigneeName ?? raw.AssigneeName ?? ""),

    status: (raw.status ?? raw.Status ?? "") as ProductionStatus,

    // ✅ time として保持（Date）
    printedAt: toDate(raw.printedAt ?? raw.PrintedAt ?? null),
    createdAt: toDate(raw.createdAt ?? raw.CreatedAt ?? null),
    updatedAt: toDate(raw.updatedAt ?? raw.UpdatedAt ?? null),

    models: rawModels,
    totalQuantity,
  };

  /* ---------------------------------------------------------
   * 一覧の name 解決ロジック
   * --------------------------------------------------------- */
  try {
    const listItems = await listProductionsHTTP();

    const match = (listItems as any[]).find((item) => {
      const itemId = item.id ?? item.ID ?? "";
      const itemBlueprintId = item.productBlueprintId ?? item.ProductBlueprintID ?? "";
      return (
        itemId === detail.id ||
        (itemBlueprintId && itemBlueprintId === detail.productBlueprintId)
      );
    });

    if (match) {
      const matchProductName =
        match.productName ?? match.ProductName ?? detail.productBlueprintId;

      detail = {
        ...detail,
        productName:
          detail.productName && detail.productName !== detail.productBlueprintId
            ? detail.productName
            : matchProductName,

        brandName: detail.brandName || match.brandName || match.BrandName || "",

        assigneeName:
          detail.assigneeName ||
          match.assigneeName ||
          match.AssigneeName ||
          "",
      };
    }
  } catch (_) {}

  return detail;
}

/* ---------------------------------------------------------
 * ProductBlueprint 詳細取得
 * --------------------------------------------------------- */
export async function loadProductBlueprintDetail(
  productBlueprintId: string,
): Promise<ProductBlueprintDetail | null> {
  const id = productBlueprintId?.trim();
  if (!id) return null;

  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(id);

  const url = `${BACKEND_API_BASE}/product-blueprints/${safeId}`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `ProductBlueprint API error: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  const raw = (await res.json()) as any;

  const qa = raw.qualityAssurance ?? raw.QualityAssurance ?? [];

  const rawTag =
    raw.productIdTag ??
    raw.ProductIdTag ??
    raw.ProductIDTag ??
    null;

  let productIdTag = "";
  if (typeof rawTag === "string") {
    productIdTag = rawTag;
  } else if (rawTag && typeof rawTag === "object") {
    productIdTag = rawTag.Type ?? rawTag.type ?? rawTag.tag ?? "";
  }

  const detail: ProductBlueprintDetail = {
    id: raw.id ?? raw.ID ?? id,

    productName: raw.productName ?? raw.ProductName ?? "",
    companyId: raw.companyId ?? raw.CompanyID ?? "",
    brandId: raw.brandId ?? raw.BrandID ?? "",
    itemType: raw.itemType ?? raw.ItemType ?? "",
    fit: raw.fit ?? raw.Fit ?? "",
    material: raw.material ?? raw.Material ?? "",
    weight: Number(raw.weight ?? raw.Weight ?? 0),

    qualityAssurance: Array.isArray(qa) ? qa : [],
    productIdTag,
    assigneeId: raw.assigneeId ?? raw.AssigneeID ?? "",

    createdBy: raw.createdBy ?? raw.CreatedBy ?? null,
    createdAt: raw.createdAt ?? raw.CreatedAt ?? "",
    updatedBy: raw.updatedBy ?? raw.UpdatedBy ?? null,
    updatedAt: raw.updatedAt ?? raw.UpdatedAt ?? "",

    deletedBy: raw.deletedBy ?? raw.DeletedBy ?? null,
    deletedAt: raw.deletedAt ?? raw.DeletedAt ?? null,

    expireAt: raw.expireAt ?? raw.ExpireAt ?? null,
  };

  return detail;
}

/* ---------------------------------------------------------
 * variations → index 変換
 * --------------------------------------------------------- */
export function buildModelIndexFromVariations(
  variations: ModelVariationResponse[],
): Record<string, ModelVariationSummary> {
  const index: Record<string, ModelVariationSummary> = {};

  variations.forEach((v) => {
    index[v.id] = {
      id: v.id,
      modelNumber: v.modelNumber,
      size: v.size,
      color: v.color?.name ?? "",
      rgb: v.color?.rgb ?? null,
    };
  });

  return index;
}

/* ---------------------------------------------------------
 * productBlueprintId → ModelVariation index
 * --------------------------------------------------------- */
export async function loadModelVariationIndexByProductBlueprintId(
  productBlueprintId: string,
): Promise<Record<string, ModelVariationSummary>> {
  const id = productBlueprintId.trim();
  if (!id) return {};

  const list = await listModelVariationsByProductBlueprintId(id);
  return buildModelIndexFromVariations(list);
}

/* ---------------------------------------------------------
 * モデル別 生産数行を生成
 * --------------------------------------------------------- */
export function buildQuantityRowsFromModels(
  models: any[],
  modelIndex: Record<string, ModelVariationSummary>,
): ProductionQuantityRow[] {
  const safeModels = Array.isArray(models) ? models : [];

  const rows: ProductionQuantityRow[] = safeModels.map((m: any, index) => {
    const id = m.ModelID ?? m.id ?? m.ID ?? `${index}`;

    const quantityRaw = m.Quantity ?? 0;

    const quantity = Number.isFinite(Number(quantityRaw))
      ? Math.max(0, Math.floor(Number(quantityRaw)))
      : 0;

    const meta = id ? modelIndex[id] : undefined;

    const row: ProductionQuantityRow = {
      id,
      modelNumber: meta?.modelNumber ?? "",
      size: meta?.size ?? "",
      color: meta?.color ?? "",
      rgb: meta?.rgb ?? null,
      quantity,
    };

    return row;
  });

  return rows;
}

/* ---------------------------------------------------------
 * Production 更新リクエスト
 * --------------------------------------------------------- */
export async function updateProductionDetail(params: {
  productionId: string;
  rows: ProductionQuantityRow[];
  assigneeId?: string | null;
}): Promise<ProductionDetail | null> {
  const { productionId, rows, assigneeId } = params;
  const id = productionId.trim();
  if (!id) throw new Error("productionId is required");

  const token = await getIdTokenOrThrow();
  const safeId = encodeURIComponent(id);

  const modelsPayload = rows.map((r) => ({
    modelId: r.id,
    quantity: Number.isFinite(Number(r.quantity))
      ? Math.max(0, Math.floor(Number(r.quantity)))
      : 0,
  }));

  const payload: any = {
    assigneeId: assigneeId ?? null,
    models: modelsPayload,
  };

  const res = await fetch(`${BACKEND_API_BASE}/productions/${safeId}`, {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(payload),
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Production update failed: ${res.status} ${res.statusText}${
        body ? ` - ${body}` : ""
      }`,
    );
  }

  return loadProductionDetail(id);
}

/* ---------------------------------------------------------
 * 印刷完了シグナル受信
 *   - Production を printed に更新
 *   - ProductBlueprint の printed 更新は printService 側に委譲
 * --------------------------------------------------------- */
export async function notifyPrintLogCompleted(params: {
  productionId: string;
  logCount: number;
  totalQrCount: number;
  reusedExistingLogs?: boolean;
}): Promise<void> {
  const { productionId } = params;

  const id = productionId.trim();
  if (!id) {
    return;
  }

  const user = auth.currentUser;
  if (!user) {
    return;
  }

  const printedBy = user.uid;

  // ✅ payload は ISO 文字列で送る（API契約として妥当）
  const printedAt = new Date().toISOString();

  try {
    const token = await getIdTokenOrThrow();
    const safeId = encodeURIComponent(id);

    const payload: any = {
      status: "printed" as ProductionStatus,
      printedAt,
      printedBy,
    };

    const res = await fetch(`${BACKEND_API_BASE}/productions/${safeId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    });

    if (!res.ok) {
      const body = await res.text().catch(() => "");
      console.error(
        "[notifyPrintLogCompleted] failed to update production printed status",
        {
          productionId: id,
          status: res.status,
          statusText: res.statusText,
          body,
        },
      );
    }
  } catch (e) {
    console.error(
      "[notifyPrintLogCompleted] unexpected error while updating production printed status",
      {
        productionId: id,
        error: e,
      },
    );
  }
}
