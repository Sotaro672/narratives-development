// frontend/console/shell/src/features/inquiry/presentation/components/inquiryInfoCard.tsx

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

export type InquiryInfoCardProps = {
  userFullName?: string | null;
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
  userFullName,
  createdAt,
  updatedAt,
}: InquiryInfoCardProps) {
  const userNameLabel =
    textOrDash(userFullName);

  const createdAtLabel =
    safeDateTimeLabelJa(
      createdAt,
      "-",
    );

  const updatedAtLabel =
    safeDateTimeLabelJa(
      updatedAt,
      "-",
    );

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          問い合わせ情報
        </CardTitle>
      </CardHeader>

      <CardContent>
        <div className="inq-detail">
          <div className="inq-detail__meta">
            <div>
              <span className="inq-detail__label">
                ユーザー名
              </span>

              <span className="inq-detail__value">
                {userNameLabel}
              </span>
            </div>

            <div>
              <span className="inq-detail__label">
                問い合わせ日
              </span>

              <span className="inq-detail__value">
                {createdAtLabel}
              </span>
            </div>

            <div>
              <span className="inq-detail__label">
                最終更新日
              </span>

              <span className="inq-detail__value">
                {updatedAtLabel}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}