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

// AdminService
import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;

  /**
   * assigneeIdから表示名を取得する。
   *
   * NOTE:
   * フロント側で/members/{uid}を叩く名前解決は行わない。
   * Backend responseのassigneeName、createdByName、
   * displayName、nameを正とする。
   */
  getAssigneeNameById: (
    assigneeId:
      | string
      | null
      | undefined,
  ) => Promise<string>;

  getDefaultAssigneeName: () => string;
};

function s(
  value: unknown,
): string {
  return String(
    value ?? "",
  ).trim();
}

/**
 * Candidate側のIDを正規化する。
 *
 * ProductBlueprint responseの正:
 * - assigneeId
 * - assigneeName
 * - createdBy
 * - createdByName
 *
 * AssigneeCandidate側はAdminServiceのresponseに合わせて
 * idとnameを正とする。
 */
function getCandidateId(
  candidate: unknown,
): string {
  const value =
    candidate as any;

  return (
    s(value?.id) ||
    s(value?.assigneeId) ||
    s(value?.createdBy) ||
    ""
  );
}

/**
 * Candidate側の表示名を取得する。
 *
 * Backend responseのname系を正として使用する。
 */
function getCandidateName(
  candidate: unknown,
  fallback = "",
): string {
  const value =
    candidate as any;

  return (
    s(value?.assigneeName) ||
    s(value?.createdByName) ||
    s(value?.displayName) ||
    s(value?.name) ||
    s(value?.email) ||
    s(fallback)
  );
}

function normalizeAssigneeCandidates(
  candidates: AssigneeCandidate[],
): AssigneeCandidate[] {
  return (
    Array.isArray(candidates)
      ? candidates
      : []
  )
    .map((candidate) => {
      const value =
        candidate as any;

      const id =
        getCandidateId(value);

      if (!id) {
        return null;
      }

      const name =
        getCandidateName(
          value,
          id,
        );

      return {
        ...value,
        id,
        name,
      } as AssigneeCandidate;
    })
    .filter(
      (
        candidate,
      ): candidate is AssigneeCandidate =>
        candidate !== null,
    );
}

function normalizeNameMapById(
  args: {
    candidates:
      AssigneeCandidate[];
    nameMap:
      Record<string, string>;
  },
): Record<string, string> {
  const output:
    Record<string, string> = {};

  const rawNameMap =
    args.nameMap ?? {};

  for (
    const [
      key,
      value,
    ] of Object.entries(
      rawNameMap,
    )
  ) {
    const normalizedKey =
      s(key);

    const normalizedValue =
      s(value);

    if (
      !normalizedKey ||
      !normalizedValue
    ) {
      continue;
    }

    output[normalizedKey] =
      normalizedValue;
  }

  const candidates =
    Array.isArray(
      args.candidates,
    )
      ? args.candidates
      : [];

  for (
    const candidate of candidates
  ) {
    const id =
      getCandidateId(
        candidate,
      );

    if (!id) {
      continue;
    }

    const name =
      getCandidateName(
        candidate,
      );

    if (!name) {
      continue;
    }

    output[id] = name;
  }

  return output;
}

export function useAdminCard(): UseAdminCardResult {
  const {
    currentMember,
  } = useAuthContext();

  const [
    loadingMembers,
    setLoadingMembers,
  ] = useState(false);

  const [
    assigneeCandidates,
    setAssigneeCandidates,
  ] = useState<
    AssigneeCandidate[]
  >([]);

  const [
    assigneeNameMap,
    setAssigneeNameMap,
  ] = useState<
    Record<string, string>
  >({});

  useEffect(() => {
    let alive = true;

    async function loadCandidates() {
      setLoadingMembers(true);

      try {
        const {
          candidates,
          nameMap,
        } =
          await fetchAssigneeCandidatesForCurrentCompany();

        if (!alive) {
          return;
        }

        const normalizedCandidates =
          normalizeAssigneeCandidates(
            candidates,
          );

        const normalizedNameMap =
          normalizeNameMapById({
            candidates:
              normalizedCandidates,
            nameMap:
              nameMap ?? {},
          });

        setAssigneeCandidates(
          normalizedCandidates,
        );

        setAssigneeNameMap(
          normalizedNameMap,
        );
      } finally {
        if (alive) {
          setLoadingMembers(false);
        }
      }
    }

    void loadCandidates();

    return () => {
      alive = false;
    };
  }, []);

  /**
   * currentMemberはGET /members/meのresponseを正とする。
   *
   * 正:
   * - id
   * - uid
   * - firstName
   * - lastName
   * - email
   * - displayName
   */
  const currentMemberId =
    useMemo(
      () =>
        s(currentMember?.id),
      [currentMember?.id],
    );

  const defaultAssigneeName =
    useMemo(() => {
      const displayName =
        s(
          currentMember
            ?.displayName,
        );

      if (displayName) {
        return displayName;
      }

      const fullName =
        `${
          currentMember
            ?.lastName ?? ""
        } ${
          currentMember
            ?.firstName ?? ""
        }`.trim();

      if (fullName) {
        return fullName;
      }

      return (
        s(
          currentMember
            ?.email,
        ) ||
        currentMemberId ||
        "未設定"
      );
    }, [
      currentMember?.displayName,
      currentMember?.lastName,
      currentMember?.firstName,
      currentMember?.email,
      currentMemberId,
    ]);

  const getDefaultAssigneeName =
    useCallback(
      () =>
        defaultAssigneeName,
      [defaultAssigneeName],
    );

  const getAssigneeNameById =
    useCallback(
      async (
        assigneeId:
          | string
          | null
          | undefined,
      ): Promise<string> => {
        const normalizedId =
          s(assigneeId);

        if (!normalizedId) {
          return "未設定";
        }

        const matched =
          assigneeCandidates.find(
            (candidate) =>
              getCandidateId(
                candidate,
              ) ===
              normalizedId,
          );

        const candidateName =
          getCandidateName(
            matched,
          );

        if (candidateName) {
          return candidateName;
        }

        const mappedName =
          assigneeNameMap[
            normalizedId
          ];

        if (mappedName) {
          return mappedName;
        }

        return defaultAssigneeName;
      },
      [
        assigneeCandidates,
        assigneeNameMap,
        defaultAssigneeName,
      ],
    );

  return {
    assigneeCandidates,
    loadingMembers,
    getAssigneeNameById,
    getDefaultAssigneeName,
  };
}