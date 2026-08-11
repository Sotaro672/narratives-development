// services/solana-bubblegum/src/application/ports/merkle-tree-registry-port.ts

export type MerkleTreeRegistryRecord = {
  treeAddress: string;

  cluster: string;

  maxDepth: number;

  maxBufferSize: number;

  canopyDepth: number;

  public: boolean;

  txSignature: string;

  createdAt: Date;

  updatedAt: Date;
};


export interface MerkleTreeRegistryPort {
  getByKey(
    key: string,
  ): Promise<MerkleTreeRegistryRecord | null>;

  save(
    key: string,
    record: MerkleTreeRegistryRecord,
  ): Promise<void>;
}