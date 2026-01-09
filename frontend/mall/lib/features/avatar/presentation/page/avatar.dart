// frontend\mall\lib\features\avatar\presentation\page\avatar.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;

import '../../../../app/routing/routes.dart';
import '../../../wallet/infrastructure/repository_http.dart';
import '../../../../app/config/api_base.dart';

class _ProfileCounts {
  const _ProfileCounts({
    required this.postCount,
    required this.followerCount,
    required this.followingCount,
    required this.tokenCount,
  });

  final int postCount;
  final int followerCount;
  final int followingCount;
  final int tokenCount;
}

enum _ProfileTab { posts, tokens }

class _MeAvatar {
  const _MeAvatar({required this.avatarId, required this.walletAddress});

  final String avatarId;
  final String walletAddress;
}

class AvatarPage extends StatefulWidget {
  const AvatarPage({super.key, this.from});

  /// ✅ router.dart から渡される「遷移元」
  final String? from;

  @override
  State<AvatarPage> createState() => _AvatarPageState();
}

class _AvatarPageState extends State<AvatarPage> {
  late final WalletRepositoryHttp _walletRepo;

  Future<_MeAvatar?>? _meAvatarFuture;
  Future<WalletDTO?>? _walletFuture;

  // ✅ URL 正規化・自動戻りの多重実行防止
  bool _normalizedUrlOnce = false;
  bool _returnedToFromOnce = false;

  _ProfileTab _tab = _ProfileTab.posts;

  String s(String? v) => (v ?? '').trim();

  /// ✅ `from` は URL で壊れやすい（Hash + `?` `&` 混在）ので base64url で安全に運ぶ
  String _encodeFrom(String raw) {
    final t = raw.trim();
    if (t.isEmpty) return '';
    return base64UrlEncode(utf8.encode(t));
  }

  @override
  void initState() {
    super.initState();
    _walletRepo = WalletRepositoryHttp();
  }

  @override
  void dispose() {
    _walletRepo.dispose();
    super.dispose();
  }

  // Instagram っぽい「bio」表示欄（現状データ未接続なので placeholder）
  String _profileBioFor(User u) {
    return '（プロフィール未設定）';
  }

  String _currentUri(BuildContext context) {
    return GoRouterState.of(context).uri.toString();
  }

  String _effectiveFrom(BuildContext context) {
    final v = s(widget.from);
    if (v.isNotEmpty) return v;
    return _currentUri(context);
  }

  /// ✅ avatarId は「URLの query (?avatarId=...)」を唯一の正とする（あれば優先）
  String _resolveAvatarIdFromUrl(BuildContext context) {
    final uri = GoRouterState.of(context).uri;
    return s(uri.queryParameters[AppQueryKey.avatarId]);
  }

  // -----------------------------
  // /mall/me/avatar のみで解決
  // -----------------------------
  String _normalizeBase(String base) {
    var b = base.trim();
    while (b.endsWith('/')) {
      b = b.substring(0, b.length - 1);
    }
    return b;
  }

  Uri _uri(String path, [Map<String, String>? query]) {
    final base = _normalizeBase(resolveMallApiBase());
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p').replace(queryParameters: query);
  }

  Future<String?> _getIdToken({bool forceRefresh = false}) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) return null;
    final t = await u.getIdToken(forceRefresh);
    final token = (t ?? '').toString().trim();
    return token.isEmpty ? null : token;
  }

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) throw const FormatException('Empty response body');
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
  }

  Map<String, dynamic> _unwrapData(Map<String, dynamic> decoded) {
    final data = decoded['data'];
    if (data is Map<String, dynamic>) return data;
    if (data is Map) return Map<String, dynamic>.from(data);
    return decoded;
  }

  String _pickString(Map<String, dynamic> j, List<String> keys) {
    for (final k in keys) {
      if (!j.containsKey(k)) continue;
      final v = (j[k] ?? '').toString().trim();
      if (v.isNotEmpty) return v;
    }
    return '';
  }

  Future<_MeAvatar?> _fetchMeAvatar() async {
    final uri = _uri('/mall/me/avatar');

    // token
    final token1 = await _getIdToken(forceRefresh: false);
    final headers1 = <String, String>{'Accept': 'application/json'};
    if (token1 != null) headers1['Authorization'] = 'Bearer $token1';

    http.Response res;
    try {
      res = await http.get(uri, headers: headers1);
    } catch (_) {
      return null;
    }

    if (res.statusCode == 401) {
      // retry with refreshed token
      final token2 = await _getIdToken(forceRefresh: true);
      final headers2 = <String, String>{'Accept': 'application/json'};
      if (token2 != null) headers2['Authorization'] = 'Bearer $token2';

      try {
        res = await http.get(uri, headers: headers2);
      } catch (_) {
        return null;
      }
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      return null;
    }

    final decoded = _unwrapData(_decodeObject(res.body));

    // 形揺れ吸収：
    // 1) { avatarId: "...", walletAddress: "..." }
    // 2) { avatar: { id/avatarId/walletAddress... } }
    // 3) { me: { ... } }
    Map<String, dynamic> root = decoded;

    final avatarObj = decoded['avatar'];
    if (avatarObj is Map) root = Map<String, dynamic>.from(avatarObj);

    final meObj = decoded['me'];
    if (meObj is Map) root = Map<String, dynamic>.from(meObj);

    final avatarId = _pickString(root, const [
      'avatarId',
      'AvatarID',
      'AvatarId',
      'id',
      'ID',
    ]);

    final walletAddress = _pickString(root, const [
      'walletAddress',
      'WalletAddress',
      'address',
      'Address',
    ]);

    if (avatarId.trim().isEmpty) {
      // me/avatar は最低 avatarId を返してほしいので、無ければ null
      return null;
    }

    return _MeAvatar(
      avatarId: avatarId.trim(),
      walletAddress: walletAddress.trim(),
    );
  }

  void _kickoffLoads({required String urlAvatarId}) {
    _meAvatarFuture ??= _fetchMeAvatar();
    _walletFuture ??= _loadWallet(urlAvatarId: urlAvatarId);
  }

  Future<WalletDTO?> _loadWallet({required String urlAvatarId}) async {
    final me = await (_meAvatarFuture ??= _fetchMeAvatar());
    if (me == null) return null;

    // ✅ async gap の後に context を触らない（警告対策）
    final urlAid = urlAvatarId.trim();
    final effectiveAid = urlAid.isNotEmpty ? urlAid : me.avatarId;

    // walletAddress が me/avatar に含まれていなければ wallet は読めない（= tokens は空表示）
    final addr = me.walletAddress.trim();
    if (addr.isEmpty) return null;

    return _walletRepo.fetchByWalletAddress(
      avatarId: effectiveAid,
      walletAddress: addr,
    );
  }

  /// ✅ /avatar の URL に avatarId を必ず載せる（Header/Cart が拾えるようにする）
  /// NOTE: ここでは「avatarId が分かった」時だけ正規化する
  void _ensureAvatarIdInUrl(BuildContext context, String avatarId) {
    if (_normalizedUrlOnce) return;
    if (avatarId.isEmpty) return;

    final state = GoRouterState.of(context);
    final uri = state.uri;

    final current = s(uri.queryParameters[AppQueryKey.avatarId]);
    if (current == avatarId) {
      _normalizedUrlOnce = true;
      return;
    }

    _normalizedUrlOnce = true;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;

      final fixed = <String, String>{...uri.queryParameters};
      fixed[AppQueryKey.avatarId] = avatarId;

      final next = uri.replace(queryParameters: fixed);
      context.go(next.toString());
    });
  }

  /// ✅ 「cart から avatarId 必須で飛ばされた」ケースだけ、from に avatarId を付与して戻す
  void _maybeReturnToFrom(BuildContext context, String avatarId) {
    if (_returnedToFromOnce) return;
    if (avatarId.isEmpty) return;

    final intent = s(
      GoRouterState.of(context).uri.queryParameters[AppQueryKey.intent],
    );
    if (intent != 'requireAvatarId') return;

    final rawFrom = s(widget.from);
    if (rawFrom.isEmpty) return;

    final fromUri = Uri.tryParse(rawFrom);
    if (fromUri == null) return;

    if (fromUri.path != '/cart') return;

    final qp = <String, String>{...fromUri.queryParameters};
    qp[AppQueryKey.avatarId] = s(qp[AppQueryKey.avatarId]).isNotEmpty
        ? qp[AppQueryKey.avatarId]!
        : avatarId;

    final fixedFrom = fromUri.replace(queryParameters: qp).toString();

    _returnedToFromOnce = true;

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      context.go(fixedFrom);
    });
  }

  void _goToAvatarEdit(BuildContext context) {
    final current = GoRouterState.of(context).uri;

    final qp = <String, String>{
      AppQueryKey.from: _encodeFrom(current.toString()),
    };

    final aid = s(current.queryParameters[AppQueryKey.avatarId]);
    if (aid.isNotEmpty) qp[AppQueryKey.avatarId] = aid;

    final uri = Uri(path: AppRoutePath.avatarEdit, queryParameters: qp);
    context.go(uri.toString());
  }

  Widget _statItem(BuildContext context, String label, int value) {
    return Expanded(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            value.toString(),
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }

  Widget _profileBioBox(BuildContext context, String bio) {
    final cs = Theme.of(context).colorScheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: cs.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        bio.isEmpty ? '（プロフィール未設定）' : bio,
        style: Theme.of(
          context,
        ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
      ),
    );
  }

  Widget _tabBar(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    Widget tabButton({
      required _ProfileTab tab,
      required IconData icon,
      required String semanticsLabel,
    }) {
      final selected = _tab == tab;

      return Expanded(
        child: InkWell(
          onTap: () {
            if (_tab == tab) return;
            setState(() => _tab = tab);
          },
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 10),
            child: Icon(
              icon,
              size: 22,
              color: selected ? cs.onSurface : cs.onSurfaceVariant,
              semanticLabel: semanticsLabel,
            ),
          ),
        ),
      );
    }

    return Container(
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(color: cs.outlineVariant.withValues(alpha: 0.6)),
          bottom: BorderSide(color: cs.outlineVariant.withValues(alpha: 0.6)),
        ),
      ),
      child: Row(
        children: [
          tabButton(
            tab: _ProfileTab.posts,
            icon: Icons.grid_on,
            semanticsLabel: 'Posts',
          ),
          Container(
            width: 1,
            height: 24,
            color: cs.outlineVariant.withValues(alpha: 0.6),
          ),
          tabButton(
            tab: _ProfileTab.tokens,
            icon: Icons.local_offer_outlined,
            semanticsLabel: 'Tokens',
          ),
        ],
      ),
    );
  }

  Widget _tokenChips(BuildContext context, List<String> tokens) {
    if (tokens.isEmpty) {
      return Text(
        'トークンはまだありません。',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      );
    }

    String shortMint(String m) {
      final t = m.trim();
      if (t.length <= 12) return t;
      return '${t.substring(0, 6)}…${t.substring(t.length - 4)}';
    }

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: tokens.map((m) {
        return Chip(
          label: Text(shortMint(m)),
          visualDensity: VisualDensity.compact,
        );
      }).toList(),
    );
  }

  Widget _postsPlaceholder(BuildContext context) {
    return Container(
      height: 240,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
      ),
      child: Text(
        '（次）ここに投稿グリッドを表示します',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }

  Widget _missingMeAvatarView(BuildContext context) {
    final backTo = _effectiveFrom(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.info_outline, size: 40),
              const SizedBox(height: 12),
              Text(
                'アバター情報を取得できませんでした',
                style: Theme.of(context).textTheme.titleLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              const Text(
                'サインイン状態、または /mall/me/avatar の応答を確認してください。',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              ElevatedButton.icon(
                onPressed: () {
                  final qp = <String, String>{
                    AppQueryKey.from: _encodeFrom(backTo),
                  };
                  final uri = Uri(
                    path: AppRoutePath.avatarEdit,
                    queryParameters: qp,
                  );
                  context.go(uri.toString());
                },
                icon: const Icon(Icons.edit_outlined),
                label: const Text('アバター編集へ'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;

        if (user == null) {
          final backTo = _effectiveFrom(context);
          final loginUri = Uri(
            path: '/login',
            queryParameters: {AppQueryKey.from: backTo},
          );
          return Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.lock_outline, size: 40),
                    const SizedBox(height: 12),
                    Text(
                      'Sign in required',
                      style: Theme.of(context).textTheme.titleLarge,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 8),
                    const Text(
                      'プロフィールを表示するにはサインインが必要です。',
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: () => context.go(loginUri.toString()),
                      child: const Text('Sign in'),
                    ),
                  ],
                ),
              ),
            ),
          );
        }

        // ✅ URL avatarId を async 前に一度だけ確定して渡す（context跨ぎ警告対策）
        final urlAvatarId = _resolveAvatarIdFromUrl(context);

        // ✅ このページは /mall/me/avatar を唯一の真実として扱う
        _kickoffLoads(urlAvatarId: urlAvatarId);

        final photoUrl = s(user.photoURL);
        final bio = _profileBioFor(user);

        return FutureBuilder<_MeAvatar?>(
          future: _meAvatarFuture,
          builder: (context, msnap) {
            if (msnap.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }

            final me = msnap.data;
            if (me == null || me.avatarId.trim().isEmpty) {
              return _missingMeAvatarView(context);
            }

            final effectiveAid = urlAvatarId.trim().isNotEmpty
                ? urlAvatarId.trim()
                : me.avatarId;

            _ensureAvatarIdInUrl(context, effectiveAid);
            _maybeReturnToFrom(context, effectiveAid);

            return Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 560),
                child: SingleChildScrollView(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 16,
                    vertical: 16,
                  ),
                  child: FutureBuilder<WalletDTO?>(
                    future: _walletFuture,
                    builder: (context, wsnap) {
                      final wallet = wsnap.data;
                      final tokens = wallet?.tokens ?? const <String>[];

                      final postCount = 0;
                      final followerCount = 0;
                      final followingCount = 0;

                      final counts = _ProfileCounts(
                        postCount: postCount,
                        followerCount: followerCount,
                        followingCount: followingCount,
                        tokenCount: tokens.length,
                      );

                      return Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              CircleAvatar(
                                radius: 44,
                                backgroundColor: Theme.of(
                                  context,
                                ).colorScheme.surfaceContainerHighest,
                                backgroundImage: photoUrl.isNotEmpty
                                    ? NetworkImage(photoUrl)
                                    : null,
                                child: photoUrl.isEmpty
                                    ? Icon(
                                        Icons.person,
                                        size: 44,
                                        color: Theme.of(
                                          context,
                                        ).colorScheme.onSurfaceVariant,
                                      )
                                    : null,
                              ),
                              const SizedBox(width: 16),
                              Expanded(
                                child: Column(
                                  children: [
                                    Row(
                                      children: [
                                        _statItem(
                                          context,
                                          '投稿',
                                          counts.postCount,
                                        ),
                                        _statItem(
                                          context,
                                          'トークン',
                                          counts.tokenCount,
                                        ),
                                      ],
                                    ),
                                    const SizedBox(height: 8),
                                    Row(
                                      children: [
                                        _statItem(
                                          context,
                                          'フォロー中',
                                          counts.followingCount,
                                        ),
                                        _statItem(
                                          context,
                                          'フォロワー',
                                          counts.followerCount,
                                        ),
                                      ],
                                    ),
                                  ],
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 10),
                          Row(
                            crossAxisAlignment: CrossAxisAlignment.center,
                            children: [
                              const SizedBox(width: 88 + 16),
                              Expanded(child: _profileBioBox(context, bio)),
                              const SizedBox(width: 10),
                              IconButton(
                                tooltip: 'Edit Avatar',
                                onPressed: () => _goToAvatarEdit(context),
                                icon: const Icon(Icons.edit_outlined),
                              ),
                            ],
                          ),
                          const SizedBox(height: 14),
                          _tabBar(context),
                          const SizedBox(height: 12),
                          if (_tab == _ProfileTab.tokens) ...[
                            if (wsnap.connectionState ==
                                ConnectionState.waiting)
                              const Padding(
                                padding: EdgeInsets.symmetric(vertical: 8),
                                child: LinearProgressIndicator(),
                              )
                            else if (wsnap.hasError)
                              Text(
                                'トークンの取得に失敗しました: ${wsnap.error}',
                                style: Theme.of(context).textTheme.bodyMedium
                                    ?.copyWith(
                                      color: Theme.of(
                                        context,
                                      ).colorScheme.error,
                                    ),
                              )
                            else
                              _tokenChips(context, tokens),
                          ] else ...[
                            _postsPlaceholder(context),
                          ],
                        ],
                      );
                    },
                  ),
                ),
              ),
            );
          },
        );
      },
    );
  }
}
