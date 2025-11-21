// frontend/console/member/src/application/invitationService.ts

// 認証（IDトークン取得用）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// memberService から API_BASE を利用（同じバックエンドURLを共有）
import { API_BASE } from "./memberListService";

/**
 * メンバー招待メール送信トリガー
 * - POST /members/{memberId}/invitation
 */
export async function sendMemberInvitation(
  memberId: string,
  email: string | null | undefined,
): Promise<void> {
  if (!email) {
    // eslint-disable-next-line no-console
    console.warn(
      "[invitationService.sendMemberInvitation] email が空のため、招待メールは送信しません。",
    );
    return;
  }

  // 認証トークン取得
  const token = await auth.currentUser?.getIdToken();
  if (!token) {
    throw new Error("未認証のため招待メールを送信できません。");
  }

  const inviteUrl = `${API_BASE}/members/${encodeURIComponent(
    memberId,
  )}/invitation`;

  // eslint-disable-next-line no-console
  console.log(
    "[invitationService.sendMemberInvitation] POST (invitation)",
    inviteUrl,
  );

  try {
    const inviteRes = await fetch(inviteUrl, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({}),
    });

    if (!inviteRes.ok) {
      const inviteText = await inviteRes.text().catch(() => "");
      // eslint-disable-next-line no-console
      console.error(
        `[invitationService.sendMemberInvitation] 招待メール送信に失敗しました (status ${inviteRes.status}) ${inviteText}`,
      );
    } else {
      // eslint-disable-next-line no-console
      console.log(
        "[invitationService.sendMemberInvitation] 招待メール送信リクエスト成功",
      );
    }
  } catch (invErr) {
    // eslint-disable-next-line no-console
    console.error(
      "[invitationService.sendMemberInvitation] 招待メール送信中にエラーが発生しました",
      invErr,
    );
  }
}
