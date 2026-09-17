// frontend/mall/src/features/howToUse/presentation/components/mall/CancelOderGuide.tsx

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
          <HowToUseVideo
            storagePath="mall/cancel/cancel-order.mp4"
            label="Mallで注文内容を確認し、注文をキャンセルする手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページへ移動し、取引履歴を開けてください。</HowToUseStep>
            <HowToUseStep>キャンセルする取引を開き、「注文をキャンセル」ボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="発送前の注文のみキャンセル可能です。">
          発送前の注文はキャンセルできます。発送メールが送信されてからはキャンセルできません。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="フリマ取引のキャンセル">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/cancel/cancel-trade.mp4"
            label="Mallでフリマ取引の内容を確認し、取引をキャンセルする手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページへ移動し、キャンセルする取引履歴を開いてください。</HowToUseStep>
            <HowToUseStep>「取引画面」ボタンを押してください。</HowToUseStep>
            <HowToUseStep>確認画面の「注文をキャンセル」ボタンを押し、出品者へのメッセージを投稿すると注文がキャンセルされます。</HowToUseStep>
            <HowToUseStep>出品者へは注文がキャンセルされた旨のメール通知が送信されます。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="発送前のみキャンセルが可能です。">
          出品者が発送前ならキャンセルが可能です。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}