// frontend/transaction/src/pages/transactionDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardReadonly,
} from "../../../shared/ui/card";

/**
 * 取引詳細ページ
 * layout="single" を採用し、取引情報を整理表示する
 */
export default function TransactionDetail() {
  const navigate = useNavigate();
  const { transactionId } = useParams<{ transactionId: string }>();

  // ─────────────────────────────────────────────
  // モックデータ（API接続前）
  // ─────────────────────────────────────────────
  const [transaction] = React.useState({
    id: transactionId ?? "txn_20241102001",
    brand: "LUMINA Fashion",
    type: "送金", // "受取" | "送金"
    amount: 24800,
    description: "2024年10月度 売上精算",
    counterparty: "CR Garments",
    datetime: "2024/11/02 18:45:00",
    status: "完了", // "完了" | "処理中" | "失敗"
    method: "Solana SPLトークン",
    txHash: "9rZy...eFQk",
  });

  // 戻るボタン
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 金額フォーマット
  const formatAmount = (amt: number) =>
    amt.toLocaleString("ja-JP", { style: "currency", currency: "JPY" });

  // ステータスバッジ
  const statusBadge = (status: string) => {
    switch (status) {
      case "完了":
        return (
          <span className="inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700 text-xs font-semibold">
            完了
          </span>
        );
      case "処理中":
        return (
          <span className="inline-flex items-center px-2 py-1 rounded-full bg-yellow-50 text-yellow-700 text-xs font-semibold">
            処理中
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center px-2 py-1 rounded-full bg-red-50 text-red-700 text-xs font-semibold">
            失敗
          </span>
        );
    }
  };

  return (
    <PageStyle
      layout="single"
      title={`取引詳細：${transaction.id}`}
      onBack={handleBack}
    >
      <div className="space-y-4 max-w-3xl">
        {/* 基本情報 */}
        <Card>
          <CardHeader>
            <CardTitle>取引情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>ブランド</CardLabel>
            <CardReadonly>{transaction.brand}</CardReadonly>

            <CardLabel>種別</CardLabel>
            <CardReadonly>{transaction.type}</CardReadonly>

            <CardLabel>金額</CardLabel>
            <CardReadonly>{formatAmount(transaction.amount)}</CardReadonly>

            <CardLabel>取引先</CardLabel>
            <CardReadonly>{transaction.counterparty}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="border rounded-lg px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {transaction.description}
            </div>
          </CardContent>
        </Card>

        {/* 状況・日時 */}
        <Card>
          <CardHeader>
            <CardTitle>ステータス / 日時</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>ステータス</CardLabel>
            <div>{statusBadge(transaction.status)}</div>

            <CardLabel>日時</CardLabel>
            <CardReadonly>{transaction.datetime}</CardReadonly>
          </CardContent>
        </Card>

        {/* 決済情報 */}
        <Card>
          <CardHeader>
            <CardTitle>決済情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>決済方法</CardLabel>
            <CardReadonly>{transaction.method}</CardReadonly>

            <CardLabel>トランザクションハッシュ</CardLabel>
            <CardReadonly>{transaction.txHash}</CardReadonly>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
