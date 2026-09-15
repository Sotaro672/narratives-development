// frontend/mall/src/features/howToUse/presentation/components/mall/CancelOderGuide.tsx

import cancelOrderVideo from "../../../assets/mall/cancel/cancel-order.mp4";

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function CancelOderGuide() {
  return (
    <>
      <HowToUseSection title="注文のキャンセル">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo src={cancelOrderVideo} label="Mallで注文内容を確認し、注文をキャンセルする手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページへ移動し、取引履歴を開けてください。</HowToUseStep>
            <HowToUseStep>キャンセルする取引を開き、注文をキャンセルボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="発送前の注文のみキャンセル可能です。">発送前の注文はキャンセルできます。発送メールが送信されてからはキャンセルできません。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}