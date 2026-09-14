// frontend/mall/src/features/howToUse/presentation/components/console/InspectionGuide.tsx

import goToInspectorVideo from "../../../assets/console/inspection/go-to-inspector.mp4";

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function InspectionGuide() {
  return (
    <>
      <HowToUseSection title="検品スキャナーの開き方">
        <HowToUseStepList>
          <HowToUseStep>AMOLのログイン画面を開いてください。</HowToUseStep>
          <HowToUseStep>サービス選択画面から「検品スキャナーにログイン」をクリックしてください。</HowToUseStep>
          <HowToUseStep>検品スキャナーでログインしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={goToInspectorVideo}
          label="AMOLのサービス選択画面から検品スキャナーを開く手順"
          caption="検品スキャナーへの移動"
          variant="iphone-12-pro"
        />
      </HowToUseSection>

      <HowToUseSection title="検品手順">
        <HowToUseStepList>
          <HowToUseStep>検品スキャナーで商品のQRコードをスキャンしてください。</HowToUseStep>
          <HowToUseStep>表示された商品設計とモデル情報を確認し、実際の商品と一致していることを確認してください。</HowToUseStep>
          <HowToUseStep>検品結果として「合格」「不合格」「未製造」のいずれかを選択してください。</HowToUseStep>
          <HowToUseStep>続けて別の商品を検品する場合は「検品を続ける」を押してスキャン画面へ戻り、次の商品のQRコードをスキャンしてください。</HowToUseStep>
          <HowToUseStep>対象となる商品の検品がすべて完了したら「検品を完了する」を押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote title="検品結果">
          「合格」は商品が登録内容と一致し問題がない場合、「不合格」は商品に問題がある場合、「未製造」は生産予定に含まれているものの実際には製造されなかった場合に選択してください。
        </HowToUseNote>

        <HowToUseNote title="検品スキャナー">
          検品スキャナーは商品のQRコードをカメラで読み取って使用するモバイル専用アプリです。QRコードを読み取れる端末から操作してください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}