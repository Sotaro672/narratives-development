// frontend/inquiry/src/pages/inquiryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageHeader from "../../../shell/src/layout/PageHeader/PageHeader";
import { Card, CardHeader, CardTitle, CardContent } from "../../../shared/ui/card";

export default function InquiryDetail() {
  const navigate = useNavigate();
  const { inquiryId } = useParams<{ inquiryId: string }>();

  // ─────────────────────────────────────────
  // モックデータ
  // ─────────────────────────────────────────
  const [title] = React.useState("シルクブラウスのサイズ交換について");
  const [user] = React.useState("Creator Alice");
  const [status] = React.useState<"対応中" | "未対応">("未対応");
  const [type] = React.useState<"商品説明" | "交換">("交換");
  const [owner] = React.useState("佐藤 美咲");
  const [inquiredAt] = React.useState("2024/9/20");
  const [answeredAt] = React.useState("2024/9/20");
  const [body] = React.useState(
    "LUMINA Fashionのプレミアムシルクブラウスを購入しましたが、サイズが合わないため交換を希望します。交換可能かどうかと、手続き方法を教えてください。"
  );

  // 戻るボタン
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <div className="p-6">
      <PageHeader title={`問い合わせ詳細：${inquiryId ?? "不明ID"}`} onBack={onBack} />


    </div>
  );
}
