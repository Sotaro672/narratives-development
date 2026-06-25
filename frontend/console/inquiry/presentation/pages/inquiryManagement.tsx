// frontend/inquiry/src/presentation/pages/InquiryManagement.tsx
import { useEffect, useMemo, useState } from "react";

import List from "../../../shell/src/layout/List/List";

import {
  listInquiriesHTTP,
  type InquiryManagementItem,
} from "../../infrastructure/inquiryRepositoryHTTP";

const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current";

function formatDateTime(value: string | null | undefined): string {
  if (!value) return "-";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "-";

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

function textOrDash(value: string | null | undefined): string {
  const trimmed = String(value ?? "").trim();
  return trimmed || "-";
}

function getInquiryID(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.id);
}

function getSubject(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.subject);
}

function getAvatarID(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.avatarId);
}

function getStatus(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.status);
}

function getInquiryType(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.inquiryType);
}

function getProductBlueprintID(item: InquiryManagementItem): string {
  return textOrDash(item.productBlueprintId);
}

function getCreatedAt(item: InquiryManagementItem): string {
  return formatDateTime(item.inquiry.createdAt);
}

function getUpdatedAt(item: InquiryManagementItem): string {
  return formatDateTime(item.inquiry.updatedAt);
}

export default function InquiryManagementPage() {
  const [items, setItems] = useState<InquiryManagementItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setErrorMessage(null);

      try {
        const result = await listInquiriesHTTP({
          // backend 側では middleware の companyId を正として使う。
          // route 互換のため URL には non-empty placeholder を渡す。
          companyId: CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
        });

        if (!active) return;

        setItems(Array.isArray(result.items) ? result.items : []);
      } catch (error) {
        if (!active) return;

        const message =
          error instanceof Error
            ? error.message
            : "問い合わせ一覧の取得に失敗しました";

        setErrorMessage(message);
        setItems([]);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, []);

  const rows = useMemo(() => {
    return items.map((item) => {
      const inquiryID = getInquiryID(item);

      return (
        <tr key={inquiryID}>
          <td>{getSubject(item)}</td>
          <td>{getAvatarID(item)}</td>
          <td>{getStatus(item)}</td>
          <td>{getInquiryType(item)}</td>
          <td>{getProductBlueprintID(item)}</td>
          <td>{getCreatedAt(item)}</td>
          <td>{getUpdatedAt(item)}</td>
        </tr>
      );
    });
  }, [items]);

  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={[
          "件名",
          "Avatar ID",
          "ステータス",
          "タイプ",
          "商品設計ID",
          "問い合わせ日",
          "最終更新日",
        ]}
        showCreateButton={false}
        showResetButton={false}
      >
        {loading ? (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">問い合わせ一覧を読み込み中です。</div>
            </td>
          </tr>
        ) : errorMessage ? (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">{errorMessage}</div>
            </td>
          </tr>
        ) : rows.length > 0 ? (
          rows
        ) : (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">問い合わせはありません。</div>
            </td>
          </tr>
        )}
      </List>
    </div>
  );
}