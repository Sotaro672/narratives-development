// frontend/console/shell/src/features/inquiry/presentation/components/inquiryContentCard.tsx

import { Button } from "../../../../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";

import {
  getOpenedReturnRefundPolicyLabel,
  OPENED_RETURN_REFUND_POLICIES,
} from "../../../../shared/types/inquiry";

import type {
  InquiryImageFile,
  OpenedReturnRefundPolicy,
} from "../../../../shared/types/inquiry";

import InquiryImageGrid from "./inquiryImageGrid";

export type InquiryContentCardProps = {
  content?: string | null;
  images?: InquiryImageFile[];
  errorMessage?: string | null;

  showOpenedReturnRefund?: boolean;
  openedReturnPolicy?: OpenedReturnRefundPolicy | "";
  openedReturnSubmitting?: boolean;
  openedReturnPolicyLocked?: boolean;
  openedReturnCanSubmit?: boolean;
  openedReturnErrorMessage?: string | null;
  onChangeOpenedReturnPolicy?: (value: string) => void;
  onSubmitOpenedReturnRefund?: () => unknown;
};

function textOrDash(value: string | null | undefined): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

export default function InquiryContentCard({
  content,
  images,
  errorMessage,
  showOpenedReturnRefund = false,
  openedReturnPolicy = "",
  openedReturnSubmitting = false,
  openedReturnPolicyLocked = false,
  openedReturnCanSubmit = false,
  openedReturnErrorMessage,
  onChangeOpenedReturnPolicy,
  onSubmitOpenedReturnRefund,
}: InquiryContentCardProps) {
  const body = textOrDash(content);

  return (
    <Card>
      <CardHeader>
        <CardTitle>問い合わせ内容</CardTitle>
      </CardHeader>

      <CardContent>
        <div className="inq-detail">
          {errorMessage ? (
            <div className="inq__empty">{errorMessage}</div>
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

          {showOpenedReturnRefund ? (
            <div className="inq-detail__body">
              <div className="inq-detail__label">開封後返品の返金方法</div>

              <div className="flex flex-col gap-3">
                <select
                  value={openedReturnPolicy}
                  onChange={(event) => onChangeOpenedReturnPolicy?.(event.target.value)}
                  disabled={openedReturnSubmitting || openedReturnPolicyLocked}
                  aria-label="開封後返品の返金方法"
                  className="w-full rounded-md border bg-background px-3 py-2 text-sm"
                >
                  <option value="">返金方法を選択してください</option>

                  {OPENED_RETURN_REFUND_POLICIES.map((policy) => (
                    <option key={policy} value={policy}>
                      {getOpenedReturnRefundPolicyLabel(policy)}
                    </option>
                  ))}
                </select>

                {openedReturnPolicy === "half_merchandise" ? (
                  <div className="inq-detail__text">
                    商品代金と対象商品の消費税の50%を返金します。
                  </div>
                ) : null}

                {openedReturnPolicy === "merchandise_only" ? (
                  <div className="inq-detail__text">
                    商品代金と対象商品の消費税を全額返金します。
                  </div>
                ) : null}

                {openedReturnPolicy === "merchandise_round_trip_shipping" ? (
                  <div className="inq-detail__text">
                    商品代金・対象商品の消費税・購入時の配送料を返金し、返品時の配送料もブランド側が負担します。
                  </div>
                ) : null}

                {openedReturnErrorMessage ? (
                  <div className="inq__empty">{openedReturnErrorMessage}</div>
                ) : null}

                {openedReturnPolicyLocked ? (
                  <div className="inq-detail__text">
                    返金処理を開始済みのため、返金方法は変更できません。
                  </div>
                ) : null}

                <div>
                  <Button
                    type="button"
                    onClick={() => void onSubmitOpenedReturnRefund?.()}
                    disabled={
                      !openedReturnCanSubmit ||
                      openedReturnSubmitting ||
                      !onSubmitOpenedReturnRefund
                    }
                    aria-busy={openedReturnSubmitting}
                  >
                    {openedReturnSubmitting ? "返品処理中" : "返品受領"}
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