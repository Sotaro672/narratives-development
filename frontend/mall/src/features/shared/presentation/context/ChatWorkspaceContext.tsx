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

export type ChatWorkspaceComposer = {
  content: string;
  placeholder?: string;
  files?: File[];
  error?: string | null;
  submitting?: boolean;
  canSubmit: boolean;
  disabled?: boolean;
  submitLabel?: string;
  submittingLabel?: string;
  maxLength?: number | null;
  maxFiles?: number;
  accept?: string;
  onContentChange: (value: string) => void;
  onFilesAdd?: (files: File[]) => void;
  onRemoveFile?: (index: number) => void;
  onSubmit: () => void | Promise<void>;
};

type RegisteredChatWorkspaceComposer = ChatWorkspaceComposer & {
  registrationId: number;
};

type ChatWorkspaceContextValue = {
  composer: ChatWorkspaceComposer | null;
  registerComposer: (composer: ChatWorkspaceComposer) => () => void;
  clearComposer: () => void;
};

const ChatWorkspaceContext =
  createContext<ChatWorkspaceContextValue | null>(null);

type ChatWorkspaceProviderProps = {
  children: ReactNode;
};

function normalizeComposer(
  composer: ChatWorkspaceComposer,
  registrationId: number,
): RegisteredChatWorkspaceComposer {
  return {
    ...composer,
    placeholder:
      composer.placeholder?.trim() ||
      "メッセージを入力",
    files: composer.files ?? [],
    error: composer.error ?? null,
    submitting: composer.submitting ?? false,
    disabled: composer.disabled ?? false,
    submitLabel:
      composer.submitLabel?.trim() ||
      "送信",
    submittingLabel:
      composer.submittingLabel?.trim() ||
      "送信中",
    maxLength:
      composer.maxLength === undefined
        ? 500
        : composer.maxLength,
    maxFiles: composer.maxFiles ?? 10,
    accept:
      composer.accept?.trim() ||
      "image/*",
    registrationId,
  };
}

export function ChatWorkspaceProvider({
  children,
}: ChatWorkspaceProviderProps) {
  const registrationIdRef = useRef(0);
  const activeRegistrationIdRef = useRef<number | null>(null);

  const [
    registeredComposer,
    setRegisteredComposer,
  ] = useState<RegisteredChatWorkspaceComposer | null>(null);

  const registerComposer = useCallback(
    (
      composer: ChatWorkspaceComposer,
    ): (() => void) => {
      registrationIdRef.current += 1;

      const registrationId =
        registrationIdRef.current;

      activeRegistrationIdRef.current =
        registrationId;

      setRegisteredComposer(
        normalizeComposer(
          composer,
          registrationId,
        ),
      );

      return () => {
        queueMicrotask(() => {
          if (
            activeRegistrationIdRef.current !==
            registrationId
          ) {
            return;
          }

          activeRegistrationIdRef.current = null;

          setRegisteredComposer(
            (currentComposer) => {
              if (
                !currentComposer ||
                currentComposer.registrationId !==
                  registrationId
              ) {
                return currentComposer;
              }

              return null;
            },
          );
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

  const composer =
    useMemo<ChatWorkspaceComposer | null>(
      () => {
        if (!registeredComposer) {
          return null;
        }

        return {
          content:
            registeredComposer.content,
          placeholder:
            registeredComposer.placeholder,
          files:
            registeredComposer.files,
          error:
            registeredComposer.error,
          submitting:
            registeredComposer.submitting,
          canSubmit:
            registeredComposer.canSubmit,
          disabled:
            registeredComposer.disabled,
          submitLabel:
            registeredComposer.submitLabel,
          submittingLabel:
            registeredComposer.submittingLabel,
          maxLength:
            registeredComposer.maxLength,
          maxFiles:
            registeredComposer.maxFiles,
          accept:
            registeredComposer.accept,
          onContentChange:
            registeredComposer.onContentChange,
          onFilesAdd:
            registeredComposer.onFilesAdd,
          onRemoveFile:
            registeredComposer.onRemoveFile,
          onSubmit:
            registeredComposer.onSubmit,
        };
      },
      [registeredComposer],
    );

  const value =
    useMemo<ChatWorkspaceContextValue>(
      () => ({
        composer,
        registerComposer,
        clearComposer,
      }),
      [
        composer,
        registerComposer,
        clearComposer,
      ],
    );

  return (
    <ChatWorkspaceContext.Provider
      value={value}
    >
      {children}
    </ChatWorkspaceContext.Provider>
  );
}

export function useChatWorkspace(): ChatWorkspaceContextValue {
  const context =
    useContext(ChatWorkspaceContext);

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