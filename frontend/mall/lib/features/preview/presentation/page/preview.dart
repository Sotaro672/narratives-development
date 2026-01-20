// frontend/mall/lib/features/preview/presentation/page/preview.dart
import 'package:flutter/material.dart';

import '../../infrastructure/product_blueprint_patch_dto.dart';
import '../../../wallet/presentation/page/contents.dart';

// ✅ NEW: logic moved to hook
import '../hook/use_preview.dart';

class PreviewPage extends StatefulWidget {
  const PreviewPage({
    super.key,
    required this.avatarId,
    this.productId,
    this.from,
  });

  /// URL等から渡ってきた avatarId（表示用/デバッグ用）
  final String avatarId;

  /// QR入口（https://narratives.jp/{productId}）や /preview?productId=... から渡される商品ID
  final String? productId;

  final String? from;

  @override
  State<PreviewPage> createState() => _PreviewPageState();
}

class _PreviewPageState extends State<PreviewPage> {
  late final UsePreviewController _c;

  @override
  void initState() {
    super.initState();
    _c = UsePreviewController();
    _c.init(
      avatarId: widget.avatarId,
      productId: widget.productId,
      from: widget.from,
    );
  }

  @override
  void didUpdateWidget(covariant PreviewPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (oldWidget.avatarId != widget.avatarId ||
        oldWidget.productId != widget.productId ||
        oldWidget.from != widget.from) {
      _c.update(
        avatarId: widget.avatarId,
        productId: widget.productId,
        from: widget.from,
      );
      setState(() {}); // keep rebuild behavior
    }
  }

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  // ----------------------------
  // UI helpers (style-only / view-only)
  // ----------------------------
  Color _rgbToColor(int rgb) {
    final v = rgb & 0xFFFFFF;
    return Color(0xFF000000 | v);
  }

  String _withCm(dynamic v) {
    final s = (v ?? '').toString().trim();
    if (s.isEmpty) return '-';
    if (RegExp(r'\s*cm$', caseSensitive: false).hasMatch(s)) return s;
    return '${s}cm';
  }

  bool _shouldHidePatchKey(String rawKey) {
    final k = rawKey.trim();
    if (k.isEmpty) return true;

    // ✅ hide: assigneeId / brandId（末尾キーでも評価）
    const hide = <String>{'assigneeId', 'brandId'};

    if (hide.contains(k)) return true;

    final tail = k.split('.').last;
    final tailNoIndex = tail.replaceAll(RegExp(r'\[\d+\]'), '');
    if (hide.contains(tailNoIndex)) return true;

    return false;
  }

  /// DTO のキーを日本語ラベルへ変換（現在使われているもののみ）
  String _jpLabelForPatchKey(String key) {
    final k = key.trim();
    if (k.isEmpty) return '';

    if (k.endsWith('productIdTag.Type') || k.contains('productIdTag.Type')) {
      return '商品タグ';
    }

    const exact = <String, String>{
      'fit': 'フィット',
      'weight': '重さ',
      'material': '素材',
      'itemType': 'アイテム',
      'qualityAssurance': '品質保証',
      'productIdTag': '商品タグ',
      'productName': '商品名',
    };

    final hit = exact[k];
    if (hit != null) return hit;

    final tail = k.split('.').last;
    final tailNoIndex = tail.replaceAll(RegExp(r'\[\d+\]'), '');

    if (tailNoIndex == 'Type') {
      final parts = k.split('.');
      if (parts.length >= 2) {
        final parent = parts[parts.length - 2].replaceAll(
          RegExp(r'\[\d+\]'),
          '',
        );
        if (parent == 'productIdTag') return '商品タグ';
      }
    }

    final hit2 = exact[tailNoIndex];
    if (hit2 != null) return hit2;

    return '';
  }

  @override
  Widget build(BuildContext context) {
    final productId = _c.productId;
    final t = Theme.of(context).textTheme;

    final bodySmall = t.bodySmall ?? const TextStyle(fontSize: 12);
    final border = Border.all(color: Theme.of(context).dividerColor, width: 1);

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: FutureBuilder(
                future: _c.previewFuture,
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

                  final data = _c.previewDataFromSnapshot(snap.data);
                  if (data == null) {
                    return Text('プレビューが空です。', style: t.bodySmall);
                  }

                  final modelNumber = data.modelNumber.trim();
                  final size = data.size.trim();
                  final colorName = data.color.trim();
                  final rgb = data.rgb;
                  final measurements = data.measurements;

                  final pbPatchDto = ProductBlueprintPatchDTO.fromJson(
                    data.productBlueprintPatch,
                  );
                  final pbItems = pbPatchDto.items;

                  final token = data.token;
                  final mintAddress = token == null
                      ? ''
                      : token.mintAddress.trim();

                  final ownerLabel = _c.ownerLabel(data.owner);
                  final swatch = _rgbToColor(rgb);

                  final measurementEntries =
                      (measurements ?? {}).entries
                          .where((e) => e.key.trim().isNotEmpty)
                          .toList()
                        ..sort((a, b) => a.key.compareTo(b.key));

                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('商品情報', style: t.titleSmall),
                      const SizedBox(height: 8),
                      Text('所有者: $ownerLabel', style: t.bodySmall),
                      const SizedBox(height: 10),

                      if (pbItems.isNotEmpty) ...[
                        ...pbItems.expand((it) {
                          if (_shouldHidePatchKey(it.key)) {
                            return const <Widget>[];
                          }

                          final label = _jpLabelForPatchKey(it.key).trim();
                          if (label.isEmpty) return const <Widget>[];

                          final value = it.value.trim().isEmpty
                              ? '-'
                              : it.value.trim();

                          return <Widget>[
                            Padding(
                              padding: const EdgeInsets.only(bottom: 4),
                              child: Text('$label: $value', style: bodySmall),
                            ),
                          ];
                        }),
                        const SizedBox(height: 10),
                      ],

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

                      if (measurementEntries.isNotEmpty) ...[
                        const SizedBox(height: 10),
                        Text('採寸', style: t.bodySmall),
                        const SizedBox(height: 6),
                        Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: measurementEntries.map((e) {
                            return Padding(
                              padding: const EdgeInsets.only(bottom: 4),
                              child: Text(
                                '${e.key}: ${_withCm(e.value)}',
                                style: t.bodySmall,
                              ),
                            );
                          }).toList(),
                        ),
                      ],

                      if (mintAddress.isNotEmpty) ...[
                        const SizedBox(height: 12),
                        WalletContentsPage(
                          mintAddress: mintAddress,
                          productId: productId,
                          brandId: token?.brandId.trim(),
                          from: widget.from,
                          enableProductLink: false,
                          enableTokenNameLink: true,
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
