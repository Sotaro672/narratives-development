// frontend/sns/lib/features/avatar/presentation/page/avatar.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';
import '../../../wallet/infrastructure/wallet_repository_http.dart';

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

class AvatarPage extends StatefulWidget {
  const AvatarPage({super.key, this.from});

  /// ✅ router.dart から渡される「遷移元」
  final String? from;

  @override
  State<AvatarPage> createState() => _AvatarPageState();
}

class _AvatarPageState extends State<AvatarPage> {
  late final WalletRepositoryHttp _walletRepo;

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

  // 暫定: avatarId = Firebase UID（本来は選択中 avatarId を使う）
  String _resolveAvatarId(User user) {
    return s(user.uid);
  }

  void _kickoffLoads(User user) {
    final avatarId = _resolveAvatarId(user);
    _walletFuture ??= _walletRepo.fetchByAvatarId(avatarId);
  }

  /// ✅ /avatar の URL に avatarId を必ず載せる（Header/Cart が拾えるようにする）
  void _ensureAvatarIdInUrl(BuildContext context, String avatarId) {
    if (_normalizedUrlOnce) return;
    if (avatarId.isEmpty) return;

    final state = GoRouterState.of(context);
    final uri = state.uri;

    // いまのURLに avatarId があれば何もしない
    final current = s(uri.queryParameters[AppQueryKey.avatarId]);
    if (current == avatarId) {
      _normalizedUrlOnce = true;
      return;
    }

    _normalizedUrlOnce = true;

    // build中に go すると不安定なので post-frame で
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;

      final fixed = <String, String>{...uri.queryParameters};
      fixed[AppQueryKey.avatarId] = avatarId;

      // path は現状のまま（/avatar）
      final next = uri.replace(queryParameters: fixed);
      context.go(next.toString());
    });
  }

  /// ✅ 「cart から avatarId 必須で飛ばされた」ケースだけ、from に avatarId を付与して戻す
  void _maybeReturnToFrom(BuildContext context, String avatarId) {
    if (_returnedToFromOnce) return;
    if (avatarId.isEmpty) return;

    // intent=requireAvatarId のときだけ自動で戻す（通常のプロフィール閲覧では勝手に遷移しない）
    final intent = s(
      GoRouterState.of(context).uri.queryParameters[AppQueryKey.intent],
    );
    if (intent != 'requireAvatarId') return;

    final rawFrom = s(widget.from);
    if (rawFrom.isEmpty) return;

    final fromUri = Uri.tryParse(rawFrom);
    if (fromUri == null) return;

    // cart に戻す意図のときだけ（安全）
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

        // ✅ avatarId（暫定: uid）を確定
        final avatarId = _resolveAvatarId(user);

        // ✅ Wallet 等の読み込み開始
        _kickoffLoads(user);

        // ✅ 重要：/avatar URL に avatarId を載せる（Header / Cart が拾える）
        _ensureAvatarIdInUrl(context, avatarId);

        // ✅ cart -> avatar (intent=requireAvatarId) で来た場合だけ、from に avatarId を付けて戻す
        _maybeReturnToFrom(context, avatarId);

        final photoUrl = s(user.photoURL);
        final bio = _profileBioFor(user);

        return Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560),
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
              child: FutureBuilder<WalletDTO?>(
                future: _walletFuture,
                builder: (context, wsnap) {
                  final wallet = wsnap.data;
                  final tokens = wallet?.tokens ?? const <String>[];

                  // 現状は未接続のため 0（AvatarState 連携時に置換）
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
                      // ✅ アバター + 右側に stats
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          // left: avatar
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

                          // right: stats (2 rows)
                          Expanded(
                            child: Column(
                              children: [
                                Row(
                                  children: [
                                    _statItem(context, '投稿', counts.postCount),
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

                      // ✅ Profile box + Edit icon button (same row)
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.center,
                        children: [
                          const SizedBox(width: 88 + 16), // avatar幅 + gap
                          Expanded(child: _profileBioBox(context, bio)),
                          const SizedBox(width: 10),
                          // ✅ 文言なし（アイコンのみ）
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
                        if (wsnap.connectionState == ConnectionState.waiting)
                          const Padding(
                            padding: EdgeInsets.symmetric(vertical: 8),
                            child: LinearProgressIndicator(),
                          )
                        else if (wsnap.hasError)
                          Text(
                            'トークンの取得に失敗しました: ${wsnap.error}',
                            style: Theme.of(context).textTheme.bodyMedium
                                ?.copyWith(
                                  color: Theme.of(context).colorScheme.error,
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
  }
}
