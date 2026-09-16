// frontend/mall/src/features/howToUse/presentation/components/console/InquiryGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function InquiryGuide() {
  return (
    <>
      <HowToUseSection title="返品申請への対応">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「お問い合わせ」を開いてください。</HowToUseStep>
          <HowToUseStep>対応する返品申請を選択してください。</HowToUseStep>
          <HowToUseStep>返品申請に対する返信を入力してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/inquiry/response-refund.mp4" label="Consoleのお問い合わせ画面から返品申請を確認し、購入者へ回答する手順" />
      </HowToUseSection>

      <HowToUseSection title="返品受領後の手順">
        <HowToUseStepList>
          <HowToUseStep>返品商品を受領した後、画面右上にある返品受領ボタンを押してください。</HowToUseStep>
          <HowToUseStep>返品受領すると自動で返金処理と返金完了のメッセージが購入客へ送信されます。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/inquiry/accept-refund.mp4" label="Consoleのお問い合わせ画面から返品申請を確認し、返品を承認する手順" />
      </HowToUseSection>
    </>
  );
}