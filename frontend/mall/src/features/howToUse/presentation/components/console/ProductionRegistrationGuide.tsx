// frontend/mall/src/features/howToUse/presentation/components/console/ProductionRegistrationGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ProductionRegistrationGuide() {
  return (
    <>
      <HowToUseSection title="生産計画作成画面の開け方">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「商品生産」を開いてください。</HowToUseStep>
          <HowToUseStep>商品生産一覧の「生産計画を作成」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/production/go-to-production-registration.mp4" label="商品生産一覧を開き、生産計画を作成ボタンから生産計画作成画面へ移動する手順" />
      </HowToUseSection>

      <HowToUseSection title="生産数の入力">
        <HowToUseStepList>
          <HowToUseStep>生産計画の担当者を選択してください。</HowToUseStep>
          <HowToUseStep>生産する商品のブランドを選択してください。</HowToUseStep>
          <HowToUseStep>商品設計一覧から、生産する商品を選択してください。</HowToUseStep>
          <HowToUseStep>「モデル別 生産数一覧」で、各モデルの生産数を入力してください。</HowToUseStep>
          <HowToUseStep>すべての入力が完了したら、画面上部の「保存」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/production/enter-production-quantity.mp4" label="ブランドと商品設計を選択し、モデル別生産数一覧で各モデルの生産数を入力する手順" />
      </HowToUseSection>

      <HowToUseSection title="商品の印刷">
        <HowToUseStepList>
          <HowToUseStep>保存した生産計画を開いてください。</HowToUseStep>
          <HowToUseStep>画面上部の印刷ボタンをクリックしてください。</HowToUseStep>
          <HowToUseStep>表示された内容を確認し、商品の印刷を実行してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote>一度印刷した商品設計は編集できなくなります。商品名やモデル情報など、商品設計の内容に誤りがないことを確認してから印刷を実行してください。</HowToUseNote>

        <HowToUseVideo storagePath="console/production/print-product.mp4" label="保存した生産計画から商品の印刷を実行する手順" />
      </HowToUseSection>

      <HowToUseSection title="印刷結果の確認">
        <HowToUseStepList>
          <HowToUseStep>商品の印刷後、生産計画の画面を開いてください。</HowToUseStep>
          <HowToUseStep>印刷結果はQRコードとCSV出力が可能です。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/production/view-print-result.mp4" label="商品の印刷後に印刷結果を確認し、QRコード出力とCSV出力を確認する手順" />
      </HowToUseSection>
    </>
  );
}