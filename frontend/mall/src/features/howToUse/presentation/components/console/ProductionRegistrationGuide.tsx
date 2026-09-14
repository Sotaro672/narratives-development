// frontend/mall/src/features/howToUse/presentation/components/console/ProductionRegistrationGuide.tsx

import enterProductionQuantityVideo from "../../../assets/console/production-registration/enter-production-quantity.mp4";
import goToProductionRegistrationVideo from "../../../assets/console/production-registration/go-to-production-registration.mp4";

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

        <HowToUseVideo
          src={goToProductionRegistrationVideo}
          label="商品生産一覧を開き、生産計画を作成ボタンから生産計画作成画面へ移動する手順"
        />
      </HowToUseSection>

      <HowToUseSection title="生産数の入力">
        <HowToUseStepList>
          <HowToUseStep>生産計画の担当者を選択してください。</HowToUseStep>
          <HowToUseStep>生産する商品のブランドを選択してください。</HowToUseStep>
          <HowToUseStep>商品設計一覧から、生産する商品を選択してください。</HowToUseStep>
          <HowToUseStep>「モデル別 生産数一覧」で、各モデルの生産数を入力してください。</HowToUseStep>
          <HowToUseStep>すべての入力が完了したら、画面上部の「保存」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterProductionQuantityVideo}
          label="ブランドと商品設計を選択し、モデル別生産数一覧で各モデルの生産数を入力する手順"
        />

        <HowToUseNote title="商品設計">
          生産計画を作成するには、あらかじめ対象商品の商品設計を登録しておく必要があります。ブランドを選択すると、そのブランドの商品設計から生産対象を選択できます。
        </HowToUseNote>

        <HowToUseNote title="モデル別生産数">
          生産数は商品全体ではなく、商品設計に登録されているモデルごとに入力します。カラーやサイズ、容量などのバリエーションを確認し、それぞれの生産予定数を設定してください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}