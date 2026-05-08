//frontend\amol\src\features\shipping-address\types.ts
export type ShippingAddress = {
  id?: string;
  ID?: string;
  shippingAddressId?: string;
  userId?: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2?: string;
  country?: string;
  createdAt?: string;
  updatedAt?: string;
};

export type UserProfile = {
  id?: string;
  first_name?: string | null;
  first_name_kana?: string | null;
  last_name?: string | null;
  last_name_kana?: string | null;
  createdAt?: string;
  updatedAt?: string;
  deletedAt?: string | null;
};

export type ShippingAddressFormValues = {
  lastName: string;
  firstName: string;
  lastNameKana: string;
  firstNameKana: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type ErrorResponse = {
  error?: string;
};

export type ShippingAddressPageMode = "create" | "edit";

export type ZipCloudAddress = {
  address1: string;
  address2: string;
  address3: string;
  kana1: string;
  kana2: string;
  kana3: string;
  prefcode: string;
  zipcode: string;
};

export type ZipCloudResponse = {
  message: string | null;
  results: ZipCloudAddress[] | null;
  status: number;
};