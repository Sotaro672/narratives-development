// frontend/mall/lib/shared/presentation/component/app_fixed_extent_grid.dart
import 'package:flutter/material.dart';

/// ✅ スマホ幅前提の「3列グリッド + 高さ固定」共通コンポーネント
/// - avatar.dart の TokenCards と同じ計算ロジックを共通化
/// - mainAxisExtent を固定して、グリッド整合性を保つ
class AppFixedExtentGrid extends StatelessWidget {
  const AppFixedExtentGrid({
    super.key,
    required this.itemCount,
    required this.itemBuilder,
    this.crossAxisCount = 3,
    this.spacing = 10.0,
    this.childAspectRatio = 0.82,
    this.extraTextLines = 1,
    this.extraTextPadding = 4.0,
    this.shrinkWrap = true,
    this.physics = const NeverScrollableScrollPhysics(),
    this.padding = EdgeInsets.zero,
  });

  final int itemCount;
  final IndexedWidgetBuilder itemBuilder;

  /// ✅ 既定: 3列
  final int crossAxisCount;

  /// ✅ 既定: 10
  final double spacing;

  /// ✅ 既定: 0.82（avatar.dart の既存）
  final double childAspectRatio;

  /// ✅ 高さに足す「文字列行数」（avatar.dart は 1 行分足していた）
  final int extraTextLines;

  /// ✅ 文字列行数分の高さに加えて足す固定余白
  final double extraTextPadding;

  final bool shrinkWrap;
  final ScrollPhysics physics;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final itemWidth =
            (constraints.maxWidth - spacing * (crossAxisCount - 1)) /
            crossAxisCount;

        // height = width / aspectRatio
        final baseHeight = itemWidth / childAspectRatio;

        // ✅ 文字列1行分だけ増やす（フォントサイズに追従）
        final fs = Theme.of(context).textTheme.bodySmall?.fontSize ?? 12.0;
        final oneLine = fs * 1.35;

        final extra = (oneLine * extraTextLines) + extraTextPadding;
        final fixedHeight = baseHeight + extra;

        return GridView.builder(
          shrinkWrap: shrinkWrap,
          physics: physics,
          padding: padding,
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: crossAxisCount,
            crossAxisSpacing: spacing,
            mainAxisSpacing: spacing,
            mainAxisExtent: fixedHeight,
          ),
          itemCount: itemCount,
          itemBuilder: itemBuilder,
        );
      },
    );
  }
}
