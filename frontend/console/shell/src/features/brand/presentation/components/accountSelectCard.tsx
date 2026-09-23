// frontend/console/shell/src/features/brand/presentation/components/accountSelectCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";
import Stack from "../../../../shared/ui/stack";
import { Text } from "../../../../shared/ui/text";

export type AccountStatus =
  | "active"
  | "inactive"
  | "suspended"
  | "deleted";

export type AccountCandidate = {
  id: string;
  label: string;
  status?: AccountStatus;
};

export type AccountSelectCardProps = {
  title?: string;
  accountId?: string | null;
  accountLabel?: string | null;
  accountCandidates?: AccountCandidate[];
  loadingAccounts?: boolean;
  accountError?: string | null;
};

export const AccountSelectCard: React.FC<AccountSelectCardProps> = ({
  title = "売上受取口座",
  accountId,
  accountLabel,
  accountCandidates,
  loadingAccounts,
  accountError,
}) => {
  const candidates = accountCandidates ?? [];
  const loading = Boolean(loadingAccounts);

  const selectedAccount = React.useMemo(
    () => candidates.find((candidate) => candidate.id === accountId) ?? null,
    [candidates, accountId],
  );

  const displayLabel =
    accountLabel ||
    selectedAccount?.label ||
    accountId ||
    "未設定";

  return (
    <Card>
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          <Stack gap="xs">
            <Text as="div" size="xs" tone="muted">
              接続口座
            </Text>

            {loading ? (
              <Text as="div" size="sm" tone="muted">
                口座を読み込み中です…
              </Text>
            ) : accountError ? (
              <ErrorMessage size="xs">
                {accountError}
              </ErrorMessage>
            ) : (
              <Text as="div" size="sm">
                {displayLabel}
              </Text>
            )}
          </Stack>

          {accountId && (
            <Stack gap="xs">
              <Text as="div" size="xs" tone="muted">
                Account ID
              </Text>

              <Text
                as="div"
                size="xs"
                tone="muted"
                wrap="anywhere"
              >
                {accountId}
              </Text>
            </Stack>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default AccountSelectCard;