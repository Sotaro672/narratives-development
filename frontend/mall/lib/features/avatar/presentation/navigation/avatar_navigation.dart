// frontend\mall\lib\features\avatar\presentation\navigation\avatar_navigation.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';

/// `from` は URL で壊れやすいので base64url で安全に運ぶ
String encodeFrom(String raw) {
  final t = raw.trim();
  if (t.isEmpty) return '';
  return base64UrlEncode(utf8.encode(t));
}

/// 現在の画面URL
String currentUri(BuildContext context) =>
    GoRouterState.of(context).uri.toString();

/// `from` が空なら現在URLを採用
String effectiveFrom(BuildContext context, {String? from}) {
  final v = (from ?? '').trim();
  if (v.isNotEmpty) return v;
  return currentUri(context);
}

/// avatarId は URL の query (?avatarId=...) を正とする（あれば）
String resolveAvatarIdFromUrl(BuildContext context) {
  final uri = GoRouterState.of(context).uri;
  return (uri.queryParameters[AppQueryKey.avatarId] ?? '').trim();
}

/// URL に avatarId が無ければ付与して正規化する（多重実行防止は呼び出し側で行う）
///
/// NOTE:
/// - `normalizedUrlOnce` は呼び出し側で useRef / State 等に保持すること。
void ensureAvatarIdInUrl({
  required BuildContext context,
  required String avatarId,
  required bool alreadyNormalized,
  required void Function() markNormalized,
}) {
  if (alreadyNormalized) return;

  final aid = avatarId.trim();
  if (aid.isEmpty) return;

  final state = GoRouterState.of(context);
  final u = state.uri;

  final current = (u.queryParameters[AppQueryKey.avatarId] ?? '').trim();
  if (current == aid) {
    markNormalized();
    return;
  }

  markNormalized();

  WidgetsBinding.instance.addPostFrameCallback((_) {
    if (!context.mounted) return;

    final fixed = <String, String>{...u.queryParameters};
    fixed[AppQueryKey.avatarId] = aid;

    final next = u.replace(queryParameters: fixed);
    context.go(next.toString());
  });
}

/// intent=requireAvatarId かつ from=/cart の場合、avatarId を付与して from に戻る
///
/// NOTE:
/// - `returnedToFromOnce` は呼び出し側で useRef / State 等に保持すること。
void maybeReturnToFrom({
  required BuildContext context,
  required String avatarId,
  required String? from,
  required bool alreadyReturned,
  required void Function() markReturned,
}) {
  if (alreadyReturned) return;

  final aid = avatarId.trim();
  if (aid.isEmpty) return;

  final intent =
      (GoRouterState.of(context).uri.queryParameters[AppQueryKey.intent] ?? '')
          .trim();
  if (intent != 'requireAvatarId') return;

  final rawFrom = (from ?? '').trim();
  if (rawFrom.isEmpty) return;

  final fromUri = Uri.tryParse(rawFrom);
  if (fromUri == null) return;

  if (fromUri.path != '/cart') return;

  final qp = <String, String>{...fromUri.queryParameters};
  qp[AppQueryKey.avatarId] = (qp[AppQueryKey.avatarId] ?? '').trim().isNotEmpty
      ? qp[AppQueryKey.avatarId]!
      : aid;

  final fixedFrom = fromUri.replace(queryParameters: qp).toString();
  markReturned();

  WidgetsBinding.instance.addPostFrameCallback((_) {
    if (!context.mounted) return;
    context.go(fixedFrom);
  });
}

/// AvatarEdit へ遷移（from を base64url 化して付与）
void goToAvatarEdit(BuildContext context) {
  final current = GoRouterState.of(context).uri;

  final qp = <String, String>{AppQueryKey.from: encodeFrom(current.toString())};

  final aid = (current.queryParameters[AppQueryKey.avatarId] ?? '').trim();
  if (aid.isNotEmpty) qp[AppQueryKey.avatarId] = aid;

  final u = Uri(path: AppRoutePath.avatarEdit, queryParameters: qp);
  context.go(u.toString());
}
