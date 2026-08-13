// frontend/console/shell/src/features/admin/presentation/hook/useAdminCard.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeCandidates:
    AssigneeCandidate[];

  loadingMembers:
    boolean;

  /**
   * Firebase Auth UID である assigneeId から
   * 担当者候補の表示名を取得する。
   *
   * Frontend から追加の member API は呼ばない。
   */
  getAssigneeNameById: (
    assigneeId:
      | string
      | null
      | undefined,
  ) => Promise<string>;

  /**
   * 現在ログイン中 Member の表示名を返す。
   */
  getDefaultAssigneeName:
    () => string;
};

export function useAdminCard():
  UseAdminCardResult {
  const {
    currentMember,
  } =
    useAuthContext();

  const [
    loadingMembers,
    setLoadingMembers,
  ] =
    useState(
      false,
    );

  const [
    assigneeCandidates,
    setAssigneeCandidates,
  ] =
    useState<
      AssigneeCandidate[]
    >([]);

  useEffect(
    () => {
      let alive =
        true;

      async function loadCandidates() {
        setLoadingMembers(
          true,
        );

        try {
          const {
            candidates,
          } =
            await fetchAssigneeCandidatesForCurrentCompany();

          if (!alive) {
            return;
          }

          setAssigneeCandidates(
            candidates,
          );
        } finally {
          if (alive) {
            setLoadingMembers(
              false,
            );
          }
        }
      }

      void loadCandidates();

      return () => {
        alive =
          false;
      };
    },
    [],
  );

  const defaultAssigneeName =
    useMemo(
      () => {
        if (!currentMember) {
          return "未設定";
        }

        return (
          currentMember.displayName ||
          currentMember.email ||
          "未設定"
        );
      },
      [
        currentMember,
      ],
    );

  const getDefaultAssigneeName =
    useCallback(
      () =>
        defaultAssigneeName,
      [
        defaultAssigneeName,
      ],
    );

  const getAssigneeNameById =
    useCallback(
      async (
        assigneeId:
          | string
          | null
          | undefined,
      ): Promise<string> => {
        if (!assigneeId) {
          return "未設定";
        }

        const matched =
          assigneeCandidates.find(
            (candidate) =>
              candidate.id ===
              assigneeId,
          );

        if (matched) {
          return matched.name;
        }

        if (
          currentMember?.uid ===
          assigneeId
        ) {
          return (
            currentMember.displayName ||
            currentMember.email ||
            "未設定"
          );
        }

        return "未設定";
      },
      [
        assigneeCandidates,
        currentMember,
      ],
    );

  return {
    assigneeCandidates,
    loadingMembers,
    getAssigneeNameById,
    getDefaultAssigneeName,
  };
}