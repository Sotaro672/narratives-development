// frontend/mall/src/features/howToUse/presentation/components/console/ProductBlueprintGuide.tsx

import goToProductBlueprintRegistrationVideo from "../../../assets/console/product-blueprint/go-to-productBlueprint-registration.mp4";
import selectCategoryVideo from "../../../assets/console/product-blueprint/select-category.mp4";

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ProductBlueprintGuide() {
  return (
    <>
      <HowToUseSection title="商品設計画面の開け方">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから商品設計の管理画面を開いてください。</HowToUseStep>
          <HowToUseStep>商品設計一覧の「商品設計を作成」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={goToProductBlueprintRegistrationVideo}
          label="商品設計一覧を開き、商品設計を作成ボタンから商品設計画面へ移動する手順"
        />
      </HowToUseSection>

      <HowToUseSection title="商品カテゴリの選択">
        <HowToUseStepList>
          <HowToUseStep>商品を販売するブランドを選択してください。</HowToUseStep>
          <HowToUseStep>商品名を入力してください。</HowToUseStep>
          <HowToUseStep>商品カテゴリを選択してください。</HowToUseStep>
          <HowToUseStep>カテゴリを選択すると、そのカテゴリに応じた入力項目が表示されます。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={selectCategoryVideo}
          label="ブランドと商品名を入力し、商品カテゴリを選択してカテゴリに応じた入力項目を表示する手順"
        />

        <HowToUseNote title="商品カテゴリ">
          当解説画面では衣類、トップスを選択した場合で登録手順を解説します。順次登録ができるカテゴリを実装していきます。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="商品設計手順">
        <HowToUseStepList>
          <HowToUseStep>表示されたカテゴリ固有の商品情報を入力してください。</HowToUseStep>
          <HowToUseStep>アパレル商品の場合は、カラー、サイズ、採寸、型番、配送時の梱包情報を登録してください。</HowToUseStep>
          <HowToUseStep>酒類商品の場合は、容量、型番、配送時の梱包情報を登録してください。</HowToUseStep>
          <HowToUseStep>商品設計の担当者を選択してください。</HowToUseStep>
          <HowToUseStep>すべての入力が完了したら、画面上部の「保存」をクリックしてください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote title="型番と配送情報">
          アパレル商品の型番はカラーとサイズの組み合わせごと、酒類商品の型番は容量ごとに設定します。配送に使用する重量と梱包寸法も各型番に対応する情報として登録してください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}