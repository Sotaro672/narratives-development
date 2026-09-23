// frontend/console/shell/src/features/inquiry/presentation/components/inquiryReplyListCard.tsx

import AvatarIcon from "../../../../shared/ui/avatarIcon";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import Empty from "../../../../shared/ui/empty";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";
import type { InquiryDetail } from "../../../../shared/types/inquiry";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import InquiryImageGrid from "./inquiryImageGrid";

type InquiryReplyView = InquiryDetail["replies"][number];

export type InquiryReplyListCardProps = {
  replies: InquiryReplyView[];
  memberId: string;
  brandName: string;
  brandIcon: string;
  userName: string;
  onOpenReplyModal: () => void;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();
  return normalized || "-";
}

function replySenderLabel(
  reply: InquiryReplyView,
  brandName: string,
  userName: string,
): string {
  switch (reply.senderType) {
    case "member":
      return textOrDash(brandName);
    case "system":
      return "AMOL";
    case "avatar":
      return textOrDash(userName);
    default:
      return "-";
  }
}

export default function InquiryReplyListCard({
  replies,
  memberId,
  brandName,
  brandIcon,
  userName,
  onOpenReplyModal,
}: InquiryReplyListCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>返信一覧</CardTitle>

        <Button
          type="button"
          onClick={onOpenReplyModal}
        >
          返信
        </Button>
      </CardHeader>

      <CardContent>
        {replies.length > 0 ? (
          <Stack gap="lg">
            {replies.map((reply) => {
              const senderLabel = replySenderLabel(
                reply,
                brandName,
                userName,
              );

              const createdAtLabel = safeDateTimeLabelJa(
                reply.createdAt,
                "-",
              );

              const isSelf =
                reply.senderType === "member" &&
                reply.senderId === memberId;

              const showBrandIcon =
                reply.senderType === "member" &&
                Boolean(brandIcon);

              return (
                <article
                  key={reply.id}
                  className={
                    isSelf
                      ? "inq-reply-item inq-reply-item--self"
                      : "inq-reply-item inq-reply-item--other"
                  }
                >
                  <div className="inq-reply-item__header">
                    <div className="inq-reply-item__sender-profile">
                      {showBrandIcon ? (
                        <AvatarIcon
                          src={brandIcon}
                          alt=""
                        />
                      ) : null}

                      <Text
                        size="xs"
                        weight="bold"
                        wrap="nowrap"
                        className="inq-reply-item__sender"
                      >
                        {senderLabel}
                      </Text>
                    </div>

                    <Text
                      size="xs"
                      tone="muted"
                      weight="medium"
                      wrap="nowrap"
                    >
                      {createdAtLabel}
                    </Text>
                  </div>

                  <Text
                    as="p"
                    size="sm"
                    weight="medium"
                    wrap="pre-wrap-anywhere"
                    className="inq-reply-item__content"
                  >
                    {textOrDash(reply.content)}
                  </Text>

                  {reply.images && reply.images.length > 0 ? (
                    <InquiryImageGrid images={reply.images} />
                  ) : null}
                </article>
              );
            })}
          </Stack>
        ) : (
          <Empty
            compact
            description="返信はありません。"
          />
        )}
      </CardContent>
    </Card>
  );
}