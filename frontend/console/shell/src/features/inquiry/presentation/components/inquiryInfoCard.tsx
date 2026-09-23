// frontend/console/shell/src/features/inquiry/presentation/components/inquiryInfoCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import "../../../../styles/inquiry-page.css";

export type InquiryInfoCardProps = {
  userName?: string | null;
  createdAt?: string | null;
  updatedAt?: string | null;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

export default function InquiryInfoCard({
  userName,
  createdAt,
  updatedAt,
}: InquiryInfoCardProps) {
  const userNameLabel = textOrDash(userName);
  const createdAtLabel = safeDateTimeLabelJa(createdAt, "-");
  const updatedAtLabel = safeDateTimeLabelJa(updatedAt, "-");

  return (
    <Card>
      <CardHeader>
        <CardTitle>問い合わせ情報</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="sm">
          <div className="inq-info-row">
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="semibold"
              wrap="nowrap"
            >
              ユーザー名
            </Text>

            <Text
              as="div"
              size="sm"
              wrap="anywhere"
            >
              {userNameLabel}
            </Text>
          </div>

          <div className="inq-info-row">
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="semibold"
              wrap="nowrap"
            >
              問い合わせ日
            </Text>

            <Text
              as="div"
              size="sm"
              wrap="anywhere"
            >
              {createdAtLabel}
            </Text>
          </div>

          <div className="inq-info-row">
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="semibold"
              wrap="nowrap"
            >
              最終更新日
            </Text>

            <Text
              as="div"
              size="sm"
              wrap="anywhere"
            >
              {updatedAtLabel}
            </Text>
          </div>
        </Stack>
      </CardContent>
    </Card>
  );
}