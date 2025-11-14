// frontend/listImage/src/infrastructure/mockdata/mockdata.ts

import type { ListImage } from "../../../../shell/src/shared/types/listImage";

/**
 * モックデータ: ListImage
 * backend/internal/domain/listImage/entity.go および
 * frontend/shell/src/shared/types/listImage.ts に準拠。
 *
 * - URL は GCS の公開URLを模倣
 * - displayOrder は 0 始まり
 * - createdAt / updatedAt / deletedAt は ISO8601 UTC 文字列
 */
export const LIST_IMAGES: ListImage[] = [
  {
    id: "list_image_001",
    listId: "list_001",
    url: "https://storage.googleapis.com/narratives_development_list_image/list_001/image1.webp",
    fileName: "image1.webp",
    size: 204800,
    displayOrder: 0,
    createdAt: "2024-03-01T09:00:00Z",
    createdBy: "member_001",
    updatedAt: "2024-03-02T10:00:00Z",
    updatedBy: "member_001",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "list_image_002",
    listId: "list_001",
    url: "https://storage.googleapis.com/narratives_development_list_image/list_001/image2.webp",
    fileName: "image2.webp",
    size: 198752,
    displayOrder: 1,
    createdAt: "2024-03-01T09:10:00Z",
    createdBy: "member_001",
    updatedAt: "2024-03-02T10:10:00Z",
    updatedBy: "member_001",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "list_image_003",
    listId: "list_002",
    url: "https://storage.googleapis.com/narratives_development_list_image/list_002/main.webp",
    fileName: "main.webp",
    size: 175000,
    displayOrder: 0,
    createdAt: "2024-03-10T11:00:00Z",
    createdBy: "member_002",
    updatedAt: "2024-03-12T12:00:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "list_image_004",
    listId: "list_002",
    url: "https://storage.googleapis.com/narratives_development_list_image/list_002/detail1.webp",
    fileName: "detail1.webp",
    size: 220300,
    displayOrder: 1,
    createdAt: "2024-03-10T11:15:00Z",
    createdBy: "member_002",
    updatedAt: "2024-03-12T12:10:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "list_image_005",
    listId: "list_003",
    url: "https://storage.googleapis.com/narratives_development_list_image/list_003/cover.webp",
    fileName: "cover.webp",
    size: 310000,
    displayOrder: 0,
    createdAt: "2024-03-20T08:30:00Z",
    createdBy: "member_003",
    updatedAt: null,
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,
  },
];
