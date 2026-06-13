// frontend/console/admin/src/infrastructure/repository/adminRepositoryHTTP.ts

import type { Member } from "../../../../member/src/domain/entity/member";

export type MemberWithDisplayName = Member & {
  displayName?: string | null;
};