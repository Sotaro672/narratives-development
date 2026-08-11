// services/solana-bubblegum/src/infrastructure/firestore/merkle-tree-registry-repository.ts

import {
  Timestamp,
} from "@google-cloud/firestore";

import type {
  MerkleTreeRegistryPort,
  MerkleTreeRegistryRecord,
} from "../../application/ports/merkle-tree-registry-port.js";

import {
  firestore,
} from "./firestore-client.js";


type FirestoreMerkleTreeRecord = {
  treeAddress?: unknown;

  cluster?: unknown;

  maxDepth?: unknown;

  maxBufferSize?: unknown;

  canopyDepth?: unknown;

  public?: unknown;

  txSignature?: unknown;

  createdAt?: unknown;

  updatedAt?: unknown;
};


function requiredString(
  field: string,
  value: unknown,
): string {
  if (
    typeof value !== "string" ||
    value.length === 0
  ) {
    throw new Error(
      `merkle_tree_registry: invalid ${field}`,
    );
  }

  return value;
}


function requiredInteger(
  field: string,
  value: unknown,
  minimum: number,
): number {
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < minimum
  ) {
    throw new Error(
      `merkle_tree_registry: invalid ${field}`,
    );
  }

  return value;
}


function requiredBoolean(
  field: string,
  value: unknown,
): boolean {
  if (
    typeof value !== "boolean"
  ) {
    throw new Error(
      `merkle_tree_registry: invalid ${field}`,
    );
  }

  return value;
}


function requiredDate(
  field: string,
  value: unknown,
): Date {
  if (
    value instanceof Timestamp
  ) {
    return value.toDate();
  }

  if (
    value instanceof Date
  ) {
    return value;
  }

  throw new Error(
    `merkle_tree_registry: invalid ${field}`,
  );
}


function fromFirestore(
  data: FirestoreMerkleTreeRecord,
): MerkleTreeRegistryRecord {
  return {
    treeAddress:
      requiredString(
        "treeAddress",
        data.treeAddress,
      ),

    cluster:
      requiredString(
        "cluster",
        data.cluster,
      ),

    maxDepth:
      requiredInteger(
        "maxDepth",
        data.maxDepth,
        1,
      ),

    maxBufferSize:
      requiredInteger(
        "maxBufferSize",
        data.maxBufferSize,
        1,
      ),

    canopyDepth:
      requiredInteger(
        "canopyDepth",
        data.canopyDepth,
        0,
      ),

    public:
      requiredBoolean(
        "public",
        data.public,
      ),

    txSignature:
      requiredString(
        "txSignature",
        data.txSignature,
      ),

    createdAt:
      requiredDate(
        "createdAt",
        data.createdAt,
      ),

    updatedAt:
      requiredDate(
        "updatedAt",
        data.updatedAt,
      ),
  };
}


function validateRecord(
  record: MerkleTreeRegistryRecord,
): void {
  requiredString(
    "treeAddress",
    record.treeAddress,
  );

  requiredString(
    "cluster",
    record.cluster,
  );

  requiredInteger(
    "maxDepth",
    record.maxDepth,
    1,
  );

  requiredInteger(
    "maxBufferSize",
    record.maxBufferSize,
    1,
  );

  requiredInteger(
    "canopyDepth",
    record.canopyDepth,
    0,
  );

  requiredBoolean(
    "public",
    record.public,
  );

  requiredString(
    "txSignature",
    record.txSignature,
  );

  requiredDate(
    "createdAt",
    record.createdAt,
  );

  requiredDate(
    "updatedAt",
    record.updatedAt,
  );
}


function assertCompatible(
  key: string,
  existing: MerkleTreeRegistryRecord,
  next: MerkleTreeRegistryRecord,
): void {
  if (
    existing.treeAddress !==
    next.treeAddress
  ) {
    throw new Error(
      [
        "merkle_tree_registry: tree conflict",
        `key=${key}`,
        `existing=${existing.treeAddress}`,
        `new=${next.treeAddress}`,
      ].join(
        " ",
      ),
    );
  }

  if (
    existing.cluster !==
    next.cluster
  ) {
    throw new Error(
      [
        "merkle_tree_registry: cluster conflict",
        `key=${key}`,
        `existing=${existing.cluster}`,
        `new=${next.cluster}`,
      ].join(
        " ",
      ),
    );
  }

  if (
    existing.maxDepth !==
    next.maxDepth
  ) {
    throw new Error(
      [
        "merkle_tree_registry: maxDepth conflict",
        `key=${key}`,
        `existing=${existing.maxDepth}`,
        `new=${next.maxDepth}`,
      ].join(
        " ",
      ),
    );
  }

  if (
    existing.maxBufferSize !==
    next.maxBufferSize
  ) {
    throw new Error(
      [
        "merkle_tree_registry: maxBufferSize conflict",
        `key=${key}`,
        `existing=${existing.maxBufferSize}`,
        `new=${next.maxBufferSize}`,
      ].join(
        " ",
      ),
    );
  }

  if (
    existing.canopyDepth !==
    next.canopyDepth
  ) {
    throw new Error(
      [
        "merkle_tree_registry: canopyDepth conflict",
        `key=${key}`,
        `existing=${existing.canopyDepth}`,
        `new=${next.canopyDepth}`,
      ].join(
        " ",
      ),
    );
  }

  if (
    existing.public !==
    next.public
  ) {
    throw new Error(
      [
        "merkle_tree_registry: public conflict",
        `key=${key}`,
        `existing=${existing.public}`,
        `new=${next.public}`,
      ].join(
        " ",
      ),
    );
  }
}


export class FirestoreMerkleTreeRegistryRepository
  implements MerkleTreeRegistryPort {
  async getByKey(
    key: string,
  ): Promise<MerkleTreeRegistryRecord | null> {
    if (!key) {
      throw new Error(
        "merkle_tree_registry: key is required",
      );
    }

    const snapshot =
      await firestore
        .collection(
          "bubblegumMerkleTrees",
        )
        .doc(
          key,
        )
        .get();

    if (!snapshot.exists) {
      return null;
    }

    return fromFirestore(
      snapshot.data() as FirestoreMerkleTreeRecord,
    );
  }


  async save(
    key: string,
    record: MerkleTreeRegistryRecord,
  ): Promise<void> {
    if (!key) {
      throw new Error(
        "merkle_tree_registry: key is required",
      );
    }

    validateRecord(
      record,
    );

    const ref =
      firestore
        .collection(
          "bubblegumMerkleTrees",
        )
        .doc(
          key,
        );

    await firestore.runTransaction(
      async (transaction) => {
        const snapshot =
          await transaction.get(
            ref,
          );

        if (snapshot.exists) {
          const existing =
            fromFirestore(
              snapshot.data() as FirestoreMerkleTreeRecord,
            );

          assertCompatible(
            key,
            existing,
            record,
          );

          return;
        }

        transaction.create(
          ref,
          {
            treeAddress:
              record.treeAddress,

            cluster:
              record.cluster,

            maxDepth:
              record.maxDepth,

            maxBufferSize:
              record.maxBufferSize,

            canopyDepth:
              record.canopyDepth,

            public:
              record.public,

            txSignature:
              record.txSignature,

            createdAt:
              Timestamp.fromDate(
                record.createdAt,
              ),

            updatedAt:
              Timestamp.fromDate(
                record.updatedAt,
              ),
          },
        );
      },
    );
  }
}