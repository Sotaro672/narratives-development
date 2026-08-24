// frontend/console/shell/src/features/brand/presentation/components/accountSelectCard.tsx 
 
import * as React from "react"; 
 
import { 
  Card, 
  CardHeader, 
  CardTitle, 
  CardContent, 
} from "../../../../shared/ui/card"; 
 
import "../../../../styles/brand.css"; 
 
export type AccountStatus = 
  | "active" 
  | "inactive" 
  | "suspended" 
  | "deleted"; 
 
export type AccountCandidate = { 
  id: string; 
  label: string; 
  status?: AccountStatus; 
}; 
 
export type AccountSelectCardProps = { 
  title?: string; 
 
  accountId?: string | null; 
  accountLabel?: string | null; 
 
  accountCandidates?: AccountCandidate[]; 
  loadingAccounts?: boolean; 
  accountError?: string | null; 
}; 
 
export const AccountSelectCard: React.FC<AccountSelectCardProps> = ({ 
  title = "売上受取口座", 
  accountId, 
  accountLabel, 
  accountCandidates, 
  loadingAccounts, 
  accountError, 
}) => { 
  const candidates = accountCandidates ?? []; 
  const loading = Boolean(loadingAccounts); 
 
  const selectedAccount = React.useMemo( 
    () => candidates.find((candidate) => candidate.id === accountId) ?? null, 
    [candidates, accountId], 
  ); 
 
  const displayLabel = 
    accountLabel || 
    selectedAccount?.label || 
    accountId || 
    "未設定"; 
 
  return ( 
    <Card className="admin-card"> 
      <CardHeader className="admin-card__header"> 
        <CardTitle className="admin-card__title"> 
          {title} 
        </CardTitle> 
      </CardHeader> 
 
      <CardContent className="admin-card__body space-y-4"> 
        <div className="admin-card__section"> 
          <div className="admin-card__label mb-1 text-xs text-slate-500"> 
            接続口座 
          </div> 
 
          {loading ? ( 
            <div className="py-1 text-sm text-slate-400"> 
              口座を読み込み中です… 
            </div> 
          ) : accountError ? ( 
            <div className="whitespace-pre-wrap py-1 text-xs text-red-500"> 
              {accountError} 
            </div> 
          ) : ( 
            <div className="py-1 text-sm text-slate-800"> 
              {displayLabel} 
            </div> 
          )} 
        </div> 
 
        {accountId && ( 
          <div className="admin-card__section"> 
            <div className="admin-card__label mb-1 text-xs text-slate-500"> 
              Account ID 
            </div> 
 
            <div className="break-all py-1 text-xs text-slate-500"> 
              {accountId} 
            </div> 
          </div> 
        )} 
      </CardContent> 
    </Card> 
  ); 
}; 
 
export default AccountSelectCard;