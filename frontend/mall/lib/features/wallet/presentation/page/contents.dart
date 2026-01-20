// frontend\mall\lib\features\wallet\presentation\page\contents.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart'; // ✅ AvatarIdStore
import '../../infrastructure/repository_http.dart'; // ✅ WalletRepositoryHttp
import '../../infrastructure/token_metadata_dto.dart';
import '../../infrastructure/token_resolve_dto.dart';

/// Wallet token detail page (destination from TokenCard tap).
///
/// ✅ mintAddress だけ渡されても、ここで TokenResolveDTO / TokenMetadataDTO を取得して埋める。
/// ✅ metadata は CORS 回避のため backend の /metadata/proxy 経由（repository 内部）で取得する。
class WalletContentsPage extends StatefulWidget {
  const WalletContentsPage({
    super.key,
    required this.mintAddress,
    this.productId,
    this.brandId,
    this.brandName,
    this.productName,
    this.tokenName,
    this.imageUrl,
    this.from,
  });

  /// mint address (token identifier) - ✅ required
  /// NOTE: 画面には表示しない（保持のみ）
  final String mintAddress;

  /// resolved from backend
  /// NOTE: 画面には表示しない（保持のみ）
  final String? productId;
  final String? brandId;

  /// resolved names
  final String? brandName;
  final String? productName;

  /// metadata name
  final String? tokenName;

  /// metadata image url
  final String? imageUrl;

  /// optional return path (decoded, plain string)
  /// NOTE:
  /// - header側が `?from=` を読んで戻るので、このWidget自身は戻るUIを持たない
  final String? from;

  @override
  State<WalletContentsPage> createState() => _WalletContentsPageState();
}

class _WalletContentsPageState extends State<WalletContentsPage> {
  final WalletRepositoryHttp _repo = WalletRepositoryHttp();

  bool _loading = false;
  String? _error;

  TokenResolveDTO? _resolved;
  TokenMetadataDTO? _metadata;

  String _s(String? v) => (v ?? '').trim();

  @override
  void initState() {
    super.initState();

    final mint = widget.mintAddress.trim();
    if (mint.isEmpty) {
      _error = 'mintAddress is required.';
      return;
    }

    // 画面引数で十分埋まっていれば通信しない
    final hasPrefill =
        _s(widget.productId).isNotEmpty ||
        _s(widget.brandId).isNotEmpty ||
        _s(widget.brandName).isNotEmpty ||
        _s(widget.productName).isNotEmpty ||
        _s(widget.tokenName).isNotEmpty ||
        _s(widget.imageUrl).isNotEmpty;

    if (!hasPrefill) {
      _load();
      return;
    }

    // 重要項目が欠けるなら補完ロード
    final missing =
        _s(widget.brandName).isEmpty ||
        _s(widget.productName).isEmpty ||
        _s(widget.tokenName).isEmpty ||
        _s(widget.imageUrl).isEmpty;

    if (missing) _load();
  }

  @override
  void dispose() {
    _repo.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    if (_loading) return;

    setState(() {
      _loading = true;
      _error = null;
    });

    try {
      final mint = widget.mintAddress.trim();
      if (mint.isEmpty) {
        _setErr('mintAddress is required.');
        return;
      }

      // ✅ avatarId は URL ではなく store から解決（期待値）
      final avatarId = (await AvatarIdStore.I.resolveMyAvatarId() ?? '').trim();
      if (avatarId.isEmpty) {
        _setErr('avatarId could not be resolved.');
        return;
      }

      // 1) resolve（product/brand/metadataUri 等）
      final resolved = await _repo.resolveTokenByMintAddress(mint);
      if (resolved == null) {
        _setErr('Failed to resolve token by mintAddress.');
        return;
      }

      // 2) metadata（proxy 経由は repository が担保）
      TokenMetadataDTO? meta;
      final metaUri = resolved.metadataUri.trim();
      if (metaUri.isNotEmpty) {
        meta = await _repo.fetchTokenMetadata(metaUri);
      }

      if (!mounted) {
        _resolved = resolved;
        _metadata = meta;
        return;
      }

      setState(() {
        _resolved = resolved;
        _metadata = meta;
      });
    } catch (e) {
      _setErr(e.toString());
    } finally {
      if (mounted) {
        setState(() => _loading = false);
      } else {
        _loading = false;
      }
    }
  }

  void _setErr(String msg) {
    final m = msg.trim();
    if (!mounted) {
      _error = m;
      return;
    }
    setState(() => _error = m);
  }

  String _firstNonEmpty(List<String?> xs) {
    for (final v in xs) {
      final s = _s(v);
      if (s.isNotEmpty) return s;
    }
    return '';
  }

  Widget _smallIcon(BuildContext context, String url) {
    final cs = Theme.of(context).colorScheme;
    const double size = 56; // ✅ YouTubeチャンネルアイコン相当のサイズ感

    final u = url.trim();
    return ClipRRect(
      borderRadius: BorderRadius.circular(999),
      child: Container(
        width: size,
        height: size,
        color: cs.surface,
        child: u.isEmpty
            ? Icon(
                Icons.image_not_supported_outlined,
                color: cs.onSurfaceVariant,
                size: 22,
              )
            : Image.network(
                u,
                fit: BoxFit.cover,
                errorBuilder: (_, __, ___) => Icon(
                  Icons.broken_image_outlined,
                  color: cs.onSurfaceVariant,
                  size: 22,
                ),
                loadingBuilder: (context, child, p) {
                  if (p == null) return child;
                  return const Center(
                    child: SizedBox(
                      width: 18,
                      height: 18,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  );
                },
              ),
      ),
    );
  }

  void _goPreviewByProductId(BuildContext context, String productId) {
    final pid = productId.trim();
    if (pid.isEmpty) return;

    // ✅ narratives.jp/{productId} = preview.dart（router.dart の /:productId を踏む）
    // ※ URL に avatarId / mintAddress を載せない（redirect が必要なら補完する）
    context.go('/$pid');
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final resolved = _resolved;
    final meta = _metadata;

    // ① Widget引数 → ② resolved → ③ metadata の順で埋める
    // NOTE: productId/brandId/mintAddress は保持するが画面表示はしない
    final pid = _firstNonEmpty([widget.productId, resolved?.productId]);

    final bname = _firstNonEmpty([widget.brandName, resolved?.brandName]);
    final pname = _firstNonEmpty([widget.productName, resolved?.productName]);

    final tname = _firstNonEmpty([widget.tokenName, meta?.name]);
    final img = _firstNonEmpty([widget.imageUrl, meta?.image]);

    final children = <Widget>[];

    if (_loading) {
      children.add(
        Padding(
          padding: const EdgeInsets.only(bottom: 10),
          child: Row(
            children: const [
              SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              SizedBox(width: 10),
              Text('読み込み中…'),
            ],
          ),
        ),
      );
    }

    final errText = (_error ?? '').trim();
    if (errText.isNotEmpty) {
      children.add(
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.red.shade50,
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: Colors.red.shade200),
          ),
          child: Text(errText, style: TextStyle(color: Colors.red.shade800)),
        ),
      );
      children.add(const SizedBox(height: 12));
    }

    children.add(
      _Card(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _smallIcon(context, img),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start, // ✅ 左寄せ
                children: [
                  if (bname.isNotEmpty)
                    Text(
                      bname,
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    )
                  else
                    Text(
                      '（brandName 未取得）',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),

                  const SizedBox(height: 6),

                  if (tname.isNotEmpty)
                    Text(
                      tname,
                      style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    )
                  else
                    Text(
                      '（トークン名 未取得）',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),

                  const SizedBox(height: 6),

                  // ✅ productName をボタン化して、カード内の左側へ配置
                  if (pname.isNotEmpty)
                    TextButton(
                      onPressed: pid.isEmpty
                          ? null
                          : () => _goPreviewByProductId(context, pid),
                      style: TextButton.styleFrom(
                        padding: EdgeInsets.zero,
                        minimumSize: const Size(0, 0),
                        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                        alignment: Alignment.centerLeft,
                      ),
                      child: Text(
                        pname,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w700,
                          decoration: TextDecoration.underline,
                        ),
                      ),
                    )
                  else
                    Text(
                      '（productName 未取得）',
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: cs.onSurfaceVariant,
                      ),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }
}

class _Card extends StatelessWidget {
  const _Card({required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Card(
      elevation: 0,
      color: cs.surfaceContainerHighest,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
      child: Padding(padding: const EdgeInsets.all(14), child: child),
    );
  }
}
