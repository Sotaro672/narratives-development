// frontend/sns/lib/app/shell/presentation/components/footer.dart
import 'package:flutter/material.dart';

/// Minimal public footer for SNS (no-auth).
/// - SafeArea bottom padding
/// - Optional left/center/right slots
class AppFooter extends StatelessWidget {
  const AppFooter({
    super.key,
    this.left,
    this.center,
    this.right,
    this.padding = const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
  });

  final Widget? left;
  final Widget? center;
  final Widget? right;

  final EdgeInsets padding;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Theme.of(context).cardColor,
      elevation: 0,
      child: SafeArea(
        top: false,
        child: Padding(
          padding: padding,
          child: Row(
            children: [
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerLeft, child: left),
              ),
              Expanded(child: Center(child: center)),
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerRight, child: right),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
