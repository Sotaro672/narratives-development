// frontend/mall/lib/features/preview/presentation/preview.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';

import '../infrastructure/repository.dart';

/// Preview page (buyer-facing).
///
/// - /mall/preview     : ログイン前ユーザーがスキャンした時に叩く（public）
/// - /mall/me/preview  : ログイン後ユーザーがスキャンした時に叩く（auth）
///
/// 現時点ではまず productId -> modelId を表示できればOK。
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

  late final PreviewRepositoryHttp _repo;

  Future<MallPreviewResponse?>? _previewFuture;

  @override
  void initState() {
    super.initState();

    _repo = PreviewRepositoryHttp();

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

    // ✅ productId がある場合のみ preview を取りに行く
    if (productId.isNotEmpty) {
      _previewFuture = _loadPreview(productId);
    }
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

      // ✅ productId が更新されたら取り直す
      if (productId.isNotEmpty) {
        setState(() {
          _previewFuture = _loadPreview(productId);
        });
      }
    }
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  Future<MallPreviewResponse?> _loadPreview(String productId) async {
    final id = productId.trim();
    if (id.isEmpty) return null;

    final user = FirebaseAuth.instance.currentUser;

    // ログイン前 -> public
    if (user == null) {
      debugPrint('[PreviewPage] calling PUBLIC /mall/preview productId=$id');
      final r = await _repo.fetchPreviewByProductId(id);
      debugPrint(
        '[PreviewPage] PUBLIC response productId=${r.productId} modelId=${r.modelId}'
        ' modelNumber=${r.modelNumber.isEmpty ? "-" : r.modelNumber}'
        ' size=${r.size.isEmpty ? "-" : r.size}'
        ' color=${r.color.isEmpty ? "-" : r.color}'
        ' rgb=${r.rgb}'
        ' measurements=${r.measurements}',
      );
      return r;
    }

    // ログイン後 -> me
    final token = await user.getIdToken();

    debugPrint(
      '[PreviewPage] calling ME /mall/me/preview productId=$id tokenLen=${token?.length ?? 0}',
    );

    final r = await _repo.fetchMyPreviewByProductId(
      id,
      headers: {'Authorization': 'Bearer ${token ?? ''}'},
    );
    debugPrint(
      '[PreviewPage] ME response productId=${r.productId} modelId=${r.modelId}'
      ' modelNumber=${r.modelNumber.isEmpty ? "-" : r.modelNumber}'
      ' size=${r.size.isEmpty ? "-" : r.size}'
      ' color=${r.color.isEmpty ? "-" : r.color}'
      ' rgb=${r.rgb}'
      ' measurements=${r.measurements}',
    );
    return r;
  }

  /// ✅ int(0xRRGGBB) -> Flutter Color(0xAARRGGBB)
  Color _rgbToColor(int rgb) {
    final v = rgb & 0xFFFFFF;
    return Color(0xFF000000 | v);
  }

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;
    final productId = _productId;
    final from = (widget.from ?? '').trim();

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
                  // ✅ 「プレビュー」文字を削除
                  const SizedBox(height: 2),
                  Text(
                    '商品ID: ${productId.isEmpty ? '-' : productId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'アバターID: ${avatarId.isEmpty ? '-' : avatarId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text('遷移元: ${from.isEmpty ? '-' : from}', style: t.bodySmall),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: FutureBuilder<MallPreviewResponse?>(
                future: _previewFuture,
                builder: (context, snap) {
                  if (productId.isEmpty) {
                    return const Text('商品ID が無いため、プレビューを取得しません。');
                  }

                  if (snap.connectionState == ConnectionState.waiting) {
                    return const Row(
                      children: [
                        SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                        SizedBox(width: 10),
                        Text('プレビューを取得しています...'),
                      ],
                    );
                  }

                  if (snap.hasError) {
                    return Text(
                      'プレビュー取得に失敗しました: ${snap.error}',
                      style: t.bodySmall,
                    );
                  }

                  final data = snap.data;

                  final modelNumber = (data?.modelNumber ?? '').trim();
                  final size = (data?.size ?? '').trim();
                  final colorName = (data?.color ?? '').trim();
                  final rgb = data?.rgb ?? 0;
                  final measurements = data?.measurements;

                  // ✅ rgb -> Color（表示用スウォッチ）
                  final swatch = _rgbToColor(rgb);

                  final border = Border.all(
                    color: Theme.of(context).dividerColor,
                    width: 1,
                  );

                  final measurementEntries =
                      (measurements ?? {}).entries
                          .where((e) => e.key.trim().isNotEmpty)
                          .toList()
                        ..sort((a, b) => a.key.compareTo(b.key));

                  // ✅ 各採寸を横並びにする（折り返しあり）
                  final measurementChips = measurementEntries.map((e) {
                    return Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 10,
                        vertical: 6,
                      ),
                      decoration: BoxDecoration(
                        border: border,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Text('${e.key}: ${e.value}', style: t.bodySmall),
                    );
                  }).toList();

                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // ✅ タイトルを「商品情報」に変更
                      Text('商品情報', style: t.titleSmall),
                      const SizedBox(height: 8),
                      Text(
                        '型番: ${modelNumber.isEmpty ? '-' : modelNumber}',
                        style: t.bodySmall,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        'サイズ: ${size.isEmpty ? '-' : size}',
                        style: t.bodySmall,
                      ),
                      const SizedBox(height: 4),
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.center,
                        children: [
                          Flexible(
                            child: Text(
                              '色名: ${colorName.isEmpty ? '-' : colorName}',
                              style: t.bodySmall,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Container(
                            width: 18,
                            height: 18,
                            decoration: BoxDecoration(
                              color: swatch,
                              borderRadius: BorderRadius.circular(4),
                              border: border,
                            ),
                          ),
                        ],
                      ),

                      // ✅ 採寸（measurements）を Column 内に配置し、各採寸は横並び
                      if (measurementChips.isNotEmpty) ...[
                        const SizedBox(height: 10),
                        Text('採寸', style: t.bodySmall),
                        const SizedBox(height: 6),
                        Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          children: measurementChips,
                        ),
                      ],
                    ],
                  );
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
