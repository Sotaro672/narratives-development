// frontend/mall/src/features/howToUse/presentation/components/mall/ShippingAddressRegistrationGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ShippingAddressRegistrationGuide() {
  return (
    <>
      <HowToUseSection title="配送先住所の登録画面を開く">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="mall/shipping-address/go-to-shipping-address-registration.mp4" label="Mallから配送先住所の登録画面を開く手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページへ移動してください。</HowToUseStep>
            <HowToUseStep>画面右上の設定アイコンを押してください。</HowToUseStep>
            <HowToUseStep>配送先情報を選択してください。</HowToUseStep>
            <HowToUseStep>配送先住所を入力し、登録ボタンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}