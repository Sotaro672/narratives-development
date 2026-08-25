// frontend/console/shell/src/features/production/presentation/hook/useProductionDetail.tsx 
 
import * as React from "react"; 
import { useNavigate, useParams } from "react-router-dom"; 
 
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection"; 
import type { 
  ProductionDetail, 
  ProductionQuantityRow, 
} from "../../../../shared/types/production"; 
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa"; 
import { 
  loadProductionDetail, 
  updateProductionDetail, 
} from "../../application/productionDetailService"; 
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP"; 
 
type Mode = "view" | "edit"; 
 
export function useProductionDetail() { 
  const navigate = useNavigate(); 
  const { productionId } = useParams<{ productionId: string }>(); 
 
  const [production, setProduction] = React.useState<ProductionDetail | null>(null); 
  const [mode, setMode] = React.useState<Mode>("view"); 
  const [loading, setLoading] = React.useState(false); 
  const [deleting, setDeleting] = React.useState(false); 
  const [error, setError] = React.useState<string | null>(null); 
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRow[]>([]); 
 
  const { 
    assigneeId, 
    assigneeName, 
    assigneeCandidates, 
    loadingMembers, 
    handleSelectAssignee, 
    resetAssignee, 
  } = useAssigneeSelection({ 
    initialAssigneeId: production?.assigneeId ?? null, 
    initialAssigneeName: production?.assigneeName ?? null, 
    defaultToCurrentMember: false, 
  }); 
 
  const isViewMode = mode === "view"; 
  const isEditMode = mode === "edit"; 
  const canEdit = production?.printed !== true; 
 
  const creator = production?.createdByName ?? ""; 
  const createdAt = React.useMemo( 
    () => safeDateTimeLabelJa(production?.createdAt, ""), 
    [production?.createdAt], 
  ); 
 
  const updater = React.useMemo(() => { 
    const name = production?.updatedByName ?? ""; 
    const date = safeDateTimeLabelJa(production?.updatedAt, ""); 
    return name && date ? name : ""; 
  }, [production?.updatedByName, production?.updatedAt]); 
 
  const updatedAt = React.useMemo(() => { 
    const name = production?.updatedByName ?? ""; 
    const date = safeDateTimeLabelJa(production?.updatedAt, ""); 
    return name && date ? date : ""; 
  }, [production?.updatedByName, production?.updatedAt]); 
 
  const switchToView = React.useCallback(() => { 
    resetAssignee(); 
 
    if (production) { 
      setQuantityRows(production.models ?? []); 
    } 
 
    setMode("view"); 
  }, [production, resetAssignee]); 
 
  const switchToEdit = React.useCallback(() => { 
    if (!canEdit) return; 
    setMode("edit"); 
  }, [canEdit]); 
 
  const reloadProduction = React.useCallback(async () => { 
    if (!productionId) return; 
 
    try { 
      setLoading(true); 
      setError(null); 
 
      const data = await loadProductionDetail(productionId); 
 
      setProduction(data); 
      setQuantityRows(data?.models ?? []); 
    } catch { 
      setError("生産情報の取得に失敗しました"); 
      setProduction(null); 
      setQuantityRows([]); 
    } finally { 
      setLoading(false); 
    } 
  }, [productionId]); 
 
  React.useEffect(() => { 
    void reloadProduction(); 
  }, [reloadProduction]); 
 
  const onSave = React.useCallback(async () => { 
    if (!productionId || !production) return; 
 
    if (!canEdit) { 
      alert("この生産は編集できません（印刷済みです）。"); 
      return; 
    } 
 
    if (!assigneeId) { 
      alert("担当者を選択してください。"); 
      return; 
    } 
 
    try { 
      const updated = await updateProductionDetail(productionId, { 
        assigneeId, 
        models: quantityRows.map(({ modelId, quantity }) => ({ 
          modelId, 
          quantity, 
        })), 
      }); 
 
      if (updated) { 
        setProduction(updated); 
        setQuantityRows(updated.models ?? []); 
      } 
 
      setMode("view"); 
    } catch { 
      alert("更新に失敗しました"); 
    } 
  }, [ 
    productionId, 
    production, 
    quantityRows, 
    canEdit, 
    assigneeId, 
  ]); 
 
  const onDelete = React.useCallback(async () => { 
    if (!productionId || !production || deleting) return; 
 
    if (production.printed) { 
      alert("印刷済みの生産は削除できません。"); 
      return; 
    } 
 
    try { 
      setDeleting(true); 
 
      const repository = new ProductionRepositoryHTTP(); 
      await repository.delete(productionId); 
 
      navigate("/production"); 
    } catch { 
      alert("生産情報の削除に失敗しました。"); 
    } finally { 
      setDeleting(false); 
    } 
  }, [productionId, production, deleting, navigate]); 
 
  const handleBack = React.useCallback(() => { 
    navigate("/production"); 
  }, [navigate]); 
 
  const onSelectAssignee = React.useCallback( 
    (id: string) => { 
      handleSelectAssignee(id); 
    }, 
    [handleSelectAssignee], 
  ); 
 
  const onEditAssignee = React.useCallback(() => {}, []); 
  const onClickAssignee = React.useCallback(() => {}, []); 
 
  return { 
    isViewMode, 
    isEditMode, 
    switchToView, 
    switchToEdit, 
    canEdit, 
    adminMode: mode, 
    onBack: handleBack, 
    onSave, 
    onDelete, 
    deleting, 
    productionId: productionId ?? null, 
    production, 
    loading, 
    error, 
    quantityRows, 
    setQuantityRows, 
    assigneeId, 
    assigneeName, 
    assigneeCandidates, 
    loadingMembers, 
    creator, 
    createdAt, 
    updater, 
    updatedAt, 
    onSelectAssignee, 
    onEditAssignee, 
    onClickAssignee, 
    reloadProduction, 
  }; 
} 
 
export default useProductionDetail;