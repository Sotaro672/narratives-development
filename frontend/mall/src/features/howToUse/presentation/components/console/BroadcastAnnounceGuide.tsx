// frontend/mall/src/features/howToUse/presentation/components/console/BroadcastAnnounceGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function BroadcastAnnounceGuide() {
  return (
    <>
      <HowToUseSection title="告知の一斉送信">
        <HowToUseStepList>
          <HowToUseStep>サイドバーから「トークン」「告知」の順で告知管理画面を開いてください。</HowToUseStep>
          <HowToUseStep>告知一覧画面の「告知を作成」をクリックしてください。</HowToUseStep>
          <HowToUseStep>告知を送信する対象のトークンを選択してください。</HowToUseStep>
          <HowToUseStep>告知に表示する画像、タイトル、文章を入力してください。</HowToUseStep>
          <HowToUseStep>管理情報に表示される送信対象数を確認してください。</HowToUseStep>
          <HowToUseStep>画面右上の「送信」をクリックしてください。</HowToUseStep>
          <HowToUseStep>画像を設定している場合は転送が完了するまで画面を閉じず、そのままお待ちください。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo
          storagePath="console/announce/broadcast-announce.mp4"
          label="Consoleで告知対象のトークンを選択し、画像、タイトル、文章を入力して所有者へ告知を一斉送信する手順"
        />

        <HowToUseNote title="送信対象">
          告知は、選択したトークンを所有しているアバターを対象に送信されます。
        </HowToUseNote>
      </HowToUseSection>

      <HowToUseSection title="アバターへの見え方">
        <div className="how-to-use-mobile-guide">
          <HowToUseVideo
            storagePath="console/announce/view-announce-at-mall.mp4"
            label="Consoleから送信された告知をMallで開き、画像、タイトル、本文を確認する手順"
            variant="iphone-12-pro"
          />

          <HowToUseStepList>
            <HowToUseStep>ウォレットページヘッダーの告知ボタンに未読通知が来ているので、クリックしてください。</HowToUseStep>
            <HowToUseStep>受信した告知の一覧から確認する告知を選択してください。</HowToUseStep>
          </HowToUseStepList>
        </div>
      </HowToUseSection>
    </>
  );
}