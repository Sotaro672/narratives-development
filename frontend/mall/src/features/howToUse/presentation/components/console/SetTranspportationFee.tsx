// frontend/mall/src/features/howToUse/presentation/components/console/SetTranspportationFee.tsx

import setTransportationFeeVideo from "../../../assets/console/inventory/set-transportation-fee.mp4";

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function SetTranspportationFee() {
  return (
    <>
      <HowToUseSection title="配送料金体系の設定">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「配送」を開いてください。</HowToUseStep>
          <HowToUseStep>配送料金体系の設定画面を開いてください。</HowToUseStep>
          <HowToUseStep>配送方法と料金を設定してください。</HowToUseStep>
          <HowToUseStep>入力内容を確認し、保存してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={setTransportationFeeVideo}
          label="配送画面から配送料金体系を設定して保存する手順"
        />
      </HowToUseSection>
    </>
  );
}