// frontend/tokenContents/src/infrastructure/mockdata/mockdata.ts
import type { GCSTokenContent } from "../../../../shell/src/shared/types/tokenContents";

/**
 * トークンコンテンツ（画像など）モックデータ
 * frontend/shell/src/shared/types/tokenContents.ts を正として構造を統一。
 * 実際には API から取得される想定。
 */
export const MOCK_TOKEN_CONTENTS: GCSTokenContent[] = [
  {
    id: "content_001",
    name: "silk_premium_1.jpg",
    type: "image",
    url: "https://images.pexels.com/photos/373883/pexels-photo-373883.jpeg?auto=compress&cs=tinysrgb&w=800",
    size: 482394,
  },
  {
    id: "content_002",
    name: "silk_premium_2.jpg",
    type: "image",
    url: "https://images.pexels.com/photos/1036856/pexels-photo-1036856.jpeg?auto=compress&cs=tinysrgb&w=800",
    size: 563212,
  },
  {
    id: "content_003",
    name: "silk_premium_3.jpg",
    type: "image",
    url: "https://images.pexels.com/photos/3965545/pexels-photo-3965545.jpeg?auto=compress&cs=tinysrgb&w=800",
    size: 603812,
  },
];
