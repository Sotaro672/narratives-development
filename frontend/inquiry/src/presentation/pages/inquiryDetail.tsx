// frontend/inquiry/src/pages/inquiryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

export default function InquiryDetail() {
  const navigate = useNavigate();
  const { inquiryId } = useParams<{ inquiryId: string }>();

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ─────────────────────────────────────────
  // モックデータ（実装時は API 等から取得）
  // ─────────────────────────────────────────
  const [title] = React.useState("シルクブラウスのサイズ交換について");
  const [body] = React.useState(
    "LUMINA Fashion のプレミアムシルクブラウスを購入しましたが、サイズが少し大きいため交換を希望しています。交換可否と手続き方法を教えてください。"
  );
  const [user] = React.useState("Creator Alice");
  const [status] = React.useState<"対応中" | "未対応">("未対応");
  const [type] = React.useState<"商品説明" | "交換">("交換");
  const [owner, setOwner] = React.useState("佐藤 美咲");
  const [inquiredAt] = React.useState("2024/9/20");
  const [answeredAt] = React.useState("2024/9/20");
  const [creator] = React.useState("山田 太郎");
  const [createdAt] = React.useState("2024/9/19");

  return (
    <PageStyle
      layout="grid-2"
      title={`問い合わせ詳細：${inquiryId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム：問い合わせ詳細 */}
      <div className="inq-detail">
        <h2 className="inq-detail__title">{title}</h2>

        <div className="inq-detail__meta">
          <div>
            <span className="inq-detail__label">ユーザー</span>
            <span className="inq-detail__value">{user}</span>
          </div>
          <div>
            <span className="inq-detail__label">ステータス</span>
            {status === "未対応" ? (
              <span className="inq__badge inq__badge--danger">
                <span className="inq__dot" />
                未対応
              </span>
            ) : (
              <span className="inq__badge inq__badge--neutral">
                <span className="inq__dot" />
                対応中
              </span>
            )}
          </div>
          <div>
            <span className="inq-detail__label">タイプ</span>
            <span className="inq__chip">{type}</span>
          </div>
          <div>
            <span className="inq-detail__label">担当者</span>
            <span className="inq-detail__value">{owner}</span>
          </div>
          <div>
            <span className="inq-detail__label">問い合わせ日</span>
            <span className="inq-detail__value">{inquiredAt}</span>
          </div>
          <div>
            <span className="inq-detail__label">応答日</span>
            <span className="inq-detail__value">{answeredAt}</span>
          </div>
        </div>

        <div className="inq-detail__body">
          <div className="inq-detail__label">問い合わせ本文</div>
          <p className="inq-detail__text">{body}</p>
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={owner}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setOwner("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", owner)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
