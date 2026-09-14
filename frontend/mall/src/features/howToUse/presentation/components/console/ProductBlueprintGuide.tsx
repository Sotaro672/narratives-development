// frontend/mall/src/features/howToUse/presentation/components/console/ProductBlueprintGuide.tsx

import enterCategoryFieldVideo from "../../../assets/console/product-blueprint/enter-category-field.mp4";
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
          商品設計で入力する項目は選択した商品カテゴリによって変わります。商品に該当するカテゴリを選択してから、表示された項目を入力してください。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="カテゴリ固有情報の入力">
        <HowToUseStepList>
          <HowToUseStep>商品の重さを入力してください。</HowToUseStep>
          <HowToUseStep>商品のフィットを選択してください。</HowToUseStep>
          <HowToUseStep>商品の素材を入力してください。</HowToUseStep>
          <HowToUseStep>品質保証に関する情報を入力してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterCategoryFieldVideo}
          label="商品の重さ、フィット、素材、品質保証の情報を入力する手順"
        />

        <HowToUseNote title="カテゴリ固有情報">
          重さ、フィット、素材、品質保証などの商品情報は、選択した商品カテゴリに応じて表示されます。実際の商品情報に合わせて入力してください。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="商品設計手順">
        <HowToUseStepList>
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