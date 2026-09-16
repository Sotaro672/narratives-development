// frontend/mall/src/features/howToUse/presentation/components/console/ListGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ListGuide() {
  return (
    <>
      <HowToUseSection title="在庫の保管場所と配送料金を設定">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「商品」「在庫」を開いてください。</HowToUseStep>
          <HowToUseStep>出品する在庫行を選択してください。</HowToUseStep>
          <HowToUseStep>商品の在庫保管場所を選択してください。</HowToUseStep>
          <HowToUseStep>商品に適用する配送料金体系を選択してください。</HowToUseStep>
          <HowToUseStep>保存すると出品ボタンが表示されるので、押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/list/set-location-and-transportation.mp4" label="出品する商品に在庫保管場所と配送料金体系を設定する手順" />
      </HowToUseSection>

      <HowToUseSection title="出品作成">
        <HowToUseStepList>
          <HowToUseStep>商品画像と販売価格を入力してください。</HowToUseStep>
          <HowToUseStep>画面右上の作成ボタンを押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/list/create-list.mp4" label="商品の出品情報を入力し、購入者Mallへの出品を作成する手順" />
      </HowToUseSection>

      <HowToUseSection title="出品停止">
        <HowToUseStepList>
          <HowToUseStep>出品一覧から停止する出品行を押してください。</HowToUseStep>
          <HowToUseStep>出品詳細画面右上の保留ボタンを押してください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/list/stop-listing.mp4" label="出品中の商品を停止し、購入者Mallでの販売を停止する手順" />
      </HowToUseSection>
    </>
  );
}