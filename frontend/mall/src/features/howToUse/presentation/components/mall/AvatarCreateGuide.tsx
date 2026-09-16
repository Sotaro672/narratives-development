// frontend/mall/src/features/howToUse/presentation/components/mall/AvatarCreateGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function AvatarCreateGuide() {
  return (
    <>
      <HowToUseSection title="Mallアカウントの作成">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/avatar/signUp-mall.mp4" label="Mallでアカウントを作成する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>AMOLのMallを開いてください。</HowToUseStep>
            <HowToUseStep>サインインするメールアドレスとパスワードを設定してください。</HowToUseStep>
            <HowToUseStep>利用規約をよく読み、合意にチェックを入れてください。</HowToUseStep>
            <HowToUseStep>認証メール送信ボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="認証メール">認証メールが送られるので、メールに記載されているリンクからサインインしてください。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="アバターの作成">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/avatar/create-avatar.mp4" label="Mallでアバター情報を入力してアバターを作成する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>サインイン後、アバター作成画面を開いてください。</HowToUseStep>
            <HowToUseStep>アバター画像やアバター名などの必要な情報を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、アバターを作成してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}