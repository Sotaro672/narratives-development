// frontend/mall/src/features/howToUse/presentation/components/console/MemberInvitationGuide.tsx

import goToMemberRegistrationVideo from "../../../assets/console/member-invitation/go-to-member-registration.mp4";

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function MemberInvitationGuide() {
  return (
    <>
      <HowToUseSection title="メンバー登録画面の開け方">
        <HowToUseStepList>
          <HowToUseStep>サイドバーの「組織」をクリックしてください。</HowToUseStep>
          <HowToUseStep>「組織」をクリックした際に表示される「メンバー」をクリックしてください。</HowToUseStep>
          <HowToUseStep>メンバー管理画面右上にある「メンバー追加」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={goToMemberRegistrationVideo}
          label="サイドバーから組織、メンバーを選択し、メンバー追加ボタンをクリックしてメンバー登録画面へ移動する手順"
        />
      </HowToUseSection>

      <HowToUseSection title="メンバー招待手順">
        <HowToUseStepList>
          <HowToUseStep>招待するメンバーのメールアドレスを入力してください。</HowToUseStep>
          <HowToUseStep>メンバーへ割り当てるブランドを選択してください。</HowToUseStep>
          <HowToUseStep>必要な権限を設定してください。</HowToUseStep>
          <HowToUseStep>入力内容を確認し、招待を送信してください。</HowToUseStep>
          <HowToUseStep>招待されたメンバーは、届いた招待から会社名と割り当てブランドを確認し、氏名とパスワードを設定して登録を完了します。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote title="招待メール">
          招待されたメンバーは、招待情報を使用してConsoleへのアカウント登録を行います。招待するメールアドレスに誤りがないことを確認してください。
        </HowToUseNote>

        <HowToUseNote title="ブランドの割り当て">
          メンバーがConsoleで操作できる対象は、招待時に割り当てられたブランドに基づきます。担当するブランドを確認してから招待してください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}