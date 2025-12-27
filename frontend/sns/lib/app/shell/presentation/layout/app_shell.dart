// frontend/sns/lib/app/shell/presentation/layout/app_shell.dart
import 'package:flutter/material.dart';

import '../components/footer.dart';
import '../components/header.dart';
import '../components/contents.dart';

/// App-wide shell (Header + Main + Footer).
/// - Designed for public pages (no auth required)
/// - Main content is provided via [child]
class AppShell extends StatelessWidget {
  const AppShell({
    super.key,
    required this.child,
    this.title,
    this.showBack = true,
    this.actions,
    this.backgroundColor,

    // ✅ main customization
    this.mainPadding = const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    this.mainMaxWidth = 960,
    this.mainAlignment = Alignment.topCenter,
    this.mainScrollable = true,
  });

  /// Main content (page)
  final Widget child;

  /// Optional title shown in the header.
  final String? title;

  /// Show back button in header (actual pop-ability is judged in AppHeader).
  final bool showBack;

  /// Optional header actions (right side)
  final List<Widget>? actions;

  /// Optional background color for the whole page
  final Color? backgroundColor;

  // ------------------------------------------------------------
  // Main props
  // ------------------------------------------------------------

  /// Inner padding for main area.
  final EdgeInsets mainPadding;

  /// Max width for main content.
  final double mainMaxWidth;

  /// Alignment for main container.
  final AlignmentGeometry mainAlignment;

  /// If true, wraps main content in SingleChildScrollView.
  final bool mainScrollable;

  @override
  Widget build(BuildContext context) {
    final bg = backgroundColor ?? Theme.of(context).scaffoldBackgroundColor;

    return Scaffold(
      backgroundColor: bg,
      body: SafeArea(
        child: Column(
          children: [
            // Header
            AppHeader(
              title: title,
              // ✅ ここで canPop 判定して潰さない（ShellRoute だと誤判定しやすい）
              // ✅ 実際に戻れるかどうか & fallback は AppHeader 側で処理する
              showBack: showBack,
              actions: actions,
            ),

            // Main
            Expanded(
              child: AppMain(
                padding: mainPadding,
                maxWidth: mainMaxWidth,
                alignment: mainAlignment,
                scrollable: mainScrollable,
                child: child,
              ),
            ),

            // Footer
            const AppFooter(),
          ],
        ),
      ),
    );
  }
}
