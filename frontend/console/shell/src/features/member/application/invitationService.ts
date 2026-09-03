// frontend/console/shell/src/features/member/application/invitationService.ts

import { auth } from "../../../auth/infrastructure/config/firebaseClient";
import { API_BASE } from "./memberListService";

type SendInvitationRequest = {
  memberId: string;
};

/**
 * メンバー招待メール送信トリガー
 * POST /invitations
 */
export async function sendMemberInvitation(
  memberId: string,
  email: string | null | undefined,
): Promise<void> {
  if (!memberId) {
    throw new Error("memberId が空のため招待メールを送信できません。");
  }

  if (!email) {
    return;
  }

  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("未認証のため招待メールを送信できません。");
  }

  const response = await fetch(`${API_BASE}/invitations`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      memberId,
    } satisfies SendInvitationRequest),
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(
      `招待メール送信に失敗しました。status=${response.status} body=${body}`,
    );
  }
}

/**
 * 招待中Memberの招待を取消し、Memberを削除する。
 * DELETE /invitations/{memberId}
 *
 * Backend側で招待tokenを失効させた後、招待中Memberを削除する。
 * Firebase UID設定済みのMemberは取消対象にならない。
 */
export async function cancelMemberInvitation(memberId: string): Promise<void> {
  if (!memberId) {
    throw new Error("memberId が空のため招待を取り消せません。");
  }

  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("未認証のため招待を取り消せません。");
  }

  const response = await fetch(
    `${API_BASE}/invitations/${encodeURIComponent(memberId)}`,
    {
      method: "DELETE",
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  );

  if (!response.ok) {
    const body = await response.text().catch(() => "");

    if (response.status === 404) {
      throw new Error("招待中のメンバーが見つかりません。");
    }

    if (response.status === 412) {
      throw new Error("このメンバーは既に招待を完了しているため削除できません。");
    }

    throw new Error(
      `招待取消に失敗しました。status=${response.status} body=${body}`,
    );
  }
}