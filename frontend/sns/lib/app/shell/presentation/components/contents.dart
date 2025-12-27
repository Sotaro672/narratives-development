// frontend\sns\lib\app\shell\presentation\components\contents.dart
import 'package:flutter/material.dart';

/// Main area (between Header and Footer).
/// - Centers content
/// - Constrains max width
/// - Optional scroll
class AppMain extends StatelessWidget {
  const AppMain({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
    this.maxWidth = 960,
    this.alignment = Alignment.topCenter,
    this.scrollable = true,
    this.semanticLabel = 'main',
  });

  final Widget child;

  /// Inner padding for main content.
  final EdgeInsets padding;

  /// Max width constraint.
  final double maxWidth;

  /// Alignment for the main container.
  final AlignmentGeometry alignment;

  /// If true, wraps content in [SingleChildScrollView].
  final bool scrollable;

  /// Semantics label for accessibility/debugging.
  final String semanticLabel;

  @override
  Widget build(BuildContext context) {
    final content = Align(
      alignment: alignment,
      child: ConstrainedBox(
        constraints: BoxConstraints(maxWidth: maxWidth),
        child: Padding(padding: padding, child: child),
      ),
    );

    final body = scrollable ? SingleChildScrollView(child: content) : content;

    return Semantics(container: true, label: semanticLabel, child: body);
  }
}
