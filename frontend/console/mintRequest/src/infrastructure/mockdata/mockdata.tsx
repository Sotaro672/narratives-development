// frontend/mintRequest/src/infrastructure/mockdata/mockdata.tsx

import type {
  MintRequest,
  MintRequestStatus,
} from "../../../../shell/src/shared/types/mintRequest";

/**
 * ミント申請ステータスのユーティリティ
 */
const asStatus = (s: MintRequestStatus): MintRequestStatus => s;

/**
 * MINT_REQUESTS
 * - frontend/shell/src/shared/types/mintRequest.ts に準拠したモックデータ
 * - 各種 ID はダミー値（UUID/ID文字列）として定義
 * - 日付は ISO8601 形式
 */
export const MINT_REQUESTS: MintRequest[] = [
  {
    id: "mint_req_001",
    tokenBlueprintId: "token_blueprint_001",
    productionId: "production_001",
    mintQuantity: 168,
    burnDate: "2025-12-31",
    status: asStatus("planning"),
    requestedBy: null,
    requestedAt: null,
    mintedAt: null,
    createdAt: "2025-11-05T10:00:00Z",
    createdBy: "member_sato_misaki",
    updatedAt: "2025-11-05T10:00:00Z",
    updatedBy: "member_sato_misaki",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "mint_req_002",
    tokenBlueprintId: "token_blueprint_002",
    productionId: "production_002",
    mintQuantity: 120,
    burnDate: "2025-11-30",
    status: asStatus("requested"),
    requestedBy: "member_takahashi_kenta",
    requestedAt: "2025-11-06T09:30:00Z",
    mintedAt: null,
    createdAt: "2025-11-06T09:00:00Z",
    createdBy: "member_sato_misaki",
    updatedAt: "2025-11-06T09:30:00Z",
    updatedBy: "member_takahashi_kenta",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "mint_req_003",
    tokenBlueprintId: "token_blueprint_003",
    productionId: "production_003",
    mintQuantity: 80,
    burnDate: "2025-10-15",
    status: asStatus("minted"),
    requestedBy: "member_takahashi_kenta",
    requestedAt: "2025-10-01T08:00:00Z",
    mintedAt: "2025-10-02T12:00:00Z",
    createdAt: "2025-09-30T15:00:00Z",
    createdBy: "member_takahashi_kenta",
    updatedAt: "2025-10-02T12:00:00Z",
    updatedBy: "member_takahashi_kenta",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "mint_req_004",
    tokenBlueprintId: "token_blueprint_004",
    productionId: "production_004",
    mintQuantity: 50,
    burnDate: null,
    status: asStatus("requested"),
    requestedBy: "member_sato_misaki",
    requestedAt: "2025-11-07T11:15:00Z",
    mintedAt: null,
    createdAt: "2025-11-07T11:00:00Z",
    createdBy: "member_sato_misaki",
    updatedAt: "2025-11-07T11:15:00Z",
    updatedBy: "member_sato_misaki",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "mint_req_005",
    tokenBlueprintId: "token_blueprint_005",
    productionId: "production_005",
    mintQuantity: 40,
    burnDate: "2025-08-31",
    status: asStatus("planning"),
    requestedBy: null,
    requestedAt: null,
    mintedAt: null,
    createdAt: "2025-11-08T14:20:00Z",
    createdBy: "member_sato_misaki",
    updatedAt: "2025-11-08T14:20:00Z",
    updatedBy: "member_sato_misaki",
    deletedAt: null,
    deletedBy: null,
  },
];
