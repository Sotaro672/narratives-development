// frontend/admin/shell/src/shared/type/gas.ts

export type GasBalance = {
  cluster: string;
  address: string;
  balanceLamports: string;
  balanceSol: number;
};