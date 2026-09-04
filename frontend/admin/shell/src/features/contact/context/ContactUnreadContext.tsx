// frontend/admin/shell/src/features/contact/context/ContactUnreadContext.tsx
import {
  createContext,
  type ReactNode,
  useContext,
  useMemo,
} from "react";

import { useContactUnreadCount } from "../hooks/useContactUnreadCount";

type ContactUnreadContextValue = {
  unreadCount: number;
  loading: boolean;
  error: string | null;
  refreshUnreadCount: () => Promise<void>;
};

type ContactUnreadProviderProps = {
  children: ReactNode;
};

const ContactUnreadContext =
  createContext<ContactUnreadContextValue | null>(null);

export function ContactUnreadProvider({
  children,
}: ContactUnreadProviderProps) {
  const {
    unreadCount,
    loading,
    error,
    refreshUnreadCount,
  } = useContactUnreadCount();

  const value = useMemo<ContactUnreadContextValue>(
    () => ({
      unreadCount,
      loading,
      error,
      refreshUnreadCount,
    }),
    [
      unreadCount,
      loading,
      error,
      refreshUnreadCount,
    ],
  );

  return (
    <ContactUnreadContext.Provider value={value}>
      {children}
    </ContactUnreadContext.Provider>
  );
}

export function useContactUnread(): ContactUnreadContextValue {
  const context = useContext(ContactUnreadContext);

  if (!context) {
    throw new Error(
      "useContactUnread must be used within ContactUnreadProvider.",
    );
  }

  return context;
}