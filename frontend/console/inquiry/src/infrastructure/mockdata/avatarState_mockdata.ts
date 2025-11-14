// frontend/inquiry/src/infrastructure/mockdata/avatarState_mockdata.ts
// ------------------------------------------------------
// Mock data for AvatarState
// Mirrors frontend/shell/src/shared/types/avatarState.ts
// ------------------------------------------------------

import type { AvatarState } from "../../../../shell/src/shared/types/avatarState";

/**
 * Mock AvatarState records representing SNS-like activity states
 * of avatars across the system.
 */
export const AVATAR_STATES: AvatarState[] = [
  {
    id: "state-001",
    avatarId: "avatar-001",
    followerCount: 120,
    followingCount: 56,
    postCount: 34,
    lastActiveAt: "2025-11-09T12:30:00Z",
    updatedAt: "2025-11-09T13:00:00Z",
  },
  {
    id: "state-002",
    avatarId: "avatar-002",
    followerCount: 58,
    followingCount: 75,
    postCount: 18,
    lastActiveAt: "2025-11-09T09:45:00Z",
    updatedAt: "2025-11-09T10:00:00Z",
  },
  {
    id: "state-003",
    avatarId: "avatar-003",
    followerCount: 300,
    followingCount: 210,
    postCount: 102,
    lastActiveAt: "2025-11-08T22:10:00Z",
    updatedAt: "2025-11-09T00:00:00Z",
  },
  {
    id: "state-004",
    avatarId: "avatar-004",
    followerCount: 0,
    followingCount: 0,
    postCount: 0,
    lastActiveAt: "2025-11-07T14:00:00Z",
    updatedAt: "2025-11-08T00:00:00Z",
  },
  {
    id: "state-005",
    avatarId: "avatar-005",
    followerCount: 42,
    followingCount: 33,
    postCount: 15,
    lastActiveAt: "2025-11-09T15:10:00Z",
    updatedAt: "2025-11-09T15:30:00Z",
  },
];

/**
 * Default export
 */
export default AVATAR_STATES;
