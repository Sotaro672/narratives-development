// frontend/inquiry/src/infrastructure/mockdata/avatarIcon_mockdata.ts
// ─────────────────────────────────────────────
// Mock data for AvatarIcon
// Mirrors frontend/shell/src/shared/types/avatarIcon.ts
// ─────────────────────────────────────────────

import type { AvatarIcon } from "../../../../shell/src/shared/types/avatarIcon";

/**
 * Sample avatar icon mock data.
 * Represents images stored in GCS for testing or development environments.
 */
export const AVATAR_ICONS: AvatarIcon[] = [
  {
    id: "icon-001",
    avatarId: "avatar-001",
    url: "https://storage.googleapis.com/narratives_development_avatar_icon/avatar-001/icon1.png",
    fileName: "icon1.png",
    size: 152034,
  },
  {
    id: "icon-002",
    avatarId: "avatar-002",
    url: "https://storage.googleapis.com/narratives_development_avatar_icon/avatar-002/icon2.webp",
    fileName: "icon2.webp",
    size: 98432,
  },
  {
    id: "icon-003",
    avatarId: "avatar-003",
    url: "https://storage.googleapis.com/narratives_development_avatar_icon/avatar-003/icon3.jpg",
    fileName: "icon3.jpg",
    size: 203498,
  },
  {
    id: "icon-004",
    avatarId: "avatar-004",
    url: "https://storage.googleapis.com/narratives_development_avatar_icon/avatar-004/icon4.gif",
    fileName: "icon4.gif",
    size: 145028,
  },
  {
    id: "icon-005",
    avatarId: "avatar-005",
    url: "https://storage.googleapis.com/narratives_development_avatar_icon/avatar-005/icon5.jpeg",
    fileName: "icon5.jpeg",
    size: 178542,
  },
];

/**
 * Default export (useful for local imports without named imports)
 */
export default AVATAR_ICONS;
