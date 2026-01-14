// frontend/mall/lib/features/preview/presentation/page/preview.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';

import '../../infrastructure/repository.dart';

/// Preview page (buyer-facing).
///
/// - /mall/preview     : ログイン前ユーザーがスキャンした時に叩く（public）
/// - /mall/me/preview  : ログイン後ユーザーがスキャンした時に叩く（auth）
///
/// ✅ 自動verify採用:
/// - ログイン中 + avatarId/productId が揃っている場合、
///   preview取得と並行して /mall/me/orders/scan/verify を自動で叩く。
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

  late final PreviewRepositoryHttp _previewRepo;
  late final ScanVerifyRepositoryHttp _scanVerifyRepo;

  Future<MallPreviewResponse?>? _previewFuture;
  Future<MallScanVerifyResponse?>? _verifyFuture;

  @override
  void initState() {
    super.initState();

    _previewRepo = PreviewRepositoryHttp();
    _scanVerifyRepo = ScanVerifyRepositoryHttp();

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

    WidgetsBinding.instance.addPostFrameCallback((_) {
      final routeName = ModalRoute.of(context)?.settings.name;
      debugPrint('[PreviewPage] route=${routeName ?? "-"} uri=${Uri.base}');
    });

    // ✅ productId がある場合のみ preview を取りに行く
    if (productId.isNotEmpty) {
      _previewFuture = _loadPreview(productId);

      // ✅ 自動verify（ログイン中のみ）
      _verifyFuture = _autoVerifyIfNeeded(
        avatarId: avatarId,
        productId: productId,
      );
    }
  }

  @override
  void didUpdateWidget(covariant PreviewPage oldWidget) {
    super.didUpdateWidget(oldWidget);

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

      if (productId.isNotEmpty) {
        setState(() {
          _previewFuture = _loadPreview(productId);
          _verifyFuture = _autoVerifyIfNeeded(
            avatarId: avatarId,
            productId: productId,
          );
        });
      }
    }
  }

  @override
  void dispose() {
    _previewRepo.dispose();
    _scanVerifyRepo.dispose();
    super.dispose();
  }

  Future<MallPreviewResponse?> _loadPreview(String productId) async {
    final id = productId.trim();
    if (id.isEmpty) return null;

    final user = FirebaseAuth.instance.currentUser;

    // ログイン前 -> public
    if (user == null) {
      debugPrint('[PreviewPage] calling PUBLIC /mall/preview productId=$id');
      final r = await _previewRepo.fetchPreviewByProductId(id);
      debugPrint(
        '[PreviewPage] PUBLIC response productId=${r.productId} modelId=${r.modelId}'
        ' modelNumber=${r.modelNumber.isEmpty ? "-" : r.modelNumber}'
        ' size=${r.size.isEmpty ? "-" : r.size}'
        ' color=${r.color.isEmpty ? "-" : r.color}'
        ' rgb=${r.rgb}'
        ' measurements=${r.measurements}'
        ' productBlueprintPatch=${r.productBlueprintPatch}'
        ' token=${r.token?.toJson()}'
        ' owner=${r.owner?.toJson()}',
      );
      return r;
    }

    // ログイン後 -> me
    final token = (await user.getIdToken()) ?? '';

    debugPrint(
      '[PreviewPage] calling ME /mall/me/preview productId=$id tokenLen=${token.length}',
    );

    final r = await _previewRepo.fetchMyPreviewByProductId(
      id,
      headers: {'Authorization': 'Bearer $token'},
    );

    debugPrint(
      '[PreviewPage] ME response productId=${r.productId} modelId=${r.modelId}'
      ' modelNumber=${r.modelNumber.isEmpty ? "-" : r.modelNumber}'
      ' size=${r.size.isEmpty ? "-" : r.size}'
      ' color=${r.color.isEmpty ? "-" : r.color}'
      ' rgb=${r.rgb}'
      ' measurements=${r.measurements}'
      ' productBlueprintPatch=${r.productBlueprintPatch}'
      ' token=${r.token?.toJson()}'
      ' owner=${r.owner?.toJson()}',
    );

    return r;
  }

  /// ✅ 自動verify（ログイン中のみ）
  Future<MallScanVerifyResponse?> _autoVerifyIfNeeded({
    required String avatarId,
    required String productId,
  }) async {
    final aid = avatarId.trim();
    final pid = productId.trim();

    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return null;
    if (aid.isEmpty || pid.isEmpty) return null;

    final token = (await user.getIdToken()) ?? '';

    debugPrint(
      '[PreviewPage] calling AUTO VERIFY /mall/me/orders/scan/verify'
      ' avatarId=$aid productId=$pid tokenLen=${token.length}',
    );

    final r = await _scanVerifyRepo.verifyScanPurchasedByAvatarId(
      avatarId: aid,
      productId: pid,
      headers: {'Authorization': 'Bearer $token'},
    );

    debugPrint('[PreviewPage] AUTO VERIFY response: ${r.toJson()}');
    return r;
  }

  /// ✅ int(0xRRGGBB) -> Flutter Color(0xAARRGGBB)
  Color _rgbToColor(int rgb) {
    final v = rgb & 0xFFFFFF;
    return Color(0xFF000000 | v);
  }

  String _prettyJson(dynamic v) {
    try {
      return const JsonEncoder.withIndent('  ').convert(v);
    } catch (_) {
      return (v ?? '').toString();
    }
  }

  String _ownerLabel(MallOwnerInfo? owner) {
    if (owner == null) return '-';

    final brandId = owner.brandId.trim();
    final avatarId = owner.avatarId.trim();

    // 表示優先: brandId -> avatarId
    if (brandId.isNotEmpty) return brandId;
    if (avatarId.isNotEmpty) return avatarId;

    return '-';
  }

  Widget _verifyBadge(BuildContext context, MallScanVerifyResponse r) {
    final t = Theme.of(context).textTheme;

    final matched = r.matched;
    final match = r.match;

    final label = matched ? '購入済み（一致）' : '未購入（不一致）';
    final detail = match == null
        ? '-'
        : 'modelId=${match.modelId.isEmpty ? "-" : match.modelId}, '
              'tokenBlueprintId=${match.tokenBlueprintId.isEmpty ? "-" : match.tokenBlueprintId}';

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Verify', style: t.titleSmall),
        const SizedBox(height: 8),
        Text(label, style: t.bodySmall),
        const SizedBox(height: 4),
        Text(detail, style: t.bodySmall),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final avatarId = _avatarId;
    final productId = _productId;
    final from = (widget.from ?? '').trim();

    final t = Theme.of(context).textTheme;
    final border = Border.all(color: Theme.of(context).dividerColor, width: 1);

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

          // ----------------------------
          // Preview
          // ----------------------------
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
                  if (data == null) {
                    return Text('プレビューが空です。', style: t.bodySmall);
                  }

                  final modelNumber = data.modelNumber.trim();
                  final size = data.size.trim();
                  final colorName = data.color.trim();
                  final rgb = data.rgb;
                  final measurements = data.measurements;
                  final productBlueprintPatch = data.productBlueprintPatch;

                  // ✅ token info
                  final token = data.token;

                  // ✅ owner info
                  final ownerId = _ownerLabel(data.owner);

                  // ✅ rgb -> Color（表示用スウォッチ）
                  final swatch = _rgbToColor(rgb);

                  final measurementEntries =
                      (measurements ?? {}).entries
                          .where((e) => e.key.trim().isNotEmpty)
                          .toList()
                        ..sort((a, b) => a.key.compareTo(b.key));

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

                  final pbPatchPretty =
                      (productBlueprintPatch == null ||
                          productBlueprintPatch.isEmpty)
                      ? ''
                      : _prettyJson(productBlueprintPatch);

                  final tokenPretty = (token == null)
                      ? ''
                      : _prettyJson(token.toJson());

                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('商品情報', style: t.titleSmall),
                      const SizedBox(height: 8),

                      // ✅ owner 表示（要求: 所有者：{検索結果id}）
                      Text('所有者: $ownerId', style: t.bodySmall),
                      const SizedBox(height: 10),

                      if (pbPatchPretty.isNotEmpty) ...[
                        Text('productBlueprintPatch', style: t.bodySmall),
                        const SizedBox(height: 6),
                        Container(
                          width: double.infinity,
                          padding: const EdgeInsets.all(10),
                          decoration: BoxDecoration(
                            border: border,
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text(
                            pbPatchPretty,
                            style:
                                (t.bodySmall ?? const TextStyle(fontSize: 12))
                                    .copyWith(fontFamily: 'monospace'),
                          ),
                        ),
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

                      const SizedBox(height: 14),
                      Text('Token 情報', style: t.titleSmall),
                      const SizedBox(height: 8),

                      if (token == null) ...[
                        Text('未Mint（token情報なし）', style: t.bodySmall),
                      ] else ...[
                        Text(
                          'brandId: ${token.brandId.isEmpty ? '-' : token.brandId}',
                          style: t.bodySmall,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'toAddress: ${token.toAddress.isEmpty ? '-' : token.toAddress}',
                          style: t.bodySmall,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'metadataUri: ${token.metadataUri.isEmpty ? '-' : token.metadataUri}',
                          style: t.bodySmall,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'mintAddress: ${token.mintAddress.isEmpty ? '-' : token.mintAddress}',
                          style: t.bodySmall,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'onChainTxSignature: ${token.onChainTxSignature.isEmpty ? '-' : token.onChainTxSignature}',
                          style: t.bodySmall,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'mintedAt: ${token.mintedAt.isEmpty ? '-' : token.mintedAt}',
                          style: t.bodySmall,
                        ),
                        if (tokenPretty.isNotEmpty) ...[
                          const SizedBox(height: 10),
                          Text('token (raw)', style: t.bodySmall),
                          const SizedBox(height: 6),
                          Container(
                            width: double.infinity,
                            padding: const EdgeInsets.all(10),
                            decoration: BoxDecoration(
                              border: border,
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: Text(
                              tokenPretty,
                              style:
                                  (t.bodySmall ?? const TextStyle(fontSize: 12))
                                      .copyWith(fontFamily: 'monospace'),
                            ),
                          ),
                        ],
                      ],
                    ],
                  );
                },
              ),
            ),
          ),

          const SizedBox(height: 12),

          // ----------------------------
          // Verify (auto)
          // ----------------------------
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: FutureBuilder<MallScanVerifyResponse?>(
                future: _verifyFuture,
                builder: (context, snap) {
                  if (snap.connectionState == ConnectionState.none &&
                      snap.data == null &&
                      snap.error == null) {
                    return Text('Verify は未実行です。', style: t.bodySmall);
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
                        Text('購入照合（Verify）を確認しています...'),
                      ],
                    );
                  }

                  if (snap.hasError) {
                    return Text(
                      'Verify に失敗しました: ${snap.error}',
                      style: t.bodySmall,
                    );
                  }

                  final r = snap.data;
                  if (r == null) {
                    return Text('Verify は未実行です。', style: t.bodySmall);
                  }

                  return _verifyBadge(context, r);
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
