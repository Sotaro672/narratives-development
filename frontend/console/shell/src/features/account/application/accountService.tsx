// frontend/console/shell/src/features/account/application/accountService.tsx 
 
import type { 
  Account, 
  AccountStatus, 
  AccountType, 
} from "../../../shared/types/account"; 
import { 
  safeDateTimeLabelJa, 
} from "../../../shared/util/dateJa"; 
import { 
  listBrands, 
  type BrandRow, 
} from "../../brand/application/brandService"; 
import { 
  accountRepositoryHTTP, 
} from "../infrastructure/http/accountRepositoryHTTP"; 
 
export type AccountBrand = { 
  id: string; 
  name: string; 
}; 
 
export type AccountRow = { 
  id: string; 
  stripeAccountId: string; 
  memberId: string; 
  bankName: string; 
  branchName: string; 
  accountNumber: number; 
  accountNumberLabel: string; 
  accountType: AccountType | ""; 
  accountTypeLabel: string; 
  currency: string; 
  connectedBrands: AccountBrand[]; 
  status: AccountStatus; 
  statusLabel: string; 
  registeredAt: string; 
}; 
 
function accountStatusLabel( 
  status: AccountStatus, 
): string { 
  switch (status) { 
    case "active": 
      return "利用可能"; 
 
    case "inactive": 
      return "設定中"; 
 
    case "suspended": 
      return "利用停止"; 
 
    case "deleted": 
      return "削除済み"; 
  } 
} 
 
function accountNumberLabel( 
  accountNumber: number, 
): string { 
  if ( 
    !Number.isInteger( 
      accountNumber, 
    ) || 
    accountNumber <= 0 
  ) { 
    return "未設定"; 
  } 
 
  return String( 
    accountNumber, 
  ); 
} 
 
function accountTypeLabel( 
  accountType: AccountType | "", 
): string { 
  if (!accountType) { 
    return "未設定"; 
  } 
 
  return accountType; 
} 
 
function accountTextLabel( 
  value: string, 
): string { 
  const normalized = 
    String( 
      value ?? "", 
    ).trim(); 
 
  return normalized || 
    "未設定"; 
} 
 
function connectedBrandsForAccount( 
  accountId: string, 
  brands: BrandRow[], 
): AccountBrand[] { 
  return brands 
    .filter( 
      (brand) => 
        brand.accountId === accountId, 
    ) 
    .map( 
      (brand) => ({ 
        id: brand.id, 
        name: brand.name, 
      }), 
    ); 
} 
 
function toAccountRow( 
  account: Account, 
  brands: BrandRow[], 
): AccountRow { 
  return { 
    id: account.id, 
 
    stripeAccountId: 
      accountTextLabel( 
        account.stripeAccountId, 
      ), 
 
    memberId: 
      accountTextLabel( 
        account.memberId, 
      ), 
 
    bankName: 
      accountTextLabel( 
        account.bankName, 
      ), 
 
    branchName: 
      accountTextLabel( 
        account.branchName, 
      ), 
 
    accountNumber: 
      account.accountNumber, 
 
    accountNumberLabel: 
      accountNumberLabel( 
        account.accountNumber, 
      ), 
 
    accountType: 
      account.accountType, 
 
    accountTypeLabel: 
      accountTypeLabel( 
        account.accountType, 
      ), 
 
    currency: 
      accountTextLabel( 
        account.currency, 
      ), 
 
    connectedBrands: 
      connectedBrandsForAccount( 
        account.id, 
        brands, 
      ), 
 
    status: 
      account.status, 
 
    statusLabel: 
      accountStatusLabel( 
        account.status, 
      ), 
 
    registeredAt: 
      safeDateTimeLabelJa( 
        account.createdAt, 
        "", 
      ), 
  }; 
} 
 
// Account一覧取得 
// 
// - GET /accounts を利用する 
// - Brand一覧も取得し、Brand.accountId と Account.id を突合する 
// - CompanyIDはBackendの認証Contextから解決する 
// - Backendから返されたAccountとBrandを正とする 
// - deletedを含め、Companyに属するAccountをすべて返す 
// - presentation用の表示文字列と日時のみ変換する 
export async function listAccounts(): Promise< 
  AccountRow[] 
> { 
  const [ 
    accounts, 
    brands, 
  ] = await Promise.all([ 
    accountRepositoryHTTP.list(), 
    listBrands(), 
  ]); 
 
  return accounts.map( 
    (account) => 
      toAccountRow( 
        account, 
        brands, 
      ), 
  ); 
} 