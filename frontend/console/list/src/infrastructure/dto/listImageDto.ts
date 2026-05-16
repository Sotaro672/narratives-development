// frontend\console\list\src\infrastructure\dto\listImageDto.ts

export type SaveListImageFromFirebaseStorageInput = {
  listId: string;

  id: string;
  url: string;
  objectPath: string;

  size: number;
  displayOrder: number;

  fileName?: string;
  contentType?: string;

  createdBy?: string;
  createdAt?: string;
};