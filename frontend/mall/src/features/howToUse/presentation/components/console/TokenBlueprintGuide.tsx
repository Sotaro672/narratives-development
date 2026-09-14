// frontend/mall/src/features/howToUse/presentation/components/console/TokenBlueprintGuide.tsx

import goToTokenBlueprintRegistrationVideo from "../../../assets/console/token-blueprint/go-to-tokenBlueprint-registration.mp4";

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function TokenBlueprintGuide() {
  return (
    <>
      <HowToUseSection title="トークン設計画面の開け方">
        <HowToUseStepList>
          <HowToUseStep>サイドバーからトークン設計の管理画面を開いてください。</HowToUseStep>
          <HowToUseStep>トークン設計一覧の「トークン設計を作成」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={goToTokenBlueprintRegistrationVideo}
          label="トークン設計一覧を開き、トークン設計を作成ボタンからトークン設計画面へ移動する手順"
          caption="トークン設計画面への移動"
        />
      </HowToUseSection>

      <HowToUseSection title="トークン設計手順">
        <HowToUseStepList>
          <HowToUseStep>トークンを発行するブランドを選択してください。</HowToUseStep>
          <HowToUseStep>トークン名とシンボルを入力してください。</HowToUseStep>
          <HowToUseStep>トークンの説明を入力してください。</HowToUseStep>
          <HowToUseStep>トークンのアイコン画像をアップロードしてください。</HowToUseStep>
          <HowToUseStep>必要に応じて、トークン所有者へ提供する画像やファイルなどのコンテンツを追加してください。</HowToUseStep>
          <HowToUseStep>トークン設計の担当者を選択してください。</HowToUseStep>
          <HowToUseStep>すべての入力が完了したら、画面上部の「保存」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote title="トークンコンテンツ">
          トークンには画像やファイルなどのコンテンツを登録できます。登録したコンテンツはトークンに紐づく情報として管理されます。
        </HowToUseNote>

        <HowToUseNote title="保存処理">
          保存時にはトークン設計の作成と、アイコン画像やコンテンツのアップロードが実行されます。処理中は進捗画面が表示されるため、完了するまで画面を閉じないでください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}