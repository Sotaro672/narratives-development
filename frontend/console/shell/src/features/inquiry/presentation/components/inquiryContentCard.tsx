// frontend/console/shell/src/features/inquiry/presentation/components/inquiryContentCard.tsx

import { Button } from "../../../../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Label } from "../../../../shared/ui/label";

import type { InquiryImageFile } from "../../../../shared/types/inquiry";

import InquiryImageGrid from "./inquiryImageGrid";

import "../../../../styles/inquiry-page.css";

export type InquiryContentCardProps = {
  content?: string | null;
  images?: InquiryImageFile[];
  errorMessage?: string | null;

  showReturnRefund?: boolean;
  merchandiseRefundAmount?: number | "";
  merchandiseRefundMaxAmount?: number;
  refundOutboundShipping?: boolean;
  coverReturnShipping?: boolean;
  returnRefundSubmitting?: boolean;
  returnRefundSelectionLocked?: boolean;
  returnRefundCanSubmit?: boolean;
  returnRefundErrorMessage?: string | null;
  onChangeMerchandiseRefundAmount?: (value: string | number) => void;
  onChangeRefundOutboundShipping?: (value: boolean) => void;
  onChangeCoverReturnShipping?: (value: boolean) => void;
  onSubmitReturnRefund?: () => unknown;
};

function textOrDash(value: string | null | undefined): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

function formatCurrency(value: number): string {
  if (!Number.isFinite(value)) {
    return "-";
  }

  return `${Math.max(0, value).toLocaleString("ja-JP")}円`;
}

export default function InquiryContentCard({
  content,
  images,
  errorMessage,
  showReturnRefund = false,
  merchandiseRefundAmount = "",
  merchandiseRefundMaxAmount = 0,
  refundOutboundShipping = false,
  coverReturnShipping = false,
  returnRefundSubmitting = false,
  returnRefundSelectionLocked = false,
  returnRefundCanSubmit = false,
  returnRefundErrorMessage,
  onChangeMerchandiseRefundAmount,
  onChangeRefundOutboundShipping,
  onChangeCoverReturnShipping,
  onSubmitReturnRefund,
}: InquiryContentCardProps) {
  const body = textOrDash(content);
  const inputDisabled =
    returnRefundSubmitting ||
    returnRefundSelectionLocked;

  return (
    <Card>
      <CardHeader>
        <CardTitle>問い合わせ内容</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="inq-detail">
          {errorMessage ? (
            <ErrorMessage>
              {errorMessage}
            </ErrorMessage>
          ) : null}

          <div className="inq-detail__body">
            <p className="inq-detail__text">{body}</p>
          </div>

          {images && images.length > 0 ? (
            <div className="inq-detail__body">
              <div className="inq-detail__label">添付画像</div>
              <InquiryImageGrid images={images} />
            </div>
          ) : null}

          {showReturnRefund ? (
            <div className="inq-detail__body">
              <div className="inq-detail__label">返品の返金内容</div>

              <div className="inq-return-refund">
                <div className="inq-return-refund__field">
                  <Label
                    htmlFor="merchandise-refund-amount"
                    className="inq-return-refund__label"
                  >
                    返金額（税込）
                  </Label>

                  <div className="inq-return-refund__amount-row">
                    <input
                      id="merchandise-refund-amount"
                      type="number"
                      inputMode="numeric"
                      min={1}
                      max={
                        merchandiseRefundMaxAmount > 0
                          ? merchandiseRefundMaxAmount
                          : undefined
                      }
                      step={1}
                      value={merchandiseRefundAmount}
                      onChange={(event) =>
                        onChangeMerchandiseRefundAmount?.(
                          event.target.value,
                        )
                      }
                      disabled={inputDisabled}
                      aria-label="商品返金額"
                      className="inq-return-refund__amount-input"
                    />

                    <span className="inq-return-refund__currency">
                      円
                    </span>
                  </div>

                  <div className="inq-detail__text">
                    商品代金（税込）の返金上限:{" "}
                    {formatCurrency(merchandiseRefundMaxAmount)}
                  </div>

                  <div className="inq-detail__text">
                    1円以上、商品代金（税込）の範囲内で返金額を指定してください。
                  </div>
                </div>

                <label className="inq-return-refund__option">
                  <input
                    type="checkbox"
                    checked={refundOutboundShipping}
                    onChange={(event) =>
                      onChangeRefundOutboundShipping?.(
                        event.target.checked,
                      )
                    }
                    disabled={inputDisabled}
                    className="inq-return-refund__checkbox"
                  />

                  <span className="inq-return-refund__option-content">
                    <span className="inq-return-refund__option-title">
                      購入時の配送料も返金する
                    </span>

                    <span className="inq-detail__text">
                      購入者が支払った往路の配送料とその消費税を、Stripe返金額に含めます。
                    </span>
                  </span>
                </label>

                <label className="inq-return-refund__option">
                  <input
                    type="checkbox"
                    checked={coverReturnShipping}
                    onChange={(event) =>
                      onChangeCoverReturnShipping?.(
                        event.target.checked,
                      )
                    }
                    disabled={inputDisabled}
                    className="inq-return-refund__checkbox"
                  />

                  <span className="inq-return-refund__option-content">
                    <span className="inq-return-refund__option-title">
                      返品時の配送料をブランド側が負担する
                    </span>

                    <span className="inq-detail__text">
                      復路の配送料をブランド側の負担として計上します。購入者のStripe返金額には加算されません。
                    </span>
                  </span>
                </label>

                {returnRefundErrorMessage ? (
                  <ErrorMessage>
                    {returnRefundErrorMessage}
                  </ErrorMessage>
                ) : null}

                {returnRefundSelectionLocked ? (
                  <div className="inq-detail__text">
                    返金処理を開始済みのため、返金額と送料条件は変更できません。
                  </div>
                ) : null}

                <div>
                  <Button
                    type="button"
                    onClick={() => void onSubmitReturnRefund?.()}
                    disabled={
                      !returnRefundCanSubmit ||
                      returnRefundSubmitting ||
                      !onSubmitReturnRefund
                    }
                    aria-busy={returnRefundSubmitting}
                  >
                    {returnRefundSubmitting
                      ? "返品処理中"
                      : "返品受領・返金"}
                  </Button>
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}