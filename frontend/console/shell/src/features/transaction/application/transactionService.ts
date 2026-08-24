// frontend/console/shell/src/features/transaction/application/transactionService.ts 
 
import type { 
  PageResult, 
} from "../../../shared/types/common/common"; 
import type { 
  TransactionManagementRowDTO, 
  TransactionType, 
} from "../../../shared/types/transaction"; 
import { 
  safeDateTimeLabelJa, 
} from "../../../shared/util/dateJa"; 
import { 
  accountRepositoryHTTP, 
} from "../../account/infrastructure/http/accountRepositoryHTTP"; 
import { 
  transactionRepositoryHTTP, 
  type TransactionListParams, 
} from "../infrastructure/http/transactionRepositoryHTTP"; 
 
export type TransactionRow = { 
  id: string; 
  settlementId: string; 
  orderId: string; 
  paymentId: string; 
  accountId: string; 
  accountLabel: string; 
  type: TransactionType; 
  typeLabel: string; 
  amount: number; 
  amountLabel: string; 
  currency: string; 
  description: string; 
  status: string; 
  statusLabel: string; 
  stripeTransferId: string; 
  stripeTransferReversalId: string; 
  timestamp: string; 
  timestampLabel: string; 
}; 
 
function transactionTypeLabel( 
  type: TransactionType, 
): string { 
  switch (type) { 
    case "receive": 
      return "入金"; 
 
    case "send": 
      return "出金"; 
  } 
} 
 
function transactionStatusLabel( 
  status: string, 
): string { 
  switch (status) { 
    case "transferred": 
    case "reversed": 
      return "完了"; 
 
    default: 
      return status || "不明"; 
  } 
} 
 
function transactionAmountLabel( 
  amount: number, 
  currency: string, 
  type: TransactionType, 
): string { 
  const normalizedAmount = 
    Number.isFinite(amount) 
      ? Math.abs(amount) 
      : 0; 
 
  const sign = 
    type === "receive" 
      ? "+" 
      : "-"; 
 
  const normalizedCurrency = 
    String(currency ?? "") 
      .trim() 
      .toUpperCase(); 
 
  if ( 
    normalizedCurrency === "JPY" || 
    normalizedCurrency === "円" 
  ) { 
    return `${sign}${normalizedAmount.toLocaleString("ja-JP")}円`; 
  } 
 
  return `${sign}${normalizedAmount.toLocaleString("ja-JP")} ${normalizedCurrency}`.trim(); 
} 
 
function transactionAccountLabel( 
  accountId: string, 
  accountLabels: ReadonlyMap<string, string>, 
): string { 
  const label = 
    accountLabels.get(accountId); 
 
  if (label) { 
    return label; 
  } 
 
  return accountId || "不明"; 
} 
 
function toTransactionRow( 
  transaction: TransactionManagementRowDTO, 
  accountLabels: ReadonlyMap<string, string>, 
): TransactionRow { 
  return { 
    id: transaction.id, 
    settlementId: transaction.settlementId, 
    orderId: transaction.orderId, 
    paymentId: transaction.paymentId, 
    accountId: transaction.accountId, 
    accountLabel: transactionAccountLabel( 
      transaction.accountId, 
      accountLabels, 
    ), 
    type: transaction.type, 
    typeLabel: transactionTypeLabel( 
      transaction.type, 
    ), 
    amount: transaction.amount, 
    amountLabel: transactionAmountLabel( 
      transaction.amount, 
      transaction.currency, 
      transaction.type, 
    ), 
    currency: transaction.currency, 
    description: transaction.description, 
    status: transaction.status, 
    statusLabel: transactionStatusLabel( 
      transaction.status, 
    ), 
    stripeTransferId: 
      transaction.stripeTransferId ?? "", 
    stripeTransferReversalId: 
      transaction.stripeTransferReversalId ?? "", 
    timestamp: transaction.timestamp, 
    timestampLabel: safeDateTimeLabelJa( 
      transaction.timestamp, 
      "", 
    ), 
  }; 
} 
 
async function buildAccountLabels(): Promise< 
  Map<string, string> 
> { 
  const accounts = 
    await accountRepositoryHTTP.list(); 
 
  const labels = 
    new Map<string, string>(); 
 
  for (const account of accounts) { 
    const bankName = 
      String(account.bankName ?? "") 
        .trim(); 
 
    const branchName = 
      String(account.branchName ?? "") 
        .trim(); 
 
    const label = 
      [bankName, branchName] 
        .filter(Boolean) 
        .join(" "); 
 
    labels.set( 
      account.id, 
      label || account.id, 
    ); 
  } 
 
  return labels; 
} 
 
// Transaction一覧取得 
// 
// - GET /transactions を利用する 
// - CompanyIDはBackendの認証Contextから解決する 
// - Settlementから生成された実際の入出金履歴のみ表示する 
// - 残高情報は取得・表示しない 
// - Account情報は口座表示名の解決だけに利用する 
export async function listTransactions( 
  params: TransactionListParams = {}, 
): Promise<PageResult<TransactionRow>> { 
  const [ 
    result, 
    accountLabels, 
  ] = await Promise.all([ 
    transactionRepositoryHTTP.list( 
      params, 
    ), 
    buildAccountLabels(), 
  ]); 
 
  return { 
    ...result, 
    items: result.items.map( 
      (transaction) => 
        toTransactionRow( 
          transaction, 
          accountLabels, 
        ), 
    ), 
  }; 
}