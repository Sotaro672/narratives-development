// frontend/console/shell/src/features/inquiry/presentation/components/inquiryContentCard.tsx

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardSuffix,
  CardTitle,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import { Input } from "../../../../shared/ui/input";
import { Label } from "../../../../shared/ui/label";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

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
        <Stack gap="lg">
          {errorMessage ? (
            <ErrorMessage>
              {errorMessage}
            </ErrorMessage>
          ) : null}

          <Text
            as="p"
            size="md"
            wrap="pre-wrap-anywhere"
          >
            {body}
          </Text>

          {images && images.length > 0 ? (
            <Stack gap="sm">
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="bold"
              >
                添付画像
              </Text>

              <InquiryImageGrid images={images} />
            </Stack>
          ) : null}

          {showReturnRefund ? (
            <Stack gap="sm">
              <Text
                as="div"
                size="xs"
                tone="muted"
                weight="bold"
              >
                返品の返金内容
              </Text>

              <Stack gap="md">
                <Stack gap="sm">
                  <Label htmlFor="merchandise-refund-amount">
                    返金額（税込）
                  </Label>

                  <div className="inq-return-refund__amount-row">
                    <Input
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
                    />

                    <CardSuffix>
                      円
                    </CardSuffix>
                  </div>

                  <Text
                    as="div"
                    size="sm"
                    wrap="pre-wrap-anywhere"
                  >
                    商品代金（税込）:{" "}
                    {formatCurrency(merchandiseRefundMaxAmount)}
                  </Text>

                  <Text
                    as="div"
                    size="sm"
                    wrap="pre-wrap-anywhere"
                  >
                    1円以上、商品代金（税込）の範囲内で返金額を指定してください。
                  </Text>
                </Stack>

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

                  <Stack gap="xs">
                    <Text
                      size="sm"
                      weight="medium"
                    >
                      往路
                    </Text>
                  </Stack>
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

                  <Stack gap="xs">
                    <Text
                      size="sm"
                      weight="medium"
                    >
                      復路
                    </Text>
                  </Stack>
                </label>

                {returnRefundErrorMessage ? (
                  <ErrorMessage>
                    {returnRefundErrorMessage}
                  </ErrorMessage>
                ) : null}

                {returnRefundSelectionLocked ? (
                  <Text
                    as="div"
                    size="sm"
                    wrap="pre-wrap-anywhere"
                  >
                    返金処理を開始済みのため、返金額と送料条件は変更できません。
                  </Text>
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
              </Stack>
            </Stack>
          ) : null}
        </Stack>
      </CardContent>
    </Card>
  );
}