// frontend/console/shell/src/shared/types/tokenBlueprint.ts

/**
 * TokenBlueprint共通型。
 *
 * このファイルをFrontendにおけるTokenBlueprint型の
 * 唯一の正規定義とする。
 *
 * Feature配下で同名の型を再定義せず、
 * 必要な型はこのファイルからimportする。
 *
 * Backend BFFのresponseを正とし、
 * response値のnormalize・fallback・validationはここでは行わない。
 *
 * 対象:
 * - TokenBlueprint
 * - ContentFile
 * - ContentType
 *
 * 対象外:
 * - HTTP DTO
 * - Firebase Storage操作
 * - API通信
 * - Reactの状態
 * - ViewModel
 * - Application Service
 */

/* =========================================================
 * Content type
 * =======================================================*/

export type ContentType =
  | "image"
  | "video"
  | "pdf"
  | "document";

/* =========================================================
 * ContentFile
 * =======================================================*/

export interface ContentFile {
  id: string;
  name: string;
  type: ContentType;
  contentType: string;
  url: string;
  objectPath: string;
  isPublic: boolean;
  size: number;
  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
}

/* =========================================================
 * TokenBlueprint
 * =======================================================*/

export interface TokenBlueprint {
  id: string;
  name: string;
  symbol: string;

  brandId: string;
  brandName?: string;

  companyId: string;
  description?: string;

  iconUrl?: string | null;
  iconObjectPath?: string | null;
  iconFileName?: string | null;
  iconContentType?: string | null;
  iconSize?: number | null;

  contentFiles: ContentFile[];

  assigneeId: string;
  assigneeName?: string;

  minted: boolean;

  createdAt?: string;
  createdBy?: string;
  createdByName?: string;

  updatedAt?: string;
  updatedBy?: string;
  updatedByName?: string;

  deletedAt?: string | null;
  deletedBy?: string | null;

  metadataUri?: string;
}