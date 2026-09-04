// frontend/admin/shell/src/pages/InquiriesPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { listContacts, type Contact } from "../features/contact/contactApi";
import Page from "../shared/ui/Page/Page";
import Table, { type TableColumn } from "../shared/ui/Table/Table";

export default function InquiriesPage() {
  const navigate = useNavigate();
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

  const columns = useMemo<TableColumn<Contact>[]>(() => [
    {
      key: "createdAt",
      header: "受信日時",
      render: (contact) => formatCreatedAt(contact.createdAt),
      nowrap: true,
    },
    {
      key: "name",
      header: "名前",
      render: (contact) => contact.name,
      nowrap: true,
    },
    {
      key: "company",
      header: "会社名",
      render: (contact) => contact.company || "-",
      nowrap: true,
    },
    {
      key: "email",
      header: "メールアドレス",
      render: (contact) => (
        <a href={`mailto:${contact.email}`}>{contact.email}</a>
      ),
      nowrap: true,
    },
    {
      key: "status",
      header: "ステータス",
      render: (contact) => contact.status,
      nowrap: true,
    },
    {
      key: "source",
      header: "送信元",
      render: (contact) => contact.source || "-",
      nowrap: true,
    },
  ], []);

  const handleRowClick = (contact: Contact) => {
    navigate(`/inquiries/${encodeURIComponent(contact.id)}`, {
      state: { contact },
    });
  };

  return (
    <Page>
      <h1>問い合わせ</h1>

      {loading && <p>問い合わせを読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">
          問い合わせの取得に失敗しました。{error}
        </p>
      )}

      {!loading && !error && (
        <Table
          columns={columns}
          rows={contacts}
          getRowKey={(contact) => contact.id}
          emptyMessage="問い合わせはありません。"
          onRowClick={handleRowClick}
        />
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