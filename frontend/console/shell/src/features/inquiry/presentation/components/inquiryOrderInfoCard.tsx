// frontend/console/shell/src/features/inquiry/presentation/components/inquiryOrderInfoCard.tsx

import { Link as RouterLink } from "react-router-dom";

import {
  Card,
  CardContent,
  CardField,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Empty from "../../../../shared/ui/empty";
import Link from "../../../../shared/ui/link";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import type { InquiryOrderSummary } from "../../../../shared/types/inquiry";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

export type InquiryOrderInfoCardProps = {
  productName?: string | null;
  brandName?: string | null;
  orders?: InquiryOrderSummary[];
  isUnopenedReturn?: boolean;
};

function textOrDash(value: string | null | undefined): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

export default function InquiryOrderInfoCard({
  productName,
  brandName,
  orders = [],
  isUnopenedReturn = false,
}: InquiryOrderInfoCardProps) {
  const targetOrderItem =
    orders.flatMap((order: InquiryOrderSummary) => order.items)[0] ?? null;

  const productDisplayName =
    `${textOrDash(productName)} / ${textOrDash(brandName)}`;

  const tokenDisplayName =
    `${textOrDash(targetOrderItem?.tokenName)} / ${textOrDash(
      targetOrderItem?.tokenBrandName,
    )}`;

  const quantity = targetOrderItem?.qty ?? 0;

  const returnStatus =
    targetOrderItem?.isReturnCompleted
      ? "返品対応済"
      : targetOrderItem?.isReturnRequested
        ? "返品対応中"
        : "-";

  const returnRequestedAt = safeDateTimeLabelJa(
    targetOrderItem?.returnRequestedAt,
    "-",
  );

  const returnCompletedAt = safeDateTimeLabelJa(
    targetOrderItem?.returnCompletedAt,
    "-",
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>商品・注文情報</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
            >
              商品名
            </Text>

            <Text
              as="div"
              size="sm"
              wrap="anywhere"
            >
              {productDisplayName}
            </Text>
          </CardField>

          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
            >
              トークン名
            </Text>

            <Text
              as="div"
              size="sm"
              wrap="anywhere"
            >
              {tokenDisplayName}
            </Text>
          </CardField>

          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
            >
              数量
            </Text>

            <Text as="div" size="sm">
              {quantity}
            </Text>
          </CardField>

          {isUnopenedReturn ? (
            <>
              <CardField>
                <Text
                  as="div"
                  size="xs"
                  tone="muted"
                  weight="bold"
                >
                  返品ステータス
                </Text>

                <Text as="div" size="sm">
                  {returnStatus}
                </Text>
              </CardField>

              <CardField>
                <Text
                  as="div"
                  size="xs"
                  tone="muted"
                  weight="bold"
                >
                  返品申請日
                </Text>

                <Text as="div" size="sm">
                  {returnRequestedAt}
                </Text>
              </CardField>

              <CardField>
                <Text
                  as="div"
                  size="xs"
                  tone="muted"
                  weight="bold"
                >
                  返品完了日
                </Text>

                <Text as="div" size="sm">
                  {returnCompletedAt}
                </Text>
              </CardField>
            </>
          ) : null}

          {orders.length > 0 ? (
            orders.map((order: InquiryOrderSummary, index: number) => (
              <Stack key={`${order.id}-${index}`} gap="md">
                <CardField>
                  <Text
                    as="div"
                    size="xs"
                    tone="muted"
                    weight="bold"
                  >
                    注文ID
                  </Text>

                  <Text
                    as="div"
                    size="sm"
                    wrap="anywhere"
                  >
                    <Link asChild>
                      <RouterLink
                        to={`/order/${encodeURIComponent(order.id)}`}
                      >
                        {textOrDash(order.id)}
                      </RouterLink>
                    </Link>
                  </Text>
                </CardField>

                <CardField>
                  <Text
                    as="div"
                    size="xs"
                    tone="muted"
                    weight="bold"
                  >
                    発注日時
                  </Text>

                  <Text
                    as="div"
                    size="sm"
                    wrap="anywhere"
                  >
                    {safeDateTimeLabelJa(order.createdAt, "-")}
                  </Text>
                </CardField>
              </Stack>
            ))
          ) : (
            <Empty
              compact
              description="注文情報はありません。"
            />
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}