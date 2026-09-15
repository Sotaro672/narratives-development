// frontend/mall/src/features/howToUse/presentation/components/console/InquiryGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

const responseRefundVideo =
  "https://firebasestorage.googleapis.com/v0/b/narratives-development-26c2d.firebasestorage.app/o/how-to-use%2Fconsole%2Finquiry%2Fresponse-refund.mp4?alt=media";

export default function InquiryGuide() {
  return (
    <>
      <HowToUseSection title="返品申請への対応">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「お問い合わせ」を開いてください。</HowToUseStep>
          <HowToUseStep>対応する返品申請を選択してください。</HowToUseStep>
          <HowToUseStep>返品申請に対する返信を入力してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo src={responseRefundVideo} label="Consoleのお問い合わせ画面から返品申請を確認し、購入者へ回答する手順" />
      </HowToUseSection>
    </>
  );
}