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
            storagePath="mall/refund/request-resale-return.mp4"
            label="フリマ取引のチャットから返品理由を入力し、出品者へ返品の同意を求める手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>返品したいフリマ取引のチャット画面を開いてください。</HowToUseStep>
            <HowToUseStep>「返品について相談する」ボタンを押してください。</HowToUseStep>
            <HowToUseStep>返品を希望する理由を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、出品者へ返品の同意を求めてください。</HowToUseStep>
            <HowToUseStep>送信すると返品理由が取引チャットに表示され、出品者が返品に合意するかを選択できる状態になります。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="返品について">
          返品は出品者の同意後に返送手続きへ進みます。出品者が返品に合意するまでは商品を返送しないでください。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection id="resale-return-report" title="運営に報告する">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="mall/refund/report-trade-dispute.mp4"
            label="出品者が返品相談に合意しなかったフリマ取引を運営へ報告する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>出品者が返品相談に合意しなかった場合、取引チャットを確認してください。</HowToUseStep>
            <HowToUseStep>運営への報告画面を開いてください。</HowToUseStep>
            <HowToUseStep>報告内容を確認し、運営への報告に同意してください。</HowToUseStep>
            <HowToUseStep>返品を求める理由や状況を入力してください。</HowToUseStep>
            <HowToUseStep>入力内容を確認し、運営へ報告してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="運営への報告">
          報告内容と取引履歴をもとに運営が状況を確認します。運営へ報告した時点で返品や返金が確定するものではありません。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}