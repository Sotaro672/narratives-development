// frontend/mall/src/features/howToUse/presentation/components/console/MintGuide.tsx

import HowToUseNote from "../common/HowToUseNote";
import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function MintGuide() {
  return (
    <>
      <HowToUseSection title="ミント申請">
        <HowToUseStepList>
          <HowToUseStep>サイドバーの「トークン」「ミント」、該当する生産計画行の順でミント申請画面を開いてください。</HowToUseStep>
          <HowToUseStep>ブランド選択、ミントするトークン設計を選んでください。</HowToUseStep>
          <HowToUseStep>ミントにかかるガス見積を確認後、「ミントを申請」ボタンを押してください。</HowToUseStep>
          <HowToUseStep>ミント中に画面を離れてもミントは継続されます。</HowToUseStep>
        </HowToUseStepList>

        <HowToUseVideo storagePath="console/mint/mint.mp4" label="検品が完了した商品を確認し、ミント申請を実行する手順" />

        <HowToUseNote title="ミント時間">添付動画では17点のミントで9分弱かかりました。ミント途中で画面を離れてもミントは続行されます。</HowToUseNote>

        <HowToUseNote title="ミント後のトークン設計の変更">ミント後のトークン設計ではトークン名とブランド選択を変更できません。トークンアイコンとコンテンツの変更は可能です。</HowToUseNote>
      </HowToUseSection>
    </>
  );
}