// frontend/mall/lib/features/wallet/presentation/component/token_card.dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/app_grid_card.dart';
import '../../infrastructure/token_metadata_dto.dart';
import '../../infrastructure/token_resolve_dto.dart';

class TokenCard extends StatefulWidget {
  const TokenCard({
    super.key,
    required this.mintAddress,
    this.resolved,
    this.metadata,
    this.isLoading = false,
  });

  final String mintAddress;
  final TokenResolveDTO? resolved;
  final TokenMetadataDTO? metadata;
  final bool isLoading;

  @override
  State<TokenCard> createState() => _TokenCardState();
}

class _TokenCardState extends State<TokenCard> {
  static const _timeout = Duration(seconds: 30);

  Timer? _timer;
  bool _timedOut = false;

  @override
  void initState() {
    super.initState();
    _syncTimer();
  }

  @override
  void didUpdateWidget(covariant TokenCard oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.isLoading != widget.isLoading) {
      _syncTimer(resetTimeoutWhenLoaded: true);
    }
  }

  void _syncTimer({bool resetTimeoutWhenLoaded = false}) {
    if (!widget.isLoading) {
      _timer?.cancel();
      _timer = null;

      if (resetTimeoutWhenLoaded && _timedOut) {
        setState(() => _timedOut = false);
      }
      return;
    }

    if (_timer == null && !_timedOut) {
      _timer = Timer(_timeout, () {
        if (!mounted) return;
        setState(() => _timedOut = true);
      });
    }
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  String _s(dynamic v) => (v == null ? '' : v.toString()).trim();

  String _encodeFrom(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return '';
    return base64UrlEncode(utf8.encode(s));
  }

  /// ✅ URL肥大化の根本原因：from の中に from が入れ子になる
  /// →「現在URLから from を必ず除去してから」from を作る
  ///
  /// ✅ 追加（セキュリティ要件）:
  /// - avatarId / mintAddress を URL に残したくないので、from にも入れない
  String _sanitizedCurrentLocationForFrom(BuildContext context) {
    final uri = GoRouterState.of(context).uri;

    // queryParameters は「最後の値」しか取れないが、from入れ子の抑止には十分
    final qp = Map<String, String>.from(uri.queryParameters);

    // ★最重要★
    qp.remove(AppQueryKey.from);

    // ✅ セキュリティ: URLに出したくないキーは from にも残さない
    qp.remove(AppQueryKey.avatarId);
    qp.remove(AppQueryKey.mintAddress);

    // 余計な fragment がある場合も外す
    final sanitized = uri.replace(
      queryParameters: qp.isEmpty ? null : qp,
      fragment: null,
    );
    return sanitized.toString();
  }

  void _openContents(BuildContext context) {
    final mint = widget.mintAddress.trim();
    if (mint.isEmpty) return;

    // ✅ from は「戻り先」用途のみ。入れ子＆機微情報を除去した URL を base64 化する
    final sanitizedFrom = _sanitizedCurrentLocationForFrom(context);
    final from = _encodeFrom(sanitizedFrom);

    // ✅ mintAddress は URL に出さない（router 側は extra を読む）
    // ✅ avatarId も URL に出さない（navigation.dart が store で保持）
    context.pushNamed(
      AppRouteName.walletContents,
      queryParameters: {if (from.isNotEmpty) AppQueryKey.from: from},
      extra: mint,
    );
  }

  @override
  Widget build(BuildContext context) {
    final resolvedOk = widget.resolved != null;
    final metadataOk = widget.metadata != null;

    final failed = !widget.isLoading && (!resolvedOk || !metadataOk);
    final timeoutFailed = widget.isLoading && _timedOut;

    final canTap = !widget.isLoading && !failed && !timeoutFailed;

    return AppGridCard(
      // TokenCard は grid で使う前提なので margin は呼び出し側が付ける想定
      onTap: canTap ? () => _openContents(context) : null,
      padding: const EdgeInsets.all(10),
      child: LayoutBuilder(
        builder: (context, constraints) {
          if (timeoutFailed) {
            return _fixedCenterMessage(context, 'データを取得できませんでした。');
          }

          if (widget.isLoading) {
            return _fixedSkeleton(context, constraints.maxHeight);
          }

          if (failed) {
            return _fixedCenterMessage(context, 'データを取得できませんでした。');
          }

          final brandName = _s(widget.resolved?.brandName);
          final tokenName = _s(widget.metadata?.name);
          final productName = _s(widget.resolved?.productName);
          final imageUrl = _s(widget.metadata?.image);

          final cs = Theme.of(context).colorScheme;

          final titleStyle = Theme.of(context).textTheme.titleSmall?.copyWith(
            color: cs.onSurface,
            fontWeight: FontWeight.w700,
            fontSize: 12.5,
            height: 1.1,
          );

          final subStyle = Theme.of(context).textTheme.bodySmall?.copyWith(
            color: cs.onSurfaceVariant,
            fontWeight: FontWeight.w700,
            fontSize: 11.5,
            height: 1.1,
          );

          final titleLineH =
              ((titleStyle?.fontSize ?? 12.5) * (titleStyle?.height ?? 1.1));
          final subLineH =
              ((subStyle?.fontSize ?? 11.5) * (subStyle?.height ?? 1.1));

          const gap1 = 6.0;
          const gap2 = 6.0;
          const gap3 = 4.0;

          final reservedTextHeight =
              titleLineH + gap1 + titleLineH + gap3 + (subLineH * 2) + gap2;

          final imageH = (constraints.maxHeight - reservedTextHeight).clamp(
            56.0,
            9999.0,
          );

          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              _textLine(
                context,
                brandName.isEmpty ? '（brandName）' : brandName,
                style: titleStyle,
                maxLines: 1,
              ),
              const SizedBox(height: gap1),
              SizedBox(height: imageH, child: _imageBox(context, imageUrl)),
              const SizedBox(height: gap2),
              _textLine(
                context,
                tokenName.isEmpty ? '（token name）' : tokenName,
                style: titleStyle,
                maxLines: 1,
              ),
              const SizedBox(height: gap3),
              _textLine(
                context,
                productName.isEmpty ? '（productName）' : productName,
                style: subStyle,
                maxLines: 2,
              ),
            ],
          );
        },
      ),
    );
  }

  Widget _fixedCenterMessage(BuildContext context, String msg) {
    final cs = Theme.of(context).colorScheme;
    return SizedBox.expand(
      child: Center(
        child: Text(
          msg,
          textAlign: TextAlign.center,
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
            color: cs.error,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
    );
  }

  Widget _fixedSkeleton(BuildContext context, double maxH) {
    final cs = Theme.of(context).colorScheme;
    final base = cs.surface;
    final br = BorderRadius.circular(10);

    Widget box({required double h}) {
      return Container(
        height: h,
        decoration: BoxDecoration(color: base, borderRadius: br),
      );
    }

    return SizedBox.expand(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          box(h: 14),
          const SizedBox(height: 6),
          Expanded(
            child: Container(
              decoration: BoxDecoration(color: base, borderRadius: br),
            ),
          ),
          const SizedBox(height: 6),
          box(h: 14),
          const SizedBox(height: 4),
          box(h: 12),
        ],
      ),
    );
  }

  Widget _textLine(
    BuildContext context,
    String value, {
    required TextStyle? style,
    required int maxLines,
  }) {
    return Text(
      value.trim().isEmpty ? '（空）' : value.trim(),
      maxLines: maxLines,
      overflow: TextOverflow.ellipsis,
      softWrap: true,
      style: style,
    );
  }

  Widget _imageBox(BuildContext context, String url) {
    final cs = Theme.of(context).colorScheme;
    final u = url.trim();

    return ClipRRect(
      borderRadius: BorderRadius.circular(10),
      child: Container(
        color: cs.surface,
        child: u.isEmpty
            ? Center(
                child: Text(
                  'no image',
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: cs.onSurfaceVariant,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              )
            : Image.network(
                u,
                fit: BoxFit.cover,
                loadingBuilder: (context, child, loadingProgress) {
                  if (loadingProgress == null) return child;
                  return const Center(
                    child: SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    ),
                  );
                },
                errorBuilder: (context, error, stackTrace) {
                  return Center(
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Text(
                        '画像を読み込めませんでした。',
                        style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: cs.onSurfaceVariant,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                  );
                },
              ),
      ),
    );
  }
}
