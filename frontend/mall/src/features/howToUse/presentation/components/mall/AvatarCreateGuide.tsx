// frontend/mall/src/features/howToUse/presentation/components/mall/AvatarCreateGuide.tsx

import signUpMallVideo from "../../../assets/mall/avatar/signUp-mall.mp4";

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function AvatarCreateGuide() {
  return (
    <>
      <HowToUseSection title="Mallアカウントの作成">
        <HowToUseStepList>
          <HowToUseStep>AMOLのMallを開いてください。</HowToUseStep>
          <HowToUseStep>アカウント作成画面を開いてください。</HowToUseStep>
          <HowToUseStep>必要な情報を入力してアカウントを作成してください。</HowToUseStep>
          <HowToUseStep>アカウント作成後、アバター情報を登録してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo src={signUpMallVideo} label="Mallでアカウントを作成し、アバターを登録する手順" />
      </HowToUseSection>
    </>
  );
}