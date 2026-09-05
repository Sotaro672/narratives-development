// frontend/admin/shell/src/shared/type/company.ts

export type Company = {
  id: string;
  name: string;
  representativeName: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CompanyListResponse = {
  items: Company[];
};