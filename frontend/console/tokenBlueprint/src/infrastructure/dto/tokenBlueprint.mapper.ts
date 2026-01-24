// frontend/console/tokenBlueprint/src/infrastructure/dto/tokenBlueprint.mapper.ts

import type {
  TokenBlueprint,
  ContentFile,
  ContentFileType,
} from "../../domain/entity/tokenBlueprint";
import type {
  TokenBlueprintDTO,
  TokenBlueprintPageResultDTO,
  SignedIconUploadDTO,
  ContentFileDTO,
} from "./tokenBlueprint.dto";

function asObject(raw: any): Record<string, any> {
  return raw && typeof raw === "object" ? (raw as Record<string, any>) : {};
}

function s(v: any): string {
  return v == null ? "" : String(v).trim();
}

function b(v: any): boolean {
  return typeof v === "boolean" ? v : false;
}

function n(v: any): number {
  const x = Number(v);
  return Number.isFinite(x) ? x : 0;
}

/**
 * 時刻フィールドを string(ISO) に寄せる best-effort 変換。
 * - string: そのまま trim
 * - number: Date(ms) とみなして ISO 化
 * - { seconds, nanos } / { _seconds, _nanoseconds } 等: Firestore Timestamp 風として ISO 化
 * - Date: toISOString
 * - 変換不能: ""（空）
 */
function normalizeTimeToISO(v: any): string {
  if (v == null) return "";

  if (typeof v === "string") return v.trim();

  if (v instanceof Date) {
    const t = v.getTime();
    return Number.isNaN(t) ? "" : v.toISOString();
  }

  if (typeof v === "number") {
    const d = new Date(v);
    return Number.isNaN(d.getTime()) ? "" : d.toISOString();
  }

  const o = asObject(v);

  const sec =
    o.seconds != null
      ? Number(o.seconds)
      : o._seconds != null
        ? Number(o._seconds)
        : NaN;

  const nanos =
    o.nanos != null
      ? Number(o.nanos)
      : o.nanoseconds != null
        ? Number(o.nanoseconds)
        : o._nanoseconds != null
          ? Number(o._nanoseconds)
          : NaN;

  if (Number.isFinite(sec)) {
    const ms = sec * 1000 + (Number.isFinite(nanos) ? Math.floor(nanos / 1e6) : 0);
    const d = new Date(ms);
    return Number.isNaN(d.getTime()) ? "" : d.toISOString();
  }

  // fallback: stringify（最終手段）
  try {
    const str = String(v).trim();
    return str === "[object Object]" ? "" : str;
  } catch {
    return "";
  }
}

function normalizeIconUpload(raw: any): SignedIconUploadDTO | undefined {
  const o = asObject(raw);
  const uploadUrl = s(o.uploadUrl);
  const publicUrl = s(o.publicUrl);
  const objectPath = s(o.objectPath);
  if (!uploadUrl || !publicUrl || !objectPath) return undefined;

  return {
    uploadUrl,
    publicUrl,
    objectPath,
    expiresAt: o.expiresAt != null ? String(o.expiresAt) : undefined,
    contentType: o.contentType != null ? s(o.contentType) : undefined,
  };
}

/**
 * DTO(ContentFileDTO) -> domain(shared)(ContentFile)
 *
 * shell/shared の ContentFile が backend(entity.go) と同じフィールド構造
 * （id/name/type/contentType/size/objectPath/visibility/createdAt/createdBy/updatedAt/updatedBy）
 * である前提でマッピングします。
 *
 * 追加要件:
 * - backend が返す contentFiles[].url を落とさず blueprint.contentFiles に保持する
 *   （GCS が private の場合でも、アプリ側で「閲覧用URL（署名URL/プロキシURL）」を載せる運用があるため）
 */
function normalizeContentFiles(raw: any): ContentFile[] {
  const arr = Array.isArray(raw) ? raw : [];
  return arr
    .map((x) => asObject(x))
    .map((o) => {
      const visibility = s(o.visibility) || "private";

      const cf: any = {
        id: s(o.id),
        name: s(o.name),
        type: s(o.type) as ContentFileType,
        contentType: s(o.contentType),
        size: n(o.size),
        objectPath: s(o.objectPath),
        visibility,
      };

      // backend が返す url を保持（存在する場合のみ）
      const url = s(o.url);
      if (url) cf.url = url;

      // optional fields
      const ca = normalizeTimeToISO(o.createdAt);
      const ua = normalizeTimeToISO(o.updatedAt);

      if (ca) cf.createdAt = ca;
      if (o.createdBy != null) cf.createdBy = s(o.createdBy);
      if (ua) cf.updatedAt = ua;
      if (o.updatedBy != null) cf.updatedBy = s(o.updatedBy);

      return cf as ContentFile;
    })
    .filter((f) => s((f as any).id) && s((f as any).objectPath));
}

export function normalizeTokenBlueprint(raw: any): TokenBlueprint {
  const obj = asObject(raw);

  const minted = b(obj.minted);

  const iconUrl =
    obj.iconUrl === null ? null : obj.iconUrl != null ? s(obj.iconUrl) : undefined;

  const contentFiles = normalizeContentFiles(obj.contentFiles);

  const iconUpload = normalizeIconUpload(obj.iconUpload);

  // backend が返す表示名（ベストエフォート）
  const assigneeName = obj.assigneeName != null ? s(obj.assigneeName) : undefined;
  const createdByName = obj.createdByName != null ? s(obj.createdByName) : undefined;
  const updatedByName = obj.updatedByName != null ? s(obj.updatedByName) : undefined;

  // ★ 作成/更新日時（一覧で表示するために必要）
  const createdAt = normalizeTimeToISO(obj.createdAt);
  const updatedAt = normalizeTimeToISO(obj.updatedAt);

  // shared 型を正として返す（未知フィールドは極力載せない/載せても害がない範囲に）
  const out: any = {
    id: s(obj.id),
    name: s(obj.name),
    symbol: s(obj.symbol),

    brandId: s(obj.brandId),

    // companyId は shared 型では必須だが、現実のレスポンスで欠ける可能性もあるため best-effort
    companyId: obj.companyId != null ? s(obj.companyId) : "",

    description: s(obj.description),

    // shared 型では必須だが、現実のレスポンスで欠ける可能性があるため best-effort
    assigneeId: obj.assigneeId != null ? s(obj.assigneeId) : "",

    minted,

    // shared 型では必須
    metadataUri: obj.metadataUri != null ? s(obj.metadataUri) : "",

    contentFiles,

    ...(iconUrl !== undefined ? { iconUrl } : {}),
    ...(iconUpload ? { iconUpload } : {}),
    ...(obj.brandName != null ? { brandName: s(obj.brandName) } : {}),

    // ★ 名前解決結果
    ...(assigneeName !== undefined ? { assigneeName } : {}),
    ...(createdByName !== undefined ? { createdByName } : {}),
    ...(updatedByName !== undefined ? { updatedByName } : {}),

    // ★ 作成/更新情報（画面に出すため）
    ...(createdAt ? { createdAt } : {}),
    ...(obj.createdBy != null ? { createdBy: s(obj.createdBy) } : {}),
    ...(updatedAt ? { updatedAt } : {}),
    ...(obj.updatedBy != null ? { updatedBy: s(obj.updatedBy) } : {}),
  };

  return out as TokenBlueprint;
}

export function normalizePageResult(raw: any): {
  items: TokenBlueprint[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
} {
  const obj = asObject(raw);

  const rawItems = Array.isArray(obj.items) ? obj.items : [];
  const items = rawItems.map((it) => normalizeTokenBlueprint(it));

  return {
    items,
    totalCount: n(obj.totalCount),
    totalPages: n(obj.totalPages),
    page: n(obj.page) || 1,
    perPage: n(obj.perPage),
  };
}
