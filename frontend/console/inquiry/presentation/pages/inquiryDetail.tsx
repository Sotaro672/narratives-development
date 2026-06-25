// frontend/inquiry/src/pages/inquiryDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../shell/src/shared/ui/card";

import {
  getInquiryHTTP,
  type InquiryDetail as InquiryDetailDTO,
} from "../../infrastructure/inquiryRepositoryHTTP";

function textOrDash(value: string | null | undefined): string {
  const trimmed = String(value ?? "").trim();
  return trimmed || "-";
}

function statusLabel(value: string | null | undefined): string {
  const status = String(value ?? "").trim();

  switch (status) {
    case "open":
      return "未対応";
    case "in_progress":
      return "対応中";
    case "resolved":
      return "対応済み";
    case "closed":
      return "クローズ";
    default:
      return status || "-";
  }
}

function isUnresolvedStatus(value: string | null | undefined): boolean {
  const status = String(value ?? "").trim();

  return status === "" || status === "open" || status === "unresolved";
}

function typeLabel(value: string | null | undefined): string {
  const inquiryType = String(value ?? "").trim();

  switch (inquiryType) {
    case "product_description":
      return "商品説明";
    case "exchange":
      return "交換";
    case "shipping":
      return "配送";
    case "payment":
      return "決済";
    case "other":
      return "その他";
    default:
      return inquiryType || "-";
  }
}

export default function InquiryDetail() {
  const navigate = useNavigate();
  const { inquiryId } = useParams<{ inquiryId: string }>();

  const [detail, setDetail] = React.useState<InquiryDetailDTO | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  React.useEffect(() => {
    let active = true;

    async function load() {
      const trimmedInquiryId = String(inquiryId ?? "").trim();

      if (!trimmedInquiryId) {
        setDetail(null);
        setErrorMessage("問い合わせIDが指定されていません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setErrorMessage(null);

      try {
        const result = await getInquiryHTTP(trimmedInquiryId);

        if (!active) return;

        setDetail(result);
      } catch (error) {
        if (!active) return;

        const message =
          error instanceof Error
            ? error.message
            : "問い合わせ詳細の取得に失敗しました";

        setErrorMessage(message);
        setDetail(null);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, [inquiryId]);

  const inquiry = detail?.inquiry ?? null;

  const title = textOrDash(inquiry?.subject);
  const body = textOrDash(inquiry?.content);
  const avatarName = textOrDash(detail?.avatarName);
  const status = statusLabel(inquiry?.status);
  const type = typeLabel(inquiry?.inquiryType);
  const productName = textOrDash(detail?.productName);
  const brandName = textOrDash(detail?.brandName);
  const inquiredAt = safeDateTimeLabelJa(inquiry?.createdAt, "-");
  const updatedAt = safeDateTimeLabelJa(inquiry?.updatedAt, "-");

  const statusBadge = isUnresolvedStatus(inquiry?.status) ? (
    <span className="inq__badge inq__badge--danger">
      <span className="inq__dot" />
      {status}
    </span>
  ) : (
    <span className="inq__badge inq__badge--neutral">
      <span className="inq__dot" />
      {status}
    </span>
  );

  if (loading) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="inq__empty">問い合わせ詳細を読み込み中です。</div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">問い合わせ情報を読み込み中です。</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">商品情報を読み込み中です。</div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  if (errorMessage) {
    return (
      <PageStyle
        layout="grid-2"
        title="問い合わせ詳細"
        onBack={onBack}
        onSave={undefined}
      >
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ内容</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="inq__empty">{errorMessage}</div>
          </CardContent>
        </Card>

        <div>
          <Card>
            <CardHeader>
              <CardTitle>問い合わせ情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">問い合わせ情報を表示できません。</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>商品情報</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="inq__empty">商品情報を表示できません。</div>
            </CardContent>
          </Card>
        </div>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title="問い合わせ詳細"
      onBack={onBack}
      onSave={undefined}
    >
      <Card>
        <CardHeader>
          <CardTitle>問い合わせ内容</CardTitle>
        </CardHeader>

        <CardContent>
          <div className="inq-detail">
            <h2 className="inq-detail__title">{title}</h2>

            <div className="inq-detail__meta">
              <div>
                <span className="inq-detail__label">タイプ</span>
                <span className="inq__chip">{type}</span>
              </div>
            </div>

            <div className="inq-detail__body">
              <div className="inq-detail__label">問い合わせ本文</div>
              <p className="inq-detail__text">{body}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div>
        <Card>
          <CardHeader>
            <CardTitle>問い合わせ情報</CardTitle>
          </CardHeader>

          <CardContent>
            <div className="inq-detail">
              <div className="inq-detail__meta">
                <div>
                  <span className="inq-detail__label">アバター名</span>
                  <span className="inq-detail__value">{avatarName}</span>
                </div>

                <div>
                  <span className="inq-detail__label">ステータス</span>
                  {statusBadge}
                </div>

                <div>
                  <span className="inq-detail__label">問い合わせ日</span>
                  <span className="inq-detail__value">{inquiredAt}</span>
                </div>

                <div>
                  <span className="inq-detail__label">最終更新日</span>
                  <span className="inq-detail__value">{updatedAt}</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>商品情報</CardTitle>
          </CardHeader>

          <CardContent>
            <div className="inq-detail">
              <div className="inq-detail__meta">
                <div>
                  <span className="inq-detail__label">商品名</span>
                  <span className="inq-detail__value">{productName}</span>
                </div>

                <div>
                  <span className="inq-detail__label">ブランド</span>
                  <span className="inq-detail__value">{brandName}</span>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}