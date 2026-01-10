// frontend\mall\lib\app\shell\presentation\components\header.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// Minimal public header for Mall (no-auth).
/// - Back button (optional) ※ showBack のみで制御
/// - Title (optional)
/// - Right-side actions (optional)  ← Sign in / Sign out は router 側から渡す
///
/// ✅ NOTE:
/// - 戻る遷移時に、現在のURLが持つ query（例: avatarId）を必要に応じて引き継げるようにする。
class AppHeader extends StatelessWidget {
  const AppHeader({
    super.key,
    this.title,
    this.showBack = true,
    this.onTapTitle,
    this.actions,
    this.backTo = '/',
    this.preserveQueryKeys = const ['avatarId'],
  });

  final String? title;

  /// ✅ showBack=true の時だけ「戻る」を表示する
  final bool showBack;

  /// Optional callback when title is tapped (e.g., navigate to home).
  final VoidCallback? onTapTitle;

  /// Optional action widgets on the right side.
  final List<Widget>? actions;

  /// ✅ 「戻る」押下時の遷移先（popは使わず go で戻す）
  final String backTo;

  /// ✅ backTo に query が無い場合でも、現在の query の一部を引き継ぐ
  /// - 例: avatarId を維持して遷移
  final List<String> preserveQueryKeys;

  Uri _mergePreserveQuery(BuildContext context, String to) {
    final current = GoRouterState.of(context).uri;

    final dest = Uri.parse(to);
    final merged = <String, String>{...dest.queryParameters};

    for (final k in preserveQueryKeys) {
      if (merged.containsKey(k)) continue;
      final v = (current.queryParameters[k] ?? '').trim();
      if (v.isNotEmpty) merged[k] = v;
    }

    return dest.replace(queryParameters: merged);
  }

  void _handleBack(BuildContext context) {
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
