// frontend/company/src/infrastructure/mockdata/mockdata.ts

import type { Company } from "../../../../shell/src/shared/types/company";

/**
 * モックデータ: Company
 * backend/internal/domain/company/entity.go および
 * frontend/shell/src/shared/types/company.ts に準拠。
 *
 * - createdAt, updatedAt, deletedAt は ISO8601 UTC 文字列
 * - deletedAt, deletedBy は null（未削除）
 */
export const COMPANIES: Company[] = [
  {
    id: "company_001",
    name: "LUMINA Fashion",
    admin: "member_001",
    isActive: true,
    createdAt: "2024-01-10T09:00:00Z",
    createdBy: "member_001",
    updatedAt: "2024-06-15T10:30:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "company_002",
    name: "NEXUS Street",
    admin: "member_002",
    isActive: true,
    createdAt: "2024-02-01T08:30:00Z",
    createdBy: "member_002",
    updatedAt: "2024-06-20T14:00:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "company_003",
    name: "Atelier Noir",
    admin: "member_003",
    isActive: false,
    createdAt: "2024-03-05T11:15:00Z",
    createdBy: "member_003",
    updatedAt: "2024-08-01T09:45:00Z",
    updatedBy: "member_001",
    deletedAt: "2024-10-01T00:00:00Z",
    deletedBy: "member_001",
  },
  {
    id: "company_004",
    name: "Europort Japan",
    admin: "member_004",
    isActive: true,
    createdAt: "2024-04-12T10:00:00Z",
    createdBy: "member_004",
    updatedAt: "2024-07-10T08:15:00Z",
    updatedBy: "member_004",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "company_005",
    name: "Narratives Holdings",
    admin: "member_admin",
    isActive: true,
    createdAt: "2024-01-01T00:00:00Z",
    createdBy: "system",
    updatedAt: "2025-01-01T00:00:00Z",
    updatedBy: "system",
    deletedAt: null,
    deletedBy: null,
  },
];

/**
 * 説明:
 * - 各企業は `member_X` によって管理される
 * - isActive=false の場合、削除済みまたは休眠状態を想定
 * - createdBy / updatedBy はメンバーIDまたは system
 */
