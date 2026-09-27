// frontend/mall/src/features/shared/presentation/context/ChatWorkspaceContext.tsx

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

export type ChatWorkspaceAction = {
  label: string;
  onClick: () => void | Promise<void>;
  disabled?: boolean;
};

type RegisteredChatWorkspaceAction = ChatWorkspaceAction & {
  registrationId: number;
};

type ChatWorkspaceContextValue = {
  action: ChatWorkspaceAction | null;
  registerAction: (action: ChatWorkspaceAction) => () => void;
  clearAction: () => void;
};

const ChatWorkspaceContext =
  createContext<ChatWorkspaceContextValue | null>(null);

type ChatWorkspaceProviderProps = {
  children: ReactNode;
};

export function ChatWorkspaceProvider({
  children,
}: ChatWorkspaceProviderProps) {
  const registrationIdRef = useRef(0);
  const [registeredAction, setRegisteredAction] =
    useState<RegisteredChatWorkspaceAction | null>(null);

  const registerAction = useCallback(
    (action: ChatWorkspaceAction): (() => void) => {
      registrationIdRef.current += 1;
      const registrationId = registrationIdRef.current;

      setRegisteredAction({
        ...action,
        label: action.label.trim(),
        registrationId,
      });

      return () => {
        setRegisteredAction((currentAction) => {
          if (
            !currentAction ||
            currentAction.registrationId !== registrationId
          ) {
            return currentAction;
          }

          return null;
        });
      };
    },
    [],
  );

  const clearAction = useCallback(() => {
    registrationIdRef.current += 1;
    setRegisteredAction(null);
  }, []);

  const action = useMemo<ChatWorkspaceAction | null>(() => {
    if (!registeredAction) {
      return null;
    }

    return {
      label: registeredAction.label,
      onClick: registeredAction.onClick,
      disabled: registeredAction.disabled ?? false,
    };
  }, [registeredAction]);

  const value = useMemo<ChatWorkspaceContextValue>(
    () => ({
      action,
      registerAction,
      clearAction,
    }),
    [
      action,
      clearAction,
      registerAction,
    ],
  );

  return (
    <ChatWorkspaceContext.Provider value={value}>
      {children}
    </ChatWorkspaceContext.Provider>
  );
}

export function useChatWorkspace(): ChatWorkspaceContextValue {
  const context = useContext(ChatWorkspaceContext);

  if (!context) {
    throw new Error(
      "useChatWorkspace must be used within ChatWorkspaceProvider.",
    );
  }

  return context;
}

export function useOptionalChatWorkspace(): ChatWorkspaceContextValue | null {
  return useContext(ChatWorkspaceContext);
}