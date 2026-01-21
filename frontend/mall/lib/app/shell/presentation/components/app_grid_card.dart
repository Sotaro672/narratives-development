// frontend/mall/lib/shared/presentation/component/app_grid_card.dart
import 'package:flutter/material.dart';

/// ✅ Grid / List のカード表現を共通化するための薄いラッパー
/// - 見た目（角丸・背景・elevation）
/// - InkWell のタップ領域
/// - padding / margin
/// を共通化し、子のレイアウトは呼び出し元に委譲する。
class AppGridCard extends StatelessWidget {
  const AppGridCard({
    super.key,
    required this.child,
    this.onTap,
    this.padding = const EdgeInsets.all(10),
    this.margin,
    this.borderRadius = 12,
    this.elevation = 0,
  });

  final Widget child;
  final VoidCallback? onTap;
  final EdgeInsetsGeometry padding;
  final EdgeInsetsGeometry? margin;
  final double borderRadius;
  final double elevation;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    return Card(
      margin: margin,
      elevation: elevation,
      color: cs.surfaceContainerHighest,
      clipBehavior: Clip.antiAlias,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(borderRadius),
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          borderRadius: BorderRadius.circular(borderRadius),
          onTap: onTap,
          child: Padding(padding: padding, child: child),
        ),
      ),
    );
  }
}
