// frontend/console/admin/src/presentation/hook/useAdminCard.tsx
import { useEffect, useMemo, useState, useCallback } from "react";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// uid → displayName 取得
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

// AdminService
import {
  type AssigneeCandidate,
  fetchAssigneeCandidatesForCurrentCompany,
} from "../../application/AdminService";

export type UseAdminCardResult = {
  assigneeCandidates: AssigneeCandidate[];
  loadingMembers: boolean;

  /**
   * 互換のため名前は ById のまま残す。
   * ただし現在の正は uid。
   */
  getAssigneeNameById: (uid: string | null | undefined) => Promise<string>;

  getDefaultAssigneeName: () => string;
};

function s(value: unknown): string {
  return String(value ?? "").trim();
}

function getCandidateUid(candidate: unknown): string {
  const c = candidate as any;

  return (
    s(c?.uid) ||
    s(c?.firebaseUid) ||
    s(c?.firebaseUID) ||
    s(c?.authUid) ||
    s(c?.authUID) ||
    s(c?.userUid) ||
    s(c?.userUID) ||
    ""
  );
}

function getCandidateName(candidate: unknown, fallback = ""): string {
  const c = candidate as any;

  return (
    s(c?.name) ||
    s(c?.displayName) ||
    s(c?.fullName) ||
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

      const uid = getCandidateUid(c);

      /**
       * 重要:
       * - assigneeId には uid を保存したい
       * - id が docId の可能性があるため、uid が取れる場合は id を uid に上書きする
       */
      const normalizedId = uid || s(c?.id);
      if (!normalizedId) return null;

      const name = getCandidateName(c, normalizedId);

      return {
        ...c,
        id: normalizedId,
        uid: uid || normalizedId,
        name,
      } as AssigneeCandidate;
    })
    .filter(Boolean) as AssigneeCandidate[];
}

function normalizeNameMapByUid(args: {
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
    const c = candidate as any;

    const uid = getCandidateUid(c) || s(c?.id);
    if (!uid) continue;

    const name = getCandidateName(c);
    if (!name) continue;

    out[uid] = name;
  }

  return out;
}

export function useAdminCard(): UseAdminCardResult {
  const { currentMember } = useAuth();

  /**
   * 現在の方針:
   * - GET /members/{uid} で取得する
   * - assigneeId には uid を保存する
   *
   * そのため、この関数も uid を渡す前提で使う。
   */
  const { getNameLastFirstByID } = useMemberList();

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

        const normalizedNameMap = normalizeNameMapByUid({
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

  const currentMemberUid = useMemo(() => {
    const m = currentMember as any;

    return (
      s(m?.uid) ||
      s(m?.firebaseUid) ||
      s(m?.firebaseUID) ||
      s(m?.authUid) ||
      s(m?.authUID) ||
      ""
    );
  }, [currentMember]);

  const defaultAssigneeName = useMemo(() => {
    return (
      `${currentMember?.lastName ?? ""} ${currentMember?.firstName ?? ""}`.trim() ||
      currentMember?.fullName ||
      currentMember?.email ||
      currentMemberUid ||
      "未設定"
    );
  }, [currentMember, currentMemberUid]);

  const getDefaultAssigneeName = useCallback(() => {
    return defaultAssigneeName;
  }, [defaultAssigneeName]);

  const getAssigneeNameById = useCallback(
    async (uid: string | null | undefined): Promise<string> => {
      const normalizedUid = s(uid);

      if (!normalizedUid) {
        return "未設定";
      }

      // 1. まず候補一覧から uid / id で解決する
      const matched = assigneeCandidates.find((candidate: any) => {
        const candidateUid = getCandidateUid(candidate) || s(candidate?.id);
        return candidateUid === normalizedUid;
      });

      const candidateName = getCandidateName(matched);
      if (candidateName) {
        return candidateName;
      }

      // 2. AdminService の nameMap も uid key 前提で解決する
      if (assigneeNameMap[normalizedUid]) {
        return assigneeNameMap[normalizedUid];
      }

      // 3. 最後に GET /members/{uid} 経由で解決する
      try {
        const resolved = await getNameLastFirstByID(normalizedUid);
        if (resolved) {
          return resolved;
        }
      } catch {
        // ignore
      }

      return defaultAssigneeName;
    },
    [
      assigneeCandidates,
      assigneeNameMap,
      defaultAssigneeName,
      getNameLastFirstByID,
    ],
  );

  return {
    assigneeCandidates,
    loadingMembers,
    getAssigneeNameById,
    getDefaultAssigneeName,
  };
}