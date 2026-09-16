// frontend/mall/src/features/howToUse/presentation/components/console/MemberInvitationGuide.tsx

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

        <HowToUseVideo storagePath="console/member-invitation/go-to-member-registration.mp4" label="サイドバーから組織、メンバーを選択し、メンバー追加ボタンをクリックしてメンバー登録画面へ移動する手順" />
      </HowToUseSection>

      <HowToUseSection title="メンバー招待手順">
        <HowToUseStepList>
          <HowToUseStep>招待するメンバーのメールアドレスを入力してください。</HowToUseStep>
          <HowToUseStep>メンバーへ割り当てるブランドを選択してください。</HowToUseStep>
          <HowToUseStep>必要な権限を設定してください。</HowToUseStep>
          <HowToUseStep>入力内容を確認し、招待を送信してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/member-invitation/send-invitation-mail.mp4" label="メンバーのメールアドレス、割り当てブランド、権限を設定して招待メールを送信する手順" />

        <HowToUseNote title="招待メール">招待されたメンバーは、届いた招待からConsoleへのアカウント登録を行います。招待するメールアドレスに誤りがないことを確認してください。</HowToUseNote>

        <HowToUseNote title="ブランドの割り当て">メンバーがConsoleで操作できる対象は、招待時に割り当てられたブランドに基づきます。担当するブランドを確認してから招待してください。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="招待の受諾">
        <HowToUseStepList>
          <HowToUseStep>届いた招待からメンバー登録画面を開いてください。</HowToUseStep>
          <HowToUseStep>表示されている会社名と割り当てブランドを確認してください。</HowToUseStep>
          <HowToUseStep>招待されたメールアドレスを入力してください。</HowToUseStep>
          <HowToUseStep>姓、姓（かな）、名、名（かな）を入力してください。</HowToUseStep>
          <HowToUseStep>パスワードと確認用パスワードを入力してください。</HowToUseStep>
          <HowToUseStep>入力内容を確認し、「サインイン」をクリックして登録を完了してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/member-invitation/accept-invitation.mp4" label="届いた招待から会社名と割り当てブランドを確認し、メールアドレス、氏名、パスワードを入力して招待を受諾する手順" />

        <HowToUseNote title="招待内容の確認">登録する前に、表示されている会社名と割り当てブランドが正しいことを確認してください。内容に誤りがある場合は登録を進めず、招待した管理者へ確認してください。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="招待の取り消し">
        <HowToUseStepList>
          <HowToUseStep>メンバー管理画面から、取り消したい招待を選択してください。</HowToUseStep>
          <HowToUseStep>招待内容を確認し、招待を削除してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/member-invitation/delete-invitation.mp4" label="送信済みのメンバー招待を選択し、招待を削除する手順" />

        <HowToUseNote title="招待の削除">誤ったメールアドレスやブランドへ招待を送信した場合は、対象の招待を削除してから正しい内容で再度招待してください。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}