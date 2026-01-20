// frontend\mall\lib\app\shell\presentation\components\header.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../routing/routes.dart';
import '../../../routing/navigation.dart';

/// Minimal header for Mall.
///
/// Pattern B:
/// - URL の `from` には依存しない（decode/restore しない）
/// - 戻り先は「履歴(pop)」を最優先し、履歴がなければ NavStore の returnTo を使う
/// - それもなければ backTo へ
class AppHeader extends StatelessWidget {
  const AppHeader({
    super.key,
    this.title,
    this.showBack = true,
    this.onTapTitle,
    this.actions,

    /// Fallback destination when no history and no NavStore returnTo.
    this.backTo = AppRoutePath.home,

    /// If true, back uses pop first when possible.
    this.preferPop = true,
  });

  final String? title;

  /// showBack=true の時だけ「戻る」を表示する
  final bool showBack;

  /// Optional callback when title is tapped (e.g., navigate to home).
  final VoidCallback? onTapTitle;

  /// Optional action widgets on the right side.
  final List<Widget>? actions;

  /// Fallback destination when there is no history and no NavStore returnTo.
  final String backTo;

  /// Prefer pop() over go() when possible.
  final bool preferPop;

  void _handleBack(BuildContext context) {
    // 1) pop 優先（履歴がある遷移は確実に戻す）
    if (preferPop && context.canPop()) {
      context.pop();
      return;
    }

    // 2) NavStore の returnTo を使う（Pattern B のナビ状態）
    final rt = NavStore.I.consumeReturnTo().trim();
    if (rt.isNotEmpty) {
      context.go(rt);
      return;
    }

    // 3) 最後に backTo
    final b = backTo.trim().isNotEmpty ? backTo.trim() : AppRoutePath.home;
    context.go(b);
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
