// frontend/console/shell/src/features/company/presentation/hook/useCompanyDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import type { Company } from "../../../../shared/types/company";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import {
  fetchCompanyDetail,
  updateCompanyDetail,
} from "../../application/companyDetailService";

type CompanyDetailState = Company & {
  createdByName?: string | null;
  updatedByName?: string | null;
};

function getErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }

  return String(error);
}

export function useCompanyDetail() {
  const navigate = useNavigate();
  const { currentMember } = useAuthContext();

  const companyId = currentMember?.companyId ?? "";

  const [company, setCompany] = useState<CompanyDetailState | null>(null);
  const [companyName, setCompanyName] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const {
    assigneeId: adminMemberId,
    assigneeName: adminMemberName,
    assigneeCandidates: adminCandidates,
    loadingMembers,
    handleSelectAssignee: handleSelectAdmin,
  } = useAssigneeSelection({
    initialAssigneeId: company?.admin ?? null,
    defaultToCurrentMember: false,
  });

  const loadCompany = useCallback(async () => {
    if (!companyId) {
      setCompany(null);
      setCompanyName("");
      return;
    }

    const response = await fetchCompanyDetail(companyId);

    setCompany(response);
    setCompanyName(response.name);
  }, [companyId]);

  const reload = useCallback(async () => {
    if (!companyId) {
      setError("companyId が取得できません。");
      return;
    }

    try {
      setLoading(true);
      setError(null);

      await loadCompany();
    } catch (loadError: unknown) {
      setError(getErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [companyId, loadCompany]);

  useEffect(() => {
    if (!companyId) {
      return;
    }

    void reload();
  }, [companyId, reload]);

  const createdByName = company?.createdByName ?? null;
  const updatedByName = company?.updatedByName ?? null;

  const createdAt = useMemo(() => {
    if (!company?.createdAt) {
      return null;
    }

    return safeDateTimeLabelJa(company.createdAt, "");
  }, [company?.createdAt]);

  const updatedAt = useMemo(() => {
    if (!company?.updatedAt) {
      return null;
    }

    return safeDateTimeLabelJa(company.updatedAt, "");
  }, [company?.updatedAt]);

  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleSave = useCallback(async () => {
    if (!companyId || !company || saving) {
      return;
    }

    if (!companyName) {
      setError("会社名は必須です。");
      return;
    }

    if (!adminMemberId) {
      setError("管理者を選択してください。");
      return;
    }

    try {
      setSaving(true);
      setError(null);

      const updated = await updateCompanyDetail(companyId, {
        name: companyName,
        admin: adminMemberId,
      });

      setCompany(updated);
      setCompanyName(updated.name);
    } catch (saveError: unknown) {
      setError(getErrorMessage(saveError));
    } finally {
      setSaving(false);
    }
  }, [companyId, company, companyName, adminMemberId, saving]);

  return {
    company,
    companyId,
    companyName,
    setCompanyName,

    adminMemberId,
    adminMemberName,
    adminCandidates,
    loadingMembers,
    handleSelectAdmin,

    loading,
    saving,
    error,

    createdByName,
    createdAt,
    updatedByName,
    updatedAt,

    reload,
    handleBack,
    handleSave,
  };
}

export default useCompanyDetail;