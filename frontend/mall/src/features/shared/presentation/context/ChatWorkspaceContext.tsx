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

import type { ChatComposerConfig } from "../../types/chatComposer";

type RegisteredChatWorkspaceComposer = ChatComposerConfig & {
  registrationId: number;
};

type ChatWorkspaceContextValue = {
  composer: ChatComposerConfig | null;
  headerTitle: string;
  registerComposer: (composer: ChatComposerConfig) => () => void;
  clearComposer: () => void;
  setHeaderTitle: (title: string) => void;
  clearHeaderTitle: () => void;
};

const ChatWorkspaceContext = createContext<ChatWorkspaceContextValue | null>(null);

type ChatWorkspaceProviderProps = {
  children: ReactNode;
};

function normalizeComposer(
  composer: ChatComposerConfig,
  registrationId: number,
): RegisteredChatWorkspaceComposer {
  return {
    ...composer,
    placeholder: composer.placeholder?.trim() || "メッセージを入力",
    files: composer.files ?? [],
    error: composer.error ?? null,
    submitting: composer.submitting ?? false,
    disabled: composer.disabled ?? false,
    submitLabel: composer.submitLabel?.trim() || "送信",
    submittingLabel: composer.submittingLabel?.trim() || "送信中",
    maxLength: composer.maxLength === undefined ? 500 : composer.maxLength,
    maxFiles: composer.maxFiles ?? 10,
    accept: composer.accept?.trim() || "image/*",
    registrationId,
  };
}

export function ChatWorkspaceProvider({
  children,
}: ChatWorkspaceProviderProps) {
  const registrationIdRef = useRef(0);
  const activeRegistrationIdRef = useRef<number | null>(null);
  const [registeredComposer, setRegisteredComposer] = useState<RegisteredChatWorkspaceComposer | null>(null);
  const [headerTitle, setHeaderTitleState] = useState("");

  const registerComposer = useCallback(
    (composer: ChatComposerConfig): (() => void) => {
      registrationIdRef.current += 1;
      const registrationId = registrationIdRef.current;
      activeRegistrationIdRef.current = registrationId;
      setRegisteredComposer(normalizeComposer(composer, registrationId));

      return () => {
        queueMicrotask(() => {
          if (activeRegistrationIdRef.current !== registrationId) {
            return;
          }

          activeRegistrationIdRef.current = null;
          setRegisteredComposer((currentComposer) => {
            if (!currentComposer || currentComposer.registrationId !== registrationId) {
              return currentComposer;
            }

            return null;
          });
        });
      };
    },
    [],
  );

  const clearComposer = useCallback(() => {
    registrationIdRef.current += 1;
    activeRegistrationIdRef.current = null;
    setRegisteredComposer(null);
  }, []);

  const setHeaderTitle = useCallback((title: string) => {
    setHeaderTitleState(title.trim());
  }, []);

  const clearHeaderTitle = useCallback(() => {
    setHeaderTitleState("");
  }, []);

  const composer = useMemo<ChatComposerConfig | null>(() => {
    if (!registeredComposer) {
      return null;
    }

    return {
      content: registeredComposer.content,
      placeholder: registeredComposer.placeholder,
      files: registeredComposer.files,
      error: registeredComposer.error,
      submitting: registeredComposer.submitting,
      canSubmit: registeredComposer.canSubmit,
      disabled: registeredComposer.disabled,
      submitLabel: registeredComposer.submitLabel,
      submittingLabel: registeredComposer.submittingLabel,
      maxLength: registeredComposer.maxLength,
      maxFiles: registeredComposer.maxFiles,
      accept: registeredComposer.accept,
      onContentChange: registeredComposer.onContentChange,
      onFilesAdd: registeredComposer.onFilesAdd,
      onRemoveFile: registeredComposer.onRemoveFile,
      onSubmit: registeredComposer.onSubmit,
    };
  }, [registeredComposer]);

  const value = useMemo<ChatWorkspaceContextValue>(
    () => ({
      composer,
      headerTitle,
      registerComposer,
      clearComposer,
      setHeaderTitle,
      clearHeaderTitle,
    }),
    [
      composer,
      headerTitle,
      registerComposer,
      clearComposer,
      setHeaderTitle,
      clearHeaderTitle,
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
    throw new Error("useChatWorkspace must be used within ChatWorkspaceProvider.");
  }

  return context;
}

export function useOptionalChatWorkspace(): ChatWorkspaceContextValue | null {
  return useContext(ChatWorkspaceContext);
}