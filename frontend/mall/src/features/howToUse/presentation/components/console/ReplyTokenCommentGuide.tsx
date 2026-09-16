// frontend/mall/src/features/howToUse/presentation/components/console/ReplyTokenCommentGuide.tsx

import HowToUseSection from "../common/HowToUseSection";
import HowToUseStep from "../common/HowToUseStep";
import HowToUseStepList from "../common/HowToUseStepList";
import HowToUseVideo from "../common/HowToUseVideo";

export default function ReplyTokenCommentGuide() {
  return (
    <HowToUseSection title="コメント返信">
      <HowToUseStepList>
        <HowToUseStep>サイドバーから「レビュー」「トークン」を押してトークン設計レビュー画面へ移動してください。</HowToUseStep>
        <HowToUseStep>レビューを確認したトークン行を押してください。</HowToUseStep>
        <HowToUseStep>ブランドアカウントとしてコメント、コメント返信をすることができます。</HowToUseStep>
      </HowToUseStepList>

      <HowToUseVideo
        storagePath="console/comment/reply-token-comment.mp4"
        label="Consoleからトークンコンテンツに投稿されたコメントへ返信する手順"
      />
    </HowToUseSection>
  );
}