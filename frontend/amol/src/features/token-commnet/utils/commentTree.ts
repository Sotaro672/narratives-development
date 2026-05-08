// frontend/amol/src/features/token-commnet/utils/commentTree.ts

import type {
  TokenComment,
  TokenCommentTreeNode,
} from "../types/tokenCommentTypes";

function compareCreatedAtAsc(
  a: TokenCommentTreeNode,
  b: TokenCommentTreeNode
): number {
  return a.comment.createdAt.localeCompare(b.comment.createdAt);
}

function compareCreatedAtDesc(
  a: TokenCommentTreeNode,
  b: TokenCommentTreeNode
): number {
  return b.comment.createdAt.localeCompare(a.comment.createdAt);
}

function sortChildrenRecursive(node: TokenCommentTreeNode): void {
  node.children.sort(compareCreatedAtAsc);

  node.children.forEach(sortChildrenRecursive);
}

export function buildTokenCommentTree(
  comments: TokenComment[]
): TokenCommentTreeNode[] {
  const nodeMap = new Map<string, TokenCommentTreeNode>();
  const roots: TokenCommentTreeNode[] = [];

  comments.forEach((comment) => {
    if (!comment.commentId) {
      return;
    }

    nodeMap.set(comment.commentId, {
      comment,
      children: [],
    });
  });

  comments.forEach((comment) => {
    if (!comment.commentId) {
      return;
    }

    const node = nodeMap.get(comment.commentId);

    if (!node) {
      return;
    }

    if (!comment.parentCommentId) {
      roots.push(node);
      return;
    }

    const parent = nodeMap.get(comment.parentCommentId);

    if (!parent) {
      roots.push(node);
      return;
    }

    parent.children.push(node);
  });

  roots.sort(compareCreatedAtDesc);
  roots.forEach(sortChildrenRecursive);

  return roots;
}

export function flattenTokenCommentTree(
  nodes: TokenCommentTreeNode[]
): TokenComment[] {
  return nodes.flatMap((node) => [
    node.comment,
    ...flattenTokenCommentTree(node.children),
  ]);
}

export function countTokenCommentTreeNodes(
  nodes: TokenCommentTreeNode[]
): number {
  return nodes.reduce(
    (total, node) => total + 1 + countTokenCommentTreeNodes(node.children),
    0
  );
}

export function hasTokenCommentChildren(node: TokenCommentTreeNode): boolean {
  return node.children.length > 0 || node.comment.childCount > 0;
}