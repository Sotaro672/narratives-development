// frontend/sns/lib/app/shell/presentation/components/header.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// Minimal public header for SNS (no-auth).
/// - Back button (optional) ※ showBack のみで制御
/// - Title (optional)
/// - Right-side actions (optional)  ← Sign in / Sign out は router 側から渡す
class AppHeader extends StatelessWidget {
  const AppHeader({
    super.key,
    this.title,
    this.showBack = true,
    this.onTapTitle,
    this.actions,
    this.backTo = '/',
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

  void _handleBack(BuildContext context) {
    context.go(backTo);
  }

  @override
  Widget build(BuildContext context) {
    final t = (title ?? '').trim();
    final titleText = t.isNotEmpty ? t : 'sns';

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
                  '← Back',
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
