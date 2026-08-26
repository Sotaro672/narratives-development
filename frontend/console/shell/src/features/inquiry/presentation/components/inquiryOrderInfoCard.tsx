// frontend/console/shell/src/features/inquiry/presentation/components/inquiryOrderInfoCard.tsx

import { Link } from "react-router-dom";

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type {
  InquiryOrderItemSummary,
  InquiryOrderSummary,
} from "../../../../shared/types/inquiry";

export type InquiryOrderInfoCardProps = {
  productName?: string | null;
  brandName?: string | null;
  orders?: InquiryOrderSummary[];
  isUnopenedReturn?: boolean;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

function normalizeText(
  value: unknown,
): string {
  return String(value ?? "").trim();
}

function uniqueTextValues(
  values: Array<string | null | undefined>,
): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized = normalizeText(value);

    if (!normalized || normalized === "-") {
      continue;
    }

    if (seen.has(normalized)) {
      continue;
    }

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

function getOrderTransferredAtLabel(
  order: InquiryOrderSummary,
): string {
  if (order.items.length === 0) {
    return "-";
  }

  const transferredAtValues = uniqueTextValues(
    order.items.map(
      (item: InquiryOrderItemSummary) =>
        item.transferredAt ?? null,
    ),
  );

  if (transferredAtValues.length === 0) {
    return "-";
  }

  return transferredAtValues
    .map((transferredAt) =>
      safeDateTimeLabelJa(
        transferredAt,
        "-",
      ),
    )
    .join(" / ");
}

export default function InquiryOrderInfoCard({
  productName,
  brandName,
  orders = [],
  isUnopenedReturn = false,
}: InquiryOrderInfoCardProps) {
  const targetOrderItem =
    orders.flatMap(
      (order: InquiryOrderSummary) =>
        order.items,
    )[0] ?? null;

  const productDisplayName =
    `${textOrDash(productName)} / ${textOrDash(brandName)}`;

  const tokenDisplayName =
    `${textOrDash(targetOrderItem?.tokenName)} / ${textOrDash(
      targetOrderItem?.tokenBrandName,
    )}`;

  const quantity =
    targetOrderItem?.qty ?? 0;

  const returnStatus =
    targetOrderItem?.isReturnCompleted
      ? "返品対応済"
      : targetOrderItem?.isReturnRequested
        ? "返品対応中"
        : "-";

  const returnRequestedAt =
    safeDateTimeLabelJa(
      targetOrderItem?.returnRequestedAt,
      "-",
    );

  const returnCompletedAt =
    safeDateTimeLabelJa(
      targetOrderItem?.returnCompletedAt,
      "-",
    );

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          商品・注文情報
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="inq-detail">
          <div className="inq-detail__meta">
            <div>
              <span className="inq-detail__label">
                商品名
              </span>

              <span className="inq-detail__value">
                {productDisplayName}
              </span>
            </div>

            <div>
              <span className="inq-detail__label">
                トークン名
              </span>

              <span className="inq-detail__value">
                {tokenDisplayName}
              </span>
            </div>

            <div>
              <span className="inq-detail__label">
                数量
              </span>

              <span className="inq-detail__value">
                {quantity}
              </span>
            </div>

            {isUnopenedReturn ? (
              <>
                <div>
                  <span className="inq-detail__label">
                    返品ステータス
                  </span>

                  <span className="inq-detail__value">
                    {returnStatus}
                  </span>
                </div>

                <div>
                  <span className="inq-detail__label">
                    返品申請日
                  </span>

                  <span className="inq-detail__value">
                    {returnRequestedAt}
                  </span>
                </div>

                <div>
                  <span className="inq-detail__label">
                    返品完了日
                  </span>

                  <span className="inq-detail__value">
                    {returnCompletedAt}
                  </span>
                </div>
              </>
            ) : null}

            {orders.length > 0 ? (
              orders.flatMap(
                (
                  order: InquiryOrderSummary,
                  index: number,
                ) => [
                  <div
                    key={`${order.id}-id-${index}`}
                  >
                    <span className="inq-detail__label">
                      注文ID
                    </span>

                    <Link
                      to={`/order/${encodeURIComponent(
                        order.id,
                      )}`}
                      className="inq-detail__value inq-detail__value--link"
                    >
                      {textOrDash(
                        order.id,
                      )}
                    </Link>
                  </div>,

                  <div
                    key={`${order.id}-created-at-${index}`}
                  >
                    <span className="inq-detail__label">
                      発注日時
                    </span>

                    <span className="inq-detail__value">
                      {safeDateTimeLabelJa(
                        order.createdAt,
                        "-",
                      )}
                    </span>
                  </div>,

                  <div
                    key={`${order.id}-transferred-at-${index}`}
                  >
                    <span className="inq-detail__label">
                      移譲日
                    </span>

                    <span className="inq-detail__value">
                      {getOrderTransferredAtLabel(
                        order,
                      )}
                    </span>
                  </div>,
                ],
              )
            ) : (
              <div className="inq__empty">
                注文情報はありません。
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}