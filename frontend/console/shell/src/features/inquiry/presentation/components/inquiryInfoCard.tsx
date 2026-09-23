// frontend/console/shell/src/features/inquiry/presentation/components/inquiryInfoCard.tsx

import {
  Card,
  CardContent,
  CardField,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

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
        <Stack gap="md">
          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
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
          </CardField>

          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
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
          </CardField>

          <CardField>
            <Text
              as="div"
              size="xs"
              tone="muted"
              weight="bold"
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
          </CardField>
        </Stack>
      </CardContent>
    </Card>
  );
}