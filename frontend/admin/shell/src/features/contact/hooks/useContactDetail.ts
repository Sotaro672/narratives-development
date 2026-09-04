// frontend/admin/shell/src/features/contact/hooks/useContactDetail.ts
import { useEffect, useState } from "react";

import type { Contact } from "../../../shared/type/contact";
import { useContactUnread } from "../context/ContactUnreadContext";
import {
  getContact,
  markContactAsRead,
} from "../infrastructure/contactApi";

export function useContactDetail(contactId: string | undefined) {
  const { refreshUnreadCount } = useContactUnread();
  const [contact, setContact] = useState<Contact | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!contactId) {
      setContact(null);
      setLoading(false);
      setError("問い合わせIDが指定されていません。");
      return;
    }

    let cancelled = false;

    const load = async () => {
      setLoading(true);
      setError(null);
      setContact(null);

      try {
        const result = await getContact(contactId);
        const resolvedContact = result.isRead
          ? result
          : await markContactAsRead(contactId);

        if (cancelled) {
          return;
        }

        setContact(resolvedContact);

        if (!result.isRead) {
          void refreshUnreadCount();
        }
      } catch (cause) {
        if (!cancelled) {
          setError(
            cause instanceof Error
              ? cause.message
              : "問い合わせ情報の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [contactId, refreshUnreadCount]);

  return {
    contact,
    loading,
    error,
  };
}