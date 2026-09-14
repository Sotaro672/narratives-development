// frontend/mall/src/features/howToUse/presentation/components/console/ProductBlueprintGuide.tsx

import enterCategoryFieldVideo from "../../../assets/console/product-blueprint/enter-category-field.mp4";
import enterColorVideo from "../../../assets/console/product-blueprint/enter-color.mp4";
import enterMeasurementVideo from "../../../assets/console/product-blueprint/enter-meaturement.mp4";
import enterModelNumberVideo from "../../../assets/console/product-blueprint/enter-modelNumber.mp4";
import enterShippingPackageVideo from "../../../assets/console/product-blueprint/enter-shipping-package.mp4";
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

      <HowToUseSection title="カテゴリ固有情報の入力（衣類）">
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

      <HowToUseSection title="カラーの登録（衣類）">
        <HowToUseStepList>
          <HowToUseStep>商品のカラーバリエーションを追加してください。</HowToUseStep>
          <HowToUseStep>カラー名と商品の実際の色に対応するカラー情報を入力してください。</HowToUseStep>
          <HowToUseStep>複数のカラーがある場合は、それぞれのカラーバリエーションを登録してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterColorVideo}
          label="商品のカラーバリエーションを追加し、カラー情報を登録する手順"
        />

        <HowToUseNote title="カラーバリエーション">
          商品に複数のカラーがある場合は、それぞれを個別のカラーバリエーションとして登録してください。登録したカラーはサイズとの組み合わせによりモデルを構成します。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="サイズ・採寸の登録（衣類）">
        <HowToUseStepList>
          <HowToUseStep>商品のサイズバリエーションを追加してください。</HowToUseStep>
          <HowToUseStep>登録するサイズを選択してください。</HowToUseStep>
          <HowToUseStep>サイズごとに必要な採寸項目を入力してください。</HowToUseStep>
          <HowToUseStep>複数のサイズがある場合は、それぞれのサイズと採寸情報を登録してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterMeasurementVideo}
          label="商品のサイズバリエーションを追加し、サイズごとの採寸情報を入力する手順"
        />

        <HowToUseNote title="サイズと採寸">
          採寸情報はサイズごとに登録してください。登録したサイズはカラーとの組み合わせによりモデルを構成します。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="型番の登録（衣類）">
        <HowToUseStepList>
          <HowToUseStep>登録したカラーとサイズの組み合わせを確認してください。</HowToUseStep>
          <HowToUseStep>各モデルに対応する型番を入力してください。</HowToUseStep>
          <HowToUseStep>複数のモデルがある場合は、それぞれに固有の型番を設定してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterModelNumberVideo}
          label="カラーとサイズの組み合わせごとに商品の型番を入力する手順"
        />

        <HowToUseNote title="型番">
          型番はカラーとサイズの組み合わせによって作成される各モデルを識別するために使用します。それぞれのモデルに対応する型番を登録してください。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="配送時の梱包情報の登録（衣類）">
        <HowToUseStepList>
          <HowToUseStep>各モデルの配送時の重量を入力してください。</HowToUseStep>
          <HowToUseStep>梱包後の横、縦、高さを入力してください。</HowToUseStep>
          <HowToUseStep>カラーとサイズごとに、実際の配送状態に合わせた梱包情報を確認してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          src={enterShippingPackageVideo}
          label="カラーとサイズごとに配送時の重量と梱包後の横、縦、高さを入力する手順"
        />

        <HowToUseNote title="配送時の梱包情報">
          配送時の重量と梱包寸法は、配送料の計算に使用する情報です。商品単体の寸法ではなく、配送時に梱包した状態の重量と横、縦、高さを登録してください。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="商品設計の保存">
        <HowToUseStepList>
          <HowToUseStep>酒類商品の場合は、容量、型番、配送時の梱包情報を登録してください。</HowToUseStep>
          <HowToUseStep>商品設計の担当者を選択してください。</HowToUseStep>
          <HowToUseStep>すべての入力内容を確認してください。</HowToUseStep>
          <HowToUseStep>画面上部の「保存」をクリックして商品設計を登録してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseNote title="モデルと配送情報">
          アパレル商品のモデルはカラーとサイズの組み合わせによって構成されます。型番と配送時の梱包情報は、それぞれのモデルに対応する内容を登録してください。
        </HowToUseNote>
      </HowToUseSection>
    </>
  );
}