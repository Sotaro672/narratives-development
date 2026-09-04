// frontend/admin/shell/src/pages/InquiriesPage.tsx
import { useEffect, useState } from "react";

import {
  listContacts,
  type Contact,
} from "../features/contact/contactApi";
import Page from "../shared/ui/Page/Page";

export default function InquiriesPage() {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadContacts = async () => {
      try {
        setLoading(true);
        setError(null);

        const result = await listContacts({
          page: 1,
          perPage: 100,
          sort: "createdAt",
          order: "desc",
        });

        if (!cancelled) {
          setContacts(result.items);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(
            cause instanceof Error
              ? cause.message
              : "問い合わせの取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadContacts();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Page>
      <h1>問い合わせ</h1>

      {loading && <p>問い合わせを読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">
          問い合わせの取得に失敗しました。{error}
        </p>
      )}

      {!loading && !error && contacts.length === 0 && (
        <p>問い合わせはありません。</p>
      )}

      {!loading && !error && contacts.length > 0 && (
        <div style={{ overflowX: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr>
                <th style={headerCellStyle}>受信日時</th>
                <th style={headerCellStyle}>名前</th>
                <th style={headerCellStyle}>会社名</th>
                <th style={headerCellStyle}>メールアドレス</th>
                <th style={headerCellStyle}>内容</th>
                <th style={headerCellStyle}>ステータス</th>
                <th style={headerCellStyle}>送信元</th>
              </tr>
            </thead>

            <tbody>
              {contacts.map((contact) => (
                <tr key={contact.id}>
                  <td style={bodyCellStyle}>{formatCreatedAt(contact.createdAt)}</td>
                  <td style={bodyCellStyle}>{contact.name}</td>
                  <td style={bodyCellStyle}>{contact.company || "-"}</td>
                  <td style={bodyCellStyle}>
                    <a href={`mailto:${contact.email}`}>{contact.email}</a>
                  </td>
                  <td style={{ ...bodyCellStyle, whiteSpace: "pre-wrap", minWidth: "320px" }}>
                    {contact.message}
                  </td>
                  <td style={bodyCellStyle}>{contact.status}</td>
                  <td style={bodyCellStyle}>{contact.source || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Page>
  );
}

function formatCreatedAt(value: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("ja-JP");
}

const headerCellStyle = {
  padding: "12px",
  borderBottom: "1px solid #d9d9d9",
  textAlign: "left",
  verticalAlign: "top",
  whiteSpace: "nowrap",
} as const;

const bodyCellStyle = {
  padding: "12px",
  borderBottom: "1px solid #e5e5e5",
  textAlign: "left",
  verticalAlign: "top",
} as const;