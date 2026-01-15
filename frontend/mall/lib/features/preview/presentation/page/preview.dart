// frontend/mall/lib/features/preview/presentation/page/preview.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import '../../../../app/config/api_base.dart'; // ✅ resolveMallApiBase()
import '../../infrastructure/repository.dart';

/// Preview page (buyer-facing).
///
/// - /mall/preview     : ログイン前ユーザーがスキャンした時に叩く（public）
/// - /mall/me/preview  : ログイン後ユーザーがスキャンした時に叩く（auth）
///
/// ✅ 正攻法:
/// - ログイン中は URL 等で渡ってきた avatarId を信用せず、
///   /mall/me/avatar で自分の avatarId を解決してから verify/transfer を行う。
/// - verify.matched == true の場合のみ transfer を自動実行（多重実行防止あり）
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
  String get _incomingAvatarId => widget.avatarId.trim();
  String get _productId => (widget.productId ?? '').trim();

  late final PreviewRepositoryHttp _previewRepo;
  late final ScanVerifyRepositoryHttp _scanVerifyRepo;
  late final ScanTransferRepositoryHttp _scanTransferRepo;
  late final MeAvatarRepositoryHttp _meAvatarRepo;

  Future<MallPreviewResponse?>? _previewFuture;

  // me avatar -> verify -> transfer の順に進めるため、結果は State に保持する
  String? _meAvatarId; // 解決できたときだけ入る
  MallScanVerifyResponse? _verifyResult;
  MallScanTransferResponse? _transferResult;

  Object? _meAvatarError;
  Object? _verifyError;
  Object? _transferError;

  bool _busyMe = false;
  bool _busyVerify = false;
  bool _busyTransfer = false;

  // 多重実行防止（verify/transfer）
  bool _transferTriggered = false;

  /// firebase_auth の環境差（getIdToken() が String? 扱いになる等）を吸収して
  /// 常に non-null の String を返す
  Future<String> _idTokenOrEmpty(User user) async {
    try {
      final t = await user.getIdToken();
      return (t ?? '').toString();
    } catch (_) {
      return '';
    }
  }

  @override
  void initState() {
    super.initState();

    _previewRepo = PreviewRepositoryHttp();
    _scanVerifyRepo = ScanVerifyRepositoryHttp();
    _scanTransferRepo = ScanTransferRepositoryHttp();
    _meAvatarRepo = MeAvatarRepositoryHttp();

    final productId = _productId;

    if (productId.isNotEmpty) {
      _previewFuture = _loadPreview(productId);
      _kickAuthFlowIfNeeded();
    }
  }

  @override
  void didUpdateWidget(covariant PreviewPage oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (oldWidget.avatarId != widget.avatarId ||
        oldWidget.productId != widget.productId ||
        oldWidget.from != widget.from) {
      final productId = _productId;

      setState(() {
        _previewFuture = productId.isNotEmpty ? _loadPreview(productId) : null;

        // 状態リセット（商品が変わったらやり直す想定）
        _meAvatarId = null;
        _verifyResult = null;
        _transferResult = null;

        _meAvatarError = null;
        _verifyError = null;
        _transferError = null;

        _busyMe = false;
        _busyVerify = false;
        _busyTransfer = false;

        _transferTriggered = false;
      });

      if (productId.isNotEmpty) {
        _kickAuthFlowIfNeeded();
      }
    }
  }

  @override
  void dispose() {
    _previewRepo.dispose();
    _scanVerifyRepo.dispose();
    _scanTransferRepo.dispose();
    _meAvatarRepo.dispose();
    super.dispose();
  }

  // ----------------------------
  // Preview
  // ----------------------------
  Future<MallPreviewResponse?> _loadPreview(String productId) async {
    final id = productId.trim();
    if (id.isEmpty) return null;

    final user = FirebaseAuth.instance.currentUser;

    if (user == null) {
      final r = await _previewRepo.fetchPreviewByProductId(id);
      return r;
    }

    final token = await _idTokenOrEmpty(user);

    final r = await _previewRepo.fetchMyPreviewByProductId(
      id,
      headers: {'Authorization': 'Bearer $token'},
    );

    return r;
  }

  // ----------------------------
  // Auth Flow (me avatar -> verify -> transfer)
  // ----------------------------
  Future<void> _kickAuthFlowIfNeeded() async {
    final productId = _productId;
    if (productId.isEmpty) return;

    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    // すでに me avatar が取れてるなら次へ
    final current = (_meAvatarId ?? '').trim();
    if (current.isNotEmpty) {
      await _verifyAndMaybeTransfer();
      return;
    }

    await _resolveMeAvatarId();
    await _verifyAndMaybeTransfer();
  }

  Future<void> _resolveMeAvatarId() async {
    if (_busyMe) return;

    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    setState(() {
      _busyMe = true;
      _meAvatarError = null;
    });

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _meAvatarRepo.fetchMeAvatar(
        headers: {'Authorization': 'Bearer $token'},
      );

      final meAvatarId = r.avatarId.trim();

      if (mounted) {
        setState(() {
          _meAvatarId = meAvatarId.isEmpty ? null : meAvatarId;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _meAvatarError = e;
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _busyMe = false;
        });
      }
    }
  }

  Future<void> _verifyAndMaybeTransfer() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    final productId = _productId.trim();
    final meAvatarId = (_meAvatarId ?? '').trim();
    if (productId.isEmpty || meAvatarId.isEmpty) return;

    // verify が完了済みなら transfer 判定だけやる
    if (_verifyResult != null) {
      await _maybeAutoTransfer();
      return;
    }

    if (_busyVerify) return;

    setState(() {
      _busyVerify = true;
      _verifyError = null;
    });

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _scanVerifyRepo.verifyScanPurchasedByAvatarId(
        avatarId: meAvatarId,
        productId: productId,
        headers: {'Authorization': 'Bearer $token'},
      );

      if (mounted) {
        setState(() {
          _verifyResult = r;
        });
      }

      await _maybeAutoTransfer();
    } catch (e) {
      if (mounted) {
        setState(() {
          _verifyError = e;
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _busyVerify = false;
        });
      }
    }
  }

  Future<void> _maybeAutoTransfer() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) return;

    final productId = _productId.trim();
    final meAvatarId = (_meAvatarId ?? '').trim();
    final verify = _verifyResult;

    if (productId.isEmpty || meAvatarId.isEmpty || verify == null) return;
    if (!verify.matched) return;

    // ✅ 多重実行防止
    if (_transferTriggered || _transferResult != null || _busyTransfer) return;
    _transferTriggered = true;

    setState(() {
      _busyTransfer = true;
      _transferError = null;
    });

    try {
      final token = await _idTokenOrEmpty(user);

      final r = await _scanTransferRepo.transferScanPurchasedByAvatarId(
        avatarId: meAvatarId,
        productId: productId,
        headers: {'Authorization': 'Bearer $token'},
      );

      if (mounted) {
        setState(() {
          _transferResult = r;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _transferError = e;
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _busyTransfer = false;
        });
      }
    }
  }

  // ----------------------------
  // UI helpers
  // ----------------------------
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

    if (brandId.isNotEmpty) return brandId;
    if (avatarId.isNotEmpty) return avatarId;

    return '-';
  }

  Widget _verifyBadge(BuildContext context, MallScanVerifyResponse r) {
    final t = Theme.of(context).textTheme;
    final bodySmall = t.bodySmall ?? const TextStyle(fontSize: 12);

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
        Text(label, style: bodySmall),
        const SizedBox(height: 4),
        Text(detail, style: bodySmall),
      ],
    );
  }

  Widget _transferBadge(BuildContext context, MallScanTransferResponse r) {
    final t = Theme.of(context).textTheme;

    // ✅ nullableをここで確定させる（以降 copyWith を安全に呼べる）
    final bodySmall = t.bodySmall ?? const TextStyle(fontSize: 12);
    final monoSmall = bodySmall.copyWith(fontFamily: 'monospace');

    final ok = r.matched;
    final label = ok ? 'Transfer 実行済み' : 'Transfer 失敗（不一致）';

    final lines = <String>[];

    final detail = lines.join('\n');

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Transfer', style: t.titleSmall),
        const SizedBox(height: 8),
        Text(label, style: bodySmall),
        const SizedBox(height: 6),
        if (detail.isNotEmpty)
          Text(detail, style: monoSmall)
        else
          Text('-', style: bodySmall),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    final incomingAvatarId = _incomingAvatarId;
    final meAvatarId = (_meAvatarId ?? '').trim();
    final productId = _productId;
    final from = (widget.from ?? '').trim();

    final t = Theme.of(context).textTheme;

    // ✅ nullableをここで確定させる（copyWithを安全に）
    final bodySmall = t.bodySmall ?? const TextStyle(fontSize: 12);
    final monoSmall = bodySmall.copyWith(fontFamily: 'monospace');

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
                    'incoming avatarId(URL): ${incomingAvatarId.isEmpty ? '-' : incomingAvatarId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'me avatarId(API): ${meAvatarId.isEmpty ? '-' : meAvatarId}',
                    style: t.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  Text('遷移元: ${from.isEmpty ? '-' : from}', style: t.bodySmall),
                  const SizedBox(height: 10),
                  if (_busyMe) Text('me avatar 解決中...', style: t.bodySmall),
                  if (_meAvatarError != null)
                    Text(
                      'me avatar 解決に失敗: $_meAvatarError',
                      style: t.bodySmall,
                    ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 12),

          // Preview
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

                  final token = data.token;
                  final ownerId = _ownerLabel(data.owner);
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

                  final tokenPretty = token == null
                      ? ''
                      : _prettyJson(token.toJson());

                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('商品情報', style: t.titleSmall),
                      const SizedBox(height: 8),
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
                          child: Text(pbPatchPretty, style: monoSmall),
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
                            child: Text(tokenPretty, style: monoSmall),
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

          // Verify (auto)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Builder(
                builder: (context) {
                  if (_busyVerify) {
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

                  if (_verifyError != null) {
                    return Text(
                      'Verify に失敗しました: $_verifyError',
                      style: t.bodySmall,
                    );
                  }

                  final r = _verifyResult;
                  if (r == null) {
                    return Text('Verify は未実行です。', style: t.bodySmall);
                  }

                  return _verifyBadge(context, r);
                },
              ),
            ),
          ),

          const SizedBox(height: 12),

          // Transfer (auto)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: Builder(
                builder: (context) {
                  if (_busyTransfer) {
                    return const Row(
                      children: [
                        SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                        SizedBox(width: 10),
                        Text('Transfer を実行しています...'),
                      ],
                    );
                  }

                  if (_transferError != null) {
                    return Text(
                      'Transfer に失敗しました: $_transferError',
                      style: t.bodySmall,
                    );
                  }

                  final r = _transferResult;
                  if (r == null) {
                    final v = _verifyResult;
                    if (v == null) {
                      return Text('Transfer は未実行です。', style: t.bodySmall);
                    }
                    if (!v.matched) {
                      return Text(
                        '未購入（Verify不一致）のため Transfer しません。',
                        style: t.bodySmall,
                      );
                    }
                    return Text('Transfer 実行待ちです。', style: t.bodySmall);
                  }

                  return _transferBadge(context, r);
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// /mall/me/avatar 用（このファイル内で完結させるための最小実装）
class MeAvatarRepositoryHttp {
  MeAvatarRepositoryHttp({http.Client? client})
    : _client = client ?? http.Client();

  final http.Client _client;

  void dispose() {
    _client.close();
  }

  /// GET /mall/me/avatar
  Future<MallOwnerInfo> fetchMeAvatar({
    String? baseUrl,
    Map<String, String>? headers,
  }) async {
    final base = (baseUrl ?? '').trim();

    // ✅ resolveApiBase() ではなく resolveMallApiBase()
    final resolvedBase = base.isNotEmpty ? base : resolveMallApiBase();

    final b = normalizeBaseUrl(resolvedBase);
    final uri = Uri.parse('$b/mall/me/avatar');

    final mergedHeaders = <String, String>{...jsonHeaders()};
    if (headers != null) mergedHeaders.addAll(headers);

    final auth = (mergedHeaders['Authorization'] ?? '').trim();
    if (auth.isEmpty) {
      throw ArgumentError(
        'Authorization header is required for /mall/me/avatar',
      );
    }

    final res = await _client.get(uri, headers: mergedHeaders);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        'fetchMeAvatar failed: ${res.statusCode}',
        url: uri.toString(),
        body: res.body,
      );
    }

    final decoded = jsonDecode(res.body);
    if (decoded is! Map) {
      throw const FormatException('invalid json shape (expected object)');
    }

    return MallOwnerInfo.fromJson(decoded.cast<String, dynamic>());
  }
}
