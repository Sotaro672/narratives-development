// frontend/mall/src/features/howToUse/presentation/components/console/InspectionGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function InspectionGuide() {
  return (
    <>
      <HowToUseSection title="検品スキャナーの開き方">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="console/inspection/go-to-inspector.mp4" label="AMOLのサービス選択画面から検品スキャナーを開く手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>AMOLのログイン画面を開いてください。</HowToUseStep>
            <HowToUseStep>サービス選択画面から「検品スキャナーにログイン」をクリックしてください。</HowToUseStep>
            <HowToUseStep>検品スキャナーでログインしてください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>

      <HowToUseSection title="検品手順">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo storagePath="console/inspection/enter-inspection.mp4" label="商品のQRコードを読み取り、検品結果を入力する手順" variant="iphone-12-pro" />

          <HowToUseStepList>
            <HowToUseStep>検品スキャナーで商品のQRコードをスキャンしてください。</HowToUseStep>
            <HowToUseStep>検品結果として「合格」「不合格」「未製造」のいずれかを選択してください。</HowToUseStep>
            <HowToUseStep>続けて別の商品を検品する場合はヘッダーの戻るアイコンを押してください。</HowToUseStep>
          </HowToUseStepList>
        </div>

        <HowToUseNote title="検品はネガティブ登録制">この検品はネガティブ登録制です。「不合格」「未製造」の商品IDをミント対象から弾くことを目的とします。</HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="検品の完了">
        <HowToUseStepList>
          <HowToUseStep>コンソール画面に戻り、トークン、ミント、対象生産計画行を押してミント申請画面を開いてください。</HowToUseStep>
          <HowToUseStep>検品内容を確認し、「検品完了」ボタンをクリックしてください。</HowToUseStep>
          <HowToUseStep>検品完了時点で未検査の商品は、すべて「合格」として登録されます。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/inspection/complete-inspection.mp4" label="PCのミント申請画面から検品を完了する手順" />

        <HowToUseNote title="未検査商品の扱い">「検品完了」を実行すると、個別に「不合格」「未製造」として登録されていない未検査の商品はすべて「合格」として扱われます。検品完了前に、不合格品や未製造品の登録漏れがないことを確認してください。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}