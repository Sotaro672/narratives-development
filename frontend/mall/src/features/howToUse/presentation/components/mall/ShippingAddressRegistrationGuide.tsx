// frontend/mall/src/features/howToUse/presentation/components/mall/ShippingAddressRegistrationGuide.tsx

import goToShippingAddressRegistrationVideo from "../../../assets/mall/shippingAddress/go-to-shippingAddress-registration.mp4";

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ShippingAddressRegistrationGuide() {
  return (
    <>
      <HowToUseSection title="配送先住所の登録画面を開く">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            src={goToShippingAddressRegistrationVideo}
            label="Mallから配送先住所の登録画面を開く手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページへ移動してください。</HowToUseStep>
            <HowToUseStep>画面右上の設定アイコンを押してください。</HowToUseStep>
            <HowToUseStep>配送先情報を選択してください。</HowToUseStep>
            <HowToUseStep>配送先住所を入力し、登録ボタンを教えてください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}