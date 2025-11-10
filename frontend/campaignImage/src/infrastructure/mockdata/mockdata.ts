// frontend/campaignImage/src/infrastructure/mockdata/mockdata.ts

import type { CampaignImage } from "../../../../shell/src/shared/types/campaignImage";

/**
 * モックデータ: CampaignImage
 * backend/internal/domain/campaignImage/entity.go および
 * frontend/shell/src/shared/types/campaignImage.ts に準拠。
 *
 * - imageUrl は公開可能な GCS もしくは pexels 画像URL
 * - width / height / fileSize / mimeType は任意（実際の画像メタデータ想定）
 */
export const CAMPAIGN_IMAGES: CampaignImage[] = [
  {
    id: "cmp_img_001",
    campaignId: "cmp_001",
    imageUrl:
      "https://images.pexels.com/photos/6214476/pexels-photo-6214476.jpeg?auto=compress&cs=tinysrgb&w=600",
    width: 1200,
    height: 800,
    fileSize: 340000,
    mimeType: "image/jpeg",
  },
  {
    id: "cmp_img_002",
    campaignId: "cmp_001",
    imageUrl:
      "https://images.pexels.com/photos/7679650/pexels-photo-7679650.jpeg?auto=compress&cs=tinysrgb&w=600",
    width: 1080,
    height: 720,
    fileSize: 420000,
    mimeType: "image/jpeg",
  },
  {
    id: "cmp_img_003",
    campaignId: "cmp_002",
    imageUrl:
      "https://images.pexels.com/photos/8437005/pexels-photo-8437005.jpeg?auto=compress&cs=tinysrgb&w=600",
    width: 1920,
    height: 1080,
    fileSize: 820000,
    mimeType: "image/png",
  },
  {
    id: "cmp_img_004",
    campaignId: "cmp_002",
    imageUrl:
      "https://images.pexels.com/photos/8454341/pexels-photo-8454341.jpeg?auto=compress&cs=tinysrgb&w=600",
    width: 1280,
    height: 720,
    fileSize: 610000,
    mimeType: "image/webp",
  },
  {
    id: "cmp_img_005",
    campaignId: "cmp_003",
    imageUrl:
      "https://images.pexels.com/photos/7619637/pexels-photo-7619637.jpeg?auto=compress&cs=tinysrgb&w=600",
    width: 1600,
    height: 900,
    fileSize: 520000,
    mimeType: "image/jpeg",
  },
];

/**
 * 説明:
 * - 各 CampaignImage は特定のキャンペーン (campaignId) に紐づく
 * - fileSize は bytes 単位で記述
 * - mimeType は allowed list（image/jpeg, image/png, image/webp, image/gif）に準拠
 * - GCS 経由の場合の例:
 *   "https://storage.googleapis.com/narratives_development_campaign_image/{campaignId}/{filename}.jpg"
 */
