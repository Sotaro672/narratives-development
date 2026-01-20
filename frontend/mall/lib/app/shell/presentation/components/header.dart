// frontend\mall\lib\app\shell\presentation\components\header.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../routing/routes.dart';

/// Minimal public header for Mall (no-auth).
/// - Back button (optional) ※ showBack のみで制御
/// - Title (optional)
/// - Right-side actions (optional)
///
/// ✅ Back behavior (final expectation):
/// 1) If canPop() => pop() (return to previous screen; best for wallet->avatar list)
/// 2) Else if current URL has ?from=... => go(decoded from) (direct link case)
/// 3) Else => go(backTo)
class AppHeader extends StatelessWidget {
  const AppHeader({
    super.key,
    this.title,
    this.showBack = true,
    this.onTapTitle,
    this.actions,

    /// Fallback destination when no history and no `from`.
    this.backTo = '/',

    /// If true, back uses pop first when possible.
    this.preferPop = true,

    /// If true, when navigating to backTo, preserve selected query keys.
    /// NOTE: security requirement suggests keeping this empty by default.
    this.preserveQueryKeys = const <String>[],
  });

  final String? title;

  /// showBack=true の時だけ「戻る」を表示する
  final bool showBack;

  /// Optional callback when title is tapped (e.g., navigate to home).
  final VoidCallback? onTapTitle;

  /// Optional action widgets on the right side.
  final List<Widget>? actions;

  /// Fallback destination when there is no history and no `from`.
  final String backTo;

  /// Prefer pop() over go() when possible.
  final bool preferPop;

  /// Preserve selected query keys when navigating to backTo.
  final List<String> preserveQueryKeys;

  String _decodeFrom(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return '';
    // base64url でない場合も混在するので、失敗したらそのまま返す
    try {
      return utf8.decode(base64Url.decode(s));
    } catch (_) {
      return s;
    }
  }

  Uri _mergePreserveQuery(BuildContext context, String to) {
    final current = GoRouterState.of(context).uri;
    final dest = Uri.parse(to);

    final merged = <String, String>{...dest.queryParameters};

    for (final k in preserveQueryKeys) {
      if (merged.containsKey(k)) continue;
      final v = (current.queryParameters[k] ?? '').trim();
      if (v.isNotEmpty) merged[k] = v;
    }

    return dest.replace(queryParameters: merged.isEmpty ? null : merged);
  }

  void _handleBack(BuildContext context) {
    // 1) pop 優先（wallet->avatar token list を確実に戻す）
    if (preferPop && context.canPop()) {
      context.pop();
      return;
    }

    // 2) 直リンク等で履歴が無い場合: from を優先
    final current = GoRouterState.of(context).uri;
    final decodedFrom = _decodeFrom(current.queryParameters[AppQueryKey.from]);
    final f = decodedFrom.trim();
    if (f.isNotEmpty) {
      context.go(f);
      return;
    }

    // 3) 最後に backTo
    final uri = _mergePreserveQuery(context, backTo);
    context.go(uri.toString());
  }

  @override
  Widget build(BuildContext context) {
    final t = (title ?? '').trim();
    final titleText = t.isNotEmpty ? t : 'Mall';

    return Material(
      color: Theme.of(context).cardColor,
      elevation: 0,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Row(
          children: [
            if (showBack)
              TextButton(
                onPressed: () => _handleBack(context),
                style: TextButton.styleFrom(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 8,
                  ),
                ),
                child: Text(
                  '←',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              )
            else
              const SizedBox(width: 44), // layout stability

            Expanded(
              child: GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: onTapTitle,
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 4),
                  child: Text(
                    titleText,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                ),
              ),
            ),

            if (actions != null) ...actions!,
          ],
        ),
      ),
    );
  }
}
