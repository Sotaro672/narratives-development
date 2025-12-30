// frontend/sns/lib/app/shell/presentation/layout/app_shell.dart
import 'package:flutter/material.dart';

import '../components/footer.dart';
import '../components/header.dart';
import '../components/contents.dart';

/// App-wide shell (Header + Main + Footer).
class AppShell extends StatelessWidget {
  const AppShell({
    super.key,
    required this.child,
    this.title,
    this.showBack = true,
    this.actions,
    this.backgroundColor,

    // ✅ NEW: footer slot
    this.footer,

    // ✅ main customization
    this.mainPadding = const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    this.mainMaxWidth = 960,
    this.mainAlignment = Alignment.topCenter,
    this.mainScrollable = true,
  });

  final Widget child;
  final String? title;
  final bool showBack;
  final List<Widget>? actions;
  final Color? backgroundColor;

  /// ✅ footer widget (ex: SignedInFooter / null / AppFooter)
  final Widget? footer;

  final EdgeInsets mainPadding;
  final double mainMaxWidth;
  final AlignmentGeometry mainAlignment;
  final bool mainScrollable;

  @override
  Widget build(BuildContext context) {
    final bg = backgroundColor ?? Theme.of(context).scaffoldBackgroundColor;

    return Scaffold(
      backgroundColor: bg,
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(title: title, showBack: showBack, actions: actions),

            Expanded(
              child: AppMain(
                padding: mainPadding,
                maxWidth: mainMaxWidth,
                alignment: mainAlignment,
                scrollable: mainScrollable,
                child: child,
              ),
            ),

            // ✅ footer: 渡されなければデフォルト AppFooter
            footer ?? const AppFooter(),
          ],
        ),
      ),
    );
  }
}
