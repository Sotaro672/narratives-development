// frontend/mall/src/features/howToUse/presentation/components/mall/ResaleReturnGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ResaleReturnGuide() {
  return (
    <>
      <HowToUseSection id="resale-return-consultation" title="返品について相談する">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/resale-return/request-resale-return.mp4"
            label="フリマ取引のチャットから返品理由を入力し、出品者へ返品の同意を求める手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>返品したいフリマ取引のチャット画面を開いてください。</HowToUseStep>
            <HowToUseStep>「返品について相談する」ボタンを押してください。</HowToUseStep>
            <HowToUseStep>返品を希望する理由を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、出品者へ返品の同意を求めてください。</HowToUseStep>
            <HowToUseStep>送信すると、返品理由が取引チャットに表示され、出品者が返品に合意するかを選択できる状態になります。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="返品について">
          返品は、出品者の同意後に返送手続きへ進みます。出品者が返品に合意するまでは商品を返送しないでください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}