// frontend/mall/lib/features/preview/presentation/page/preview.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import '../../../../app/config/api_base.dart'; // ✅ resolveMallApiBase()
import '../../infrastructure/repository.dart';

// ✅ productBlueprintPatch DTO
import '../../infrastructure/product_blueprint_patch_dto.dart';

// ✅ WalletContentsPage のカードを Preview 側で再利用する
import '../../../wallet/presentation/page/contents.dart';

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
    } catch (_) {
      // UI で表示しないため握りつぶします（必要なら logger を入れてください）
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
    } catch (_) {
      // UI で表示しないため握りつぶします（必要なら logger を入れてください）
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
    });

    try {
      final token = await _idTokenOrEmpty(user);

      // ✅ deprecated を使わない：avatarId は server 側で解決される前提
      // POST /mall/me/orders/scan/transfer には { productId } だけを送る想定の API
      final r = await _scanTransferRepo.transferScanPurchased(
        productId: productId,
        headers: {'Authorization': 'Bearer $token'},
      );

      if (mounted) {
        setState(() {
          _transferResult = r;
        });
      }
    } catch (_) {
      // UI で表示しないため握りつぶします（必要なら logger を入れてください）
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

  String _ownerLabel(MallOwnerInfo? owner) {
    if (owner == null) return '-';

    final brandId = owner.brandId.trim();
    final avatarId = owner.avatarId.trim();

    if (brandId.isNotEmpty) return brandId;
    if (avatarId.isNotEmpty) return avatarId;

    return '-';
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

  /// DTO のキーを日本語ラベルへ変換（要望分のみ）
  ///
  /// - productName -> 商品名
  /// - productIdTag.Type -> 商品タグ（末尾キーが Type の時は親キーも見る）
  String _jpLabelForPatchKey(String key) {
    final k = key.trim();
    if (k.isEmpty) return '';

    // ✅ productIdTag.Type 専用（flatten でも対応）
    // 例: "productIdTag.Type" / "x.productIdTag.Type"
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

    // flatten 対応（末尾キー）
    final tail = k.split('.').last;
    final tailNoIndex = tail.replaceAll(RegExp(r'\[\d+\]'), '');

    // ✅ 末尾が Type の場合は 1つ上のキーも見る
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

    return k;
  }

  @override
  Widget build(BuildContext context) {
    final productId = _productId;
    final t = Theme.of(context).textTheme;

    final bodySmall = t.bodySmall ?? const TextStyle(fontSize: 12);
    final border = Border.all(color: Theme.of(context).dividerColor, width: 1);

    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Preview（商品情報のみ表示）
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

                  // ✅ productBlueprintPatch を DTO に組み上げる
                  final pbPatchDto = ProductBlueprintPatchDTO.fromJson(
                    data.productBlueprintPatch,
                  );
                  final pbItems = pbPatchDto.items;

                  // ✅ Token情報表示はしないが、WalletContentsPage を出すため mintAddress は保持
                  final token = data.token;
                  final mintAddress = token == null
                      ? ''
                      : token.mintAddress.trim();

                  final ownerId = _ownerLabel(data.owner);
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
                      Text('所有者: $ownerId', style: t.bodySmall),
                      const SizedBox(height: 10),

                      // ✅ assigneeId 行は削除（非表示キーはスキップ）
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

                      // ✅ Preview埋め込み時の期待値:
                      // - productName ボタンは出さない
                      // - tokenName を押下可能にし、contents.dart へ遷移
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

          // ✅ 「Verify」「Transfer」カードは非表示（削除済み）
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
    if (headers != null) {
      mergedHeaders.addAll(headers);
    }

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
