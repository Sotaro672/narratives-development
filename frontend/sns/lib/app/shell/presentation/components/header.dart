// frontend/sns/lib/app/shell/presentation/component/header.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// Minimal public header for SNS (no-auth).
/// - Back button (optional)
/// - Title (optional)
/// - Right-side actions (optional)
class AppHeader extends StatelessWidget {
  const AppHeader({
    super.key,
    this.title,
    this.showBack = true,
    this.onTapTitle,
    this.actions,
    this.homePath = '/',
  });

  final String? title;

  /// If true, show back button when we can go back (router/navigator),
  /// otherwise fallback to [homePath].
  final bool showBack;

  /// Optional callback when title is tapped (e.g., navigate to home).
  final VoidCallback? onTapTitle;

  /// Optional action widgets on the right side.
  final List<Widget>? actions;

  /// Fallback destination when we can't pop.
  final String homePath;

  bool _canPop(BuildContext context) {
    // ✅ go_router がいる場合はそちらを優先
    try {
      if (context.canPop()) return true;
    } catch (_) {
      // ignore
    }

    // ✅ Navigator が取れる場合のみ
    final nav = Navigator.maybeOf(context);
    return nav?.canPop() ?? false;
  }

  void _handleBack(BuildContext context) {
    // ✅ まずは pop を試す（go_router）
    try {
      if (context.canPop()) {
        context.pop();
        return;
      }
    } catch (_) {
      // ignore
    }

    // ✅ 次に Navigator
    final nav = Navigator.maybeOf(context);
    if (nav != null && nav.canPop()) {
      nav.maybePop();
      return;
    }

    // ✅ それでも無理なら home へ戻す（直リンク/リロード対策）
    try {
      context.go(homePath);
      return;
    } catch (_) {
      // ignore
    }

    // ✅ 最後の砦：タイトルタップの挙動があればそれ
    onTapTitle?.call();
  }

  @override
  Widget build(BuildContext context) {
    final t = (title ?? '').trim();
    final titleText = t.isNotEmpty ? t : 'sns';

    // ✅ “戻れるか” を判定し、戻れない場合はホームへ戻す挙動にする
    final canPop = _canPop(context);
    final shouldShowBack = showBack;

    return Material(
      color: Theme.of(context).cardColor,
      elevation: 0,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Row(
          children: [
            if (shouldShowBack)
              IconButton(
                tooltip: canPop ? 'Back' : 'Home',
                onPressed: () => _handleBack(context),
                icon: Icon(canPop ? Icons.arrow_back : Icons.home),
              )
            else
              const SizedBox(width: 44),

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
