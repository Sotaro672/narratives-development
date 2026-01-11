// frontend\mall\lib\features\preview\presentation\preview.dart
import 'package:flutter/material.dart';

/// Preview page (buyer-facing).
///
/// 現時点では「ルート追加のための受け皿」として、
/// avatarId / productId / from を表示する最小実装にしています。
/// preview_query の表示ロジックは、このページに後から足していけます。
class PreviewPage extends StatefulWidget {
  const PreviewPage({
    super.key,
    required this.avatarId,
    this.productId,
    this.from,
  });

  final String avatarId;

  /// ✅ QR入口（https://narratives.jp/{productId}）や
  /// ✅ /preview?productId=... から渡される商品ID
  final String? productId;

  final String? from;

  @override
  State<PreviewPage> createState() => _PreviewPageState();
}

class _PreviewPageState extends State<PreviewPage> {
  String get _avatarId => widget.avatarId.trim();
  String get _productId => (widget.productId ?? '').trim();

  @override
  void initState() {
    super.initState();

    // ✅ ページ到達ログ（QRスキャンで遷移してきたかの確認に使う）
    final avatarId = _avatarId;
    final productId = _productId;
    final from = (widget.from ?? '').trim();

    debugPrint(
      '[PreviewPage] mounted'
      ' productId=${productId.isEmpty ? "-" : productId}'
      ' avatarId=${avatarId.isEmpty ? "-" : avatarId}'
      ' from=${from.isEmpty ? "-" : from}',
    );

    // 追加で「次フレームで context/route を見たい」場合（必要なら）
    WidgetsBinding.instance.addPostFrameCallback((_) {
      final routeName = ModalRoute.of(context)?.settings.name;
      debugPrint('[PreviewPage] route=${routeName ?? "-"} uri=${Uri.base}');
    });
  }

  @override
  void didUpdateWidget(covariant PreviewPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    // ✅ クエリやパラメータ更新で widget が差し替わった時も追跡できるように
    if (oldWidget.avatarId != widget.avatarId ||
        oldWidget.productId != widget.productId ||
        oldWidget.from != widget.from) {
      final avatarId = _avatarId;
      final productId = _productId;
      final from = (widget.from ?? '').trim();

      debugPrint(
        '[PreviewPage] updated'
        ' productId=${productId.isEmpty ? "-" : productId}'
        ' avatarId=${avatarId.isEmpty ? "-" : avatarId}'
        ' from=${from.isEmpty ? "-" : from}',
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;
    final productId = _productId;
    final from = (widget.from ?? '').trim();

    // ✅ ここは “必須” から外す（QRから入った直後は未ログインで avatarId が空になり得る）
    final t = Theme.of(context).textTheme;

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Preview', style: t.titleMedium),
                  const SizedBox(height: 8),
                  Text(
                    'productId: ${productId.isEmpty ? '-' : productId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'avatarId: ${avatarId.isEmpty ? '-' : avatarId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'from: ${from.isEmpty ? '-' : from}',
                    style: t.bodySmall,
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          const Card(
            child: Padding(
              padding: EdgeInsets.all(14),
              child: Text('ここに preview_query の結果を表示します。'),
            ),
          ),
        ],
      ),
    );
  }
}
