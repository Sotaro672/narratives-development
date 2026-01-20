// frontend\mall\lib\features\wallet\presentation\page\contents.dart
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../../app/config/api_base.dart';
import '../../../../app/routing/navigation.dart'; // ✅ AvatarIdStore
import '../../infrastructure/token_metadata_dto.dart';
import '../../infrastructure/token_resolve_dto.dart';

/// Wallet token detail page (destination from TokenCard tap).
///
/// ✅ mintAddress だけ渡されても、ここで TokenResolveDTO / TokenMetadataDTO を取得して埋める。
/// ✅ metadata は CORS 回避のため backend の /metadata/proxy 経由で取得する。
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

  final String mintAddress;

  final String? productId;
  final String? brandId;

  final String? brandName;
  final String? productName;

  final String? tokenName;
  final String? imageUrl;

  final String? from;

  @override
  State<WalletContentsPage> createState() => _WalletContentsPageState();
}

class _WalletContentsPageState extends State<WalletContentsPage> {
  // ✅ backend 側に実在しそうな resolve 候補を複数持つ（404なら次を試す）
  // ※実際の正解パスが確定したら、候補を1つにしてOK
  static const List<String> _resolveCandidates = <String>[
    '/mall/token/resolve',
    '/mall/me/wallets/resolve',
    '/mall/me/wallets/token/resolve',
  ];

  // ✅ ログに出ている proxy ルート（これを使う）
  static const String _metadataProxyPath = '/mall/me/wallets/metadata/proxy';

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

    // 画面引数で埋まっている項目が少ない場合は取得
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

      final base = resolveApiBase().trim();
      final b = Uri.tryParse(base);
      if (base.isEmpty || b == null || !b.hasScheme || !b.hasAuthority) {
        _setErr('API base is invalid: "$base"');
        return;
      }

      // ✅ サインイン必須（Authorization が必要）
      final u = FirebaseAuth.instance.currentUser;
      if (u == null) {
        _setErr('Not signed in.');
        return;
      }

      // ✅ avatarId は URL から拾わず store から解決（セキュリティ要件）
      final avatarId = (await AvatarIdStore.I.resolveMyAvatarId() ?? '').trim();
      if (avatarId.isEmpty) {
        // サーバ側で avatarId が必須の実装ならここで止める
        // もし必須でないなら warning にして続行でも良い
        _setErr('avatarId could not be resolved.');
        return;
      }

      final token = await _getIdToken(u, forceRefresh: false);
      if (token == null) {
        _setErr('Failed to get idToken.');
        return;
      }

      // 1) resolve（product/brand/metadataUri などを得る）
      final resolved = await _fetchResolveWithFallback(
        baseUri: b,
        mintAddress: mint,
        bearer: token,
      );

      // resolved が取れない場合でも、最低限 metadata/proxy を試す余地はあるが
      // metadataUri が無いので、ここではエラー扱いにする
      if (resolved == null) {
        _setErr(
          'Resolve API not found. Tried: ${_resolveCandidates.join(', ')}',
        );
        return;
      }

      // 2) metadata は proxy 経由で取得（外部直 GET はしない）
      TokenMetadataDTO? meta;
      final metaUri = resolved.metadataUri.trim();
      if (metaUri.isNotEmpty) {
        meta = await _fetchMetadataViaProxy(
          baseUri: b,
          avatarId: avatarId,
          metadataUrl: metaUri,
          bearer: token,
        );
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

  Future<String?> _getIdToken(User u, {required bool forceRefresh}) async {
    final String? raw = await u.getIdToken(forceRefresh);
    final t = (raw ?? '').trim();
    return t.isEmpty ? null : t;
  }

  Future<TokenResolveDTO?> _fetchResolveWithFallback({
    required Uri baseUri,
    required String mintAddress,
    required String bearer,
  }) async {
    // まず通常取得 → 401/403なら refresh → それでもダメなら null
    Future<http.Response> authedGet(Uri uri, String token) {
      return http.get(
        uri,
        headers: <String, String>{
          'Accept': 'application/json',
          'Authorization': 'Bearer $token',
        },
      );
    }

    // 404 のときは次候補へ、401/403 のときは 1 回だけ token refresh して再試行
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) return null;

    String token = bearer;

    for (final path in _resolveCandidates) {
      final uri = baseUri.replace(
        path: _joinPaths(baseUri.path, path),
        queryParameters: <String, String>{'mintAddress': mintAddress},
        fragment: null,
      );

      http.Response res = await authedGet(uri, token);

      if (res.statusCode == 401 || res.statusCode == 403) {
        final refreshed = await _getIdToken(u, forceRefresh: true);
        if (refreshed == null) return null;
        token = refreshed;
        res = await authedGet(uri, token);
      }

      if (res.statusCode == 404) {
        // 次候補へ
        continue;
      }
      if (res.statusCode < 200 || res.statusCode >= 300) {
        // 404 以外の失敗はその場でエラーとして止めたい場合もあるが、
        // ここでは次候補へ回すとデバッグしづらいので即 null 返却
        return null;
      }

      final body = res.body.trim();
      if (body.isEmpty) return null;

      final decoded = jsonDecode(body);
      Map<String, dynamic>? m;
      if (decoded is Map<String, dynamic>) {
        m = decoded;
      } else if (decoded is Map) {
        m = decoded.cast<String, dynamic>();
      }
      if (m == null) return null;

      // wrapper吸収: {data:{...}} を許容
      final data = (m['data'] is Map)
          ? (m['data'] as Map).cast<String, dynamic>()
          : m;

      return TokenResolveDTO.fromJson(data);
    }

    return null;
  }

  Future<TokenMetadataDTO?> _fetchMetadataViaProxy({
    required Uri baseUri,
    required String avatarId,
    required String metadataUrl,
    required String bearer,
  }) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) return null;

    Future<http.Response> authedGet(Uri uri, String token) {
      return http.get(
        uri,
        headers: <String, String>{
          'Accept': 'application/json',
          'Authorization': 'Bearer $token',
        },
      );
    }

    String token = bearer;

    // ✅ backend proxy に投げる（url は query で渡す）
    final uri = baseUri.replace(
      path: _joinPaths(baseUri.path, _metadataProxyPath),
      queryParameters: <String, String>{
        'avatarId': avatarId,
        'url': metadataUrl,
      },
      fragment: null,
    );

    http.Response res = await authedGet(uri, token);

    if (res.statusCode == 401 || res.statusCode == 403) {
      final refreshed = await _getIdToken(u, forceRefresh: true);
      if (refreshed == null) return null;
      token = refreshed;
      res = await authedGet(uri, token);
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      return null;
    }

    final body = res.body.trim();
    if (body.isEmpty) return null;

    final decoded = jsonDecode(body);
    Map<String, dynamic>? m;
    if (decoded is Map<String, dynamic>) {
      m = decoded;
    } else if (decoded is Map) {
      m = decoded.cast<String, dynamic>();
    }
    if (m == null) return null;

    // ✅ proxy が wrapper を返す可能性も吸収
    final data = (m['data'] is Map)
        ? (m['data'] as Map).cast<String, dynamic>()
        : m;

    return TokenMetadataDTO.fromJson(data);
  }

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    final resolved = _resolved;
    final meta = _metadata;

    // ① Widget引数 → ② resolved → ③ metadata の順で埋める
    final mint = widget.mintAddress.trim();

    final pid = _firstNonEmpty([widget.productId, resolved?.productId]);
    final bid = _firstNonEmpty([widget.brandId, resolved?.brandId]);

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

    children.addAll([
      _Card(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (bname.isNotEmpty)
              Text(
                bname,
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              )
            else
              Text(
                '（brandName 未取得）',
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(color: cs.onSurfaceVariant),
              ),
            const SizedBox(height: 10),

            if (img.isNotEmpty)
              ClipRRect(
                borderRadius: BorderRadius.circular(14),
                child: AspectRatio(
                  aspectRatio: 1,
                  child: Image.network(
                    img,
                    fit: BoxFit.cover,
                    loadingBuilder: (context, child, p) {
                      if (p == null) return child;
                      return Container(
                        color: cs.surfaceContainerHighest,
                        alignment: Alignment.center,
                        child: const SizedBox(
                          width: 22,
                          height: 22,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      );
                    },
                    errorBuilder: (context, error, st) {
                      return Container(
                        color: cs.surfaceContainerHighest,
                        alignment: Alignment.center,
                        padding: const EdgeInsets.all(12),
                        child: Text(
                          '画像を読み込めませんでした。',
                          style: Theme.of(context).textTheme.bodySmall
                              ?.copyWith(
                                color: cs.onSurfaceVariant,
                                fontWeight: FontWeight.w600,
                              ),
                        ),
                      );
                    },
                  ),
                ),
              )
            else
              Container(
                height: 220,
                decoration: BoxDecoration(
                  color: cs.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(14),
                ),
                alignment: Alignment.center,
                child: Text(
                  '（画像なし）',
                  style: Theme.of(
                    context,
                  ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
                ),
              ),

            const SizedBox(height: 10),

            if (tname.isNotEmpty)
              Text(
                tname,
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
              )
            else
              Text(
                '（トークン名 未取得）',
                style: Theme.of(
                  context,
                ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
                textAlign: TextAlign.center,
              ),

            const SizedBox(height: 8),

            if (pname.isNotEmpty)
              Text(
                pname,
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: cs.onSurfaceVariant,
                  fontWeight: FontWeight.w600,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
              )
            else
              Text(
                '（productName 未取得）',
                style: Theme.of(
                  context,
                ).textTheme.bodySmall?.copyWith(color: cs.onSurfaceVariant),
                textAlign: TextAlign.center,
              ),
          ],
        ),
      ),
      const SizedBox(height: 12),
      _Card(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _row(
              context,
              label: 'productId',
              value: pid.isEmpty ? '（未取得）' : pid,
            ),
            const Divider(height: 18),
            _row(context, label: 'brandId', value: bid.isEmpty ? '（未取得）' : bid),
            const Divider(height: 18),
            _row(
              context,
              label: 'mintAddress',
              value: mint.isEmpty ? '（未取得）' : mint,
              mono: true,
            ),
          ],
        ),
      ),
    ]);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: children,
    );
  }

  String _firstNonEmpty(List<String?> xs) {
    for (final v in xs) {
      final s = _s(v);
      if (s.isNotEmpty) return s;
    }
    return '';
  }

  Widget _row(
    BuildContext context, {
    required String label,
    required String value,
    bool mono = false,
  }) {
    final cs = Theme.of(context).colorScheme;
    final v = value.trim();

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 110,
          child: Text(
            label,
            style: Theme.of(context).textTheme.labelMedium?.copyWith(
              color: cs.onSurfaceVariant,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          child: Text(
            v,
            style:
                (mono
                        ? Theme.of(context).textTheme.bodySmall
                        : Theme.of(context).textTheme.bodyMedium)
                    ?.copyWith(
                      color: cs.onSurface,
                      fontWeight: FontWeight.w600,
                      fontFamily: mono ? 'monospace' : null,
                    ),
          ),
        ),
      ],
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

String _joinPaths(String a, String b) {
  final aa = a.trim();
  final bb = b.trim();
  if (aa.isEmpty || aa == '/') return bb.startsWith('/') ? bb : '/$bb';
  if (bb.isEmpty || bb == '/') return aa;
  if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
  if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
  return aa + bb;
}
