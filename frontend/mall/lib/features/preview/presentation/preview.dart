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
        '[PreviewPage] PUBLIC response productId=${r.productId} modelId=${r.modelId}',
      );
      return r;
    }

    // ログイン後 -> me
    final token = await user.getIdToken();

    // ✅ 修正案1（おすすめ）: token は nullable 扱いになるため null-safe にする
    debugPrint(
      '[PreviewPage] calling ME /mall/me/preview productId=$id tokenLen=${token?.length ?? 0}',
    );

    final r = await _repo.fetchMyPreviewByProductId(
      id,
      headers: {'Authorization': 'Bearer ${token ?? ''}'},
    );
    debugPrint(
      '[PreviewPage] ME response productId=${r.productId} modelId=${r.modelId}',
    );
    return r;
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

          Card(
            child: Padding(
              padding: const EdgeInsets.all(14),
              child: FutureBuilder<MallPreviewResponse?>(
                future: _previewFuture,
                builder: (context, snap) {
                  // productId が無い場合
                  if (productId.isEmpty) {
                    return const Text('productId が無いため preview を取得しません。');
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
                        Text('preview を取得しています...'),
                      ],
                    );
                  }

                  if (snap.hasError) {
                    return Text(
                      'preview 取得に失敗しました: ${snap.error}',
                      style: t.bodySmall,
                    );
                  }

                  final data = snap.data;
                  final modelId = (data?.modelId ?? '').trim();

                  return Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('preview_query result', style: t.titleSmall),
                      const SizedBox(height: 8),
                      Text(
                        'modelId: ${modelId.isEmpty ? '-' : modelId}',
                        style: t.bodySmall,
                      ),
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
