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

      // ★ 追加: backend が返す url を保持する（存在する場合のみ）
      // - 署名URL/プロキシURL/公開URLなど、表示に必要なURLが入るケースがある
      const url = s(o.url);
      if (url) cf.url = url;

      // optional fields
      if (o.createdAt != null) cf.createdAt = String(o.createdAt);
      if (o.createdBy != null) cf.createdBy = s(o.createdBy);
      if (o.updatedAt != null) cf.updatedAt = String(o.updatedAt);
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

  // shared 型を正として返す（未知フィールドは極力載せない/載せても害がない範囲に）
  const out: any = {
    id: s(obj.id),
    name: s(obj.name),
    symbol: s(obj.symbol),

    brandId: s(obj.brandId),
    // companyId は domain 側で optional 拡張しているのでここで入れてOK
    companyId: obj.companyId != null ? s(obj.companyId) : undefined,

    description: s(obj.description),
    assigneeId: obj.assigneeId != null ? s(obj.assigneeId) : undefined,

    minted,
    metadataUri: obj.metadataUri != null ? s(obj.metadataUri) : undefined,

    contentFiles,

    ...(iconUrl !== undefined ? { iconUrl } : {}),
    ...(iconUpload ? { iconUpload } : {}),
    ...(obj.brandName != null ? { brandName: s(obj.brandName) } : {}),
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
