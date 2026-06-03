// frontend/console/member/src/application/invitationService.ts

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// memberService から API_BASE を利用（同じバックエンドURLを共有）
import { API_BASE } from "./memberListService";

type SendInvitationRequest = {
  memberId: string;
};

/**
 * メンバー招待メール送信トリガー
 * - POST /invitations
 */
export async function sendMemberInvitation(
  memberId: string,
  email: string | null | undefined,
): Promise<void> {
  const normalizedMemberId = memberId.trim();

  if (!normalizedMemberId) {
    throw new Error("memberId が空のため招待メールを送信できません。");
  }

  if (!email) {
    // eslint-disable-next-line no-console
    console.warn(
      "[invitationService.sendMemberInvitation] email が空のため、招待メールは送信しません。",
    );
    return;
  }

  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("未認証のため招待メールを送信できません。");
  }

  const inviteUrl = `${API_BASE}/invitations`;

  const payload: SendInvitationRequest = {
    memberId: normalizedMemberId,
  };

  // eslint-disable-next-line no-console
  console.log(
    "[invitationService.sendMemberInvitation] POST /invitations",
    inviteUrl,
    payload,
  );

  const inviteRes = await fetch(inviteUrl, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!inviteRes.ok) {
    const inviteText = await inviteRes.text().catch(() => "");
    throw new Error(
      `招待メール送信に失敗しました。status=${inviteRes.status} body=${inviteText}`,
    );
  }

  // eslint-disable-next-line no-console
  console.log(
    "[invitationService.sendMemberInvitation] 招待メール送信リクエスト成功",
  );
}