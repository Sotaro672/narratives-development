// frontend/console/admin/src/presentation/hook/useAdminCard.tsx

import { useEffect, useMemo, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// AdminService
import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;

  /**
   * assigneeId から表示名を取得する。
   *
   * NOTE:
   * フロント側で /members/{uid} を叩く名前解決は行わない。
   * backend response の assigneeName / createdByName / displayName / name を正とする。
   */
  getAssigneeNameById: (assigneeId: string | null | undefined) => Promise<string>;

  getDefaultAssigneeName: () => string;
};

function s(value: unknown): string {
  return String(value ?? "").trim();
}

/**
 * Candidate 側の ID 正規化。
 *
 * ProductBlueprint response の正:
 * - assigneeId
 * - assigneeName
 * - createdBy
 * - createdByName
 *
 * AssigneeCandidate 側は AdminService の response に合わせて id/name を正とする。
 */
function getCandidateId(candidate: unknown): string {
  const c = candidate as any;

  return (
    s(c?.id) ||
    s(c?.assigneeId) ||
    s(c?.createdBy) ||
    ""
  );
}

/**
 * Candidate 側の表示名。
 *
 * backend response の name 系を正として使う。
 */
function getCandidateName(candidate: unknown, fallback = ""): string {
  const c = candidate as any;

  return (
    s(c?.assigneeName) ||
    s(c?.createdByName) ||
    s(c?.displayName) ||
    s(c?.name) ||
    s(c?.email) ||
    s(fallback)
  );
}

function normalizeAssigneeCandidates(
  candidates: AssigneeCandidate[],
): AssigneeCandidate[] {
  return (Array.isArray(candidates) ? candidates : [])
    .map((candidate) => {
      const c = candidate as any;

      const id = getCandidateId(c);
      if (!id) return null;

      const name = getCandidateName(c, id);

      return {
        ...c,
        id,
        name,
      } as AssigneeCandidate;
    })
    .filter(Boolean) as AssigneeCandidate[];
}

function normalizeNameMapById(args: {
  candidates: AssigneeCandidate[];
  nameMap: Record<string, string>;
}): Record<string, string> {
  const out: Record<string, string> = {};

  const rawNameMap = args.nameMap ?? {};
  for (const [key, value] of Object.entries(rawNameMap)) {
    const k = s(key);
    const v = s(value);
    if (!k || !v) continue;
    out[k] = v;
  }

  for (const candidate of Array.isArray(args.candidates) ? args.candidates : []) {
    const id = getCandidateId(candidate);
    if (!id) continue;

    const name = getCandidateName(candidate);
    if (!name) continue;

    out[id] = name;
  }

  return out;
}

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();

  const [loadingMembers, setLoadingMembers] = useState(false);
  const [assigneeCandidates, setAssigneeCandidates] = useState<
    AssigneeCandidate[]
  >([]);
  const [assigneeNameMap, setAssigneeNameMap] =
    useState<Record<string, string>>({});

  useEffect(() => {
    let alive = true;

    (async () => {
      setLoadingMembers(true);

      try {
        const { candidates, nameMap } =
          await fetchAssigneeCandidatesForCurrentCompany();

        if (!alive) return;

        const normalizedCandidates = normalizeAssigneeCandidates(candidates);

        const normalizedNameMap = normalizeNameMapById({
          candidates: normalizedCandidates,
          nameMap: nameMap ?? {},
        });

        setAssigneeCandidates(normalizedCandidates);
        setAssigneeNameMap(normalizedNameMap);
      } finally {
        if (alive) {
          setLoadingMembers(false);
        }
      }
    })();

    return () => {
      alive = false;
    };
  }, []);

  /**
   * currentMember は GET /members/{uid} の response を正とする。
   *
   * 正:
   * - id
   * - uid
   * - firstName
   * - lastName
   * - email
   * - displayName
   */
  const currentMemberId = useMemo(() => {
    return s(currentMember?.id);
  }, [currentMember]);

  const defaultAssigneeName = useMemo(() => {
    const displayName = s(currentMember?.displayName);
    if (displayName) return displayName;

    const fullName = `${currentMember?.lastName ?? ""} ${
      currentMember?.firstName ?? ""
    }`.trim();
    if (fullName) return fullName;

    return s(currentMember?.email) || currentMemberId || "未設定";
  }, [currentMember, currentMemberId]);

  const getDefaultAssigneeName = useCallback(() => {
    return defaultAssigneeName;
  }, [defaultAssigneeName]);

  const getAssigneeNameById = useCallback(
    async (assigneeId: string | null | undefined): Promise<string> => {
      const normalizedId = s(assigneeId);

      if (!normalizedId) {
        return "未設定";
      }

      const matched = assigneeCandidates.find((candidate) => {
        const candidateId = getCandidateId(candidate);
        return candidateId === normalizedId;
      });

      const candidateName = getCandidateName(matched);
      if (candidateName) {
        return candidateName;
      }

      const mappedName = assigneeNameMap[normalizedId];
      if (mappedName) {
        return mappedName;
      }

      return defaultAssigneeName;
    },
    [assigneeCandidates, assigneeNameMap, defaultAssigneeName],
  );

  return {
    assigneeCandidates,
    loadingMembers,
    getAssigneeNameById,
    getDefaultAssigneeName,
  };
}