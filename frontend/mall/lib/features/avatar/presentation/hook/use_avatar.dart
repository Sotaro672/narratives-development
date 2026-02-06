// frontend\mall\lib\features\avatar\presentation\hook\use_avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

// ✅ prefix を付けて、WalletRepositoryHttp だけを参照する
import 'package:mall/features/wallet/infrastructure/repository_http.dart'
    as wallet_api;

import 'package:mall/features/wallet/infrastructure/token_metadata_dto.dart';
import 'package:mall/features/wallet/infrastructure/token_resolve_dto.dart';
import 'package:mall/features/wallet/infrastructure/wallet_dto.dart';

import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';
import 'package:mall/features/avatar/presentation/model/avatar_vm.dart';
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

// ✅ Pattern B: navigation helpers are centralized here
import 'package:mall/app/routing/navigation.dart';

// ✅ ✅ Header title source (AppScaffold uses this)
import 'package:mall/app/routing/avatar_name_store.dart';

/// Pattern B:
/// - URL の `from` を廃止
/// - URL から avatarId を読まない / URL に avatarId を入れない
/// - 戻り先制御は NavStore 側（router.dart / 各ページのUI側）で行う
AvatarVm useAvatarVm(BuildContext context) {
  // ---------------------------
  // ✅ logging helpers (Webでも確実に出す)
  // ---------------------------
  void log(String msg) {
    final out = '[useAvatarVm] $msg';
    debugPrint(out);
    // ignore: avoid_print
    print(out);
  }

  String s(Object? v) => (v ?? '').toString().trim();

  String short(String v, {int max = 240}) {
    final t = s(v);
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...';
  }

  // ---------------------------
  // Repository / Client lifecycle
  // ---------------------------
  final walletRepo = useMemoized(() => wallet_api.WalletRepositoryHttp());
  useEffect(() {
    log('walletRepo created');
    return () {
      log('walletRepo dispose');
      walletRepo.dispose();
    };
  }, [walletRepo]);

  // ---------------------------
  // Auth stream
  // ---------------------------
  final authSnap = useStream<User?>(
    FirebaseAuth.instance.userChanges(),
    initialData: FirebaseAuth.instance.currentUser,
  );
  final user = FirebaseAuth.instance.currentUser ?? authSnap.data;

  useEffect(() {
    final u = user;
    if (u == null) {
      log('auth user=null (signed out)');
    } else {
      log(
        'auth user uid="${s(u.uid)}" '
        'email="${s(u.email)}" '
        'displayName="${s(u.displayName)}"',
      );
    }
    return null;
  }, [user?.uid, user?.email, user?.displayName]);

  // ✅ user が変わったら apiClient を作り直す
  final apiClient = useMemoized(
    () => AvatarApiClient(
      enableLogging: true,
      logger: (m) => log('apiClient $m'),
    ),
    [user?.uid],
  );
  useEffect(() {
    log('AvatarApiClient created (dep uid="${s(user?.uid)}")');
    return () {
      log('AvatarApiClient dispose');
      apiClient.dispose();
    };
  }, [apiClient]);

  // ---------------------------
  // Tab state
  // ---------------------------
  final tabState = useState<ProfileTab>(ProfileTab.tokens);

  // ---------------------------
  // Data loads
  // ---------------------------
  final meAvatarFuture = useMemoized(() async {
    if (user == null) {
      log('meAvatarFuture skipped (user=null)');

      // ✅ ヘッダー名は backend 由来に一本化
      AvatarNameStore.I.clear();

      return null;
    }

    log('meAvatarFuture start fetchMyAvatarProfile() uid="${s(user.uid)}"');
    final res = await apiClient.fetchMyAvatarProfile();

    log(
      'meAvatarFuture done '
      'avatarId="${s(res?.avatarId)}" '
      'avatarName="${s(res?.avatarName)}" '
      'profile="${short(s(res?.profile))}" '
      'walletAddress="${s(res?.walletAddress)}"',
    );

    // ✅ ヘッダー名は backend 由来に一本化
    log(
      'AvatarNameStore.setAvatarName("${s(res?.avatarName)}") from meAvatarFuture',
    );
    AvatarNameStore.I.setAvatarName(res?.avatarName);

    return res;
  }, [apiClient, user?.uid]);
  final meAvatarSnap = useFuture<MeAvatar?>(meAvatarFuture);

  useEffect(
    () {
      log(
        'meAvatarSnap state=${meAvatarSnap.connectionState} '
        'hasData=${meAvatarSnap.data != null} '
        'hasError=${meAvatarSnap.hasError}',
      );

      if (meAvatarSnap.hasError) {
        log('meAvatarSnap error="${meAvatarSnap.error}"');
        AvatarNameStore.I.clear();
      }

      final me = meAvatarSnap.data;
      if (me != null) {
        log(
          'meAvatarSnap data '
          'avatarId="${s(me.avatarId)}" '
          'avatarName="${s(me.avatarName)}" '
          'profile="${short(s(me.profile))}" '
          'walletAddress="${s(me.walletAddress)}"',
        );

        // ✅ Future 完了タイミング差も潰す
        log(
          'AvatarNameStore.setAvatarName("${s(me.avatarName)}") from meAvatarSnap',
        );
        AvatarNameStore.I.setAvatarName(me.avatarName);
      }

      return null;
    },
    [
      meAvatarSnap.connectionState,
      meAvatarSnap.hasError,
      meAvatarSnap.data?.avatarId,
      meAvatarSnap.data?.avatarName,
      meAvatarSnap.data?.profile,
      meAvatarSnap.data?.walletAddress,
    ],
  );

  // meAvatarSnap.data に基づいて walletFuture を作り直す。
  final meAvatarId = (meAvatarSnap.data?.avatarId ?? '').trim();

  final walletFuture = useMemoized(() async {
    if (user == null) {
      log('walletFuture skipped (user=null)');
      return null;
    }
    if (meAvatarId.isEmpty) {
      log('walletFuture skipped (meAvatarId empty)');
      return null;
    }

    log('walletFuture start syncAndFetchMeWallet() meAvatarId="$meAvatarId"');
    final w = await walletRepo.syncAndFetchMeWallet();
    log('walletFuture done tokensLen=${w?.tokens.length ?? 0}');
    return w;
  }, [walletRepo, user?.uid, meAvatarId]);
  final walletSnap = useFuture<WalletDTO?>(walletFuture);

  useEffect(
    () {
      log(
        'walletSnap state=${walletSnap.connectionState} '
        'hasData=${walletSnap.data != null} '
        'hasError=${walletSnap.hasError}',
      );
      if (walletSnap.hasError) {
        log('walletSnap error="${walletSnap.error}"');
      }
      final w = walletSnap.data;
      if (w != null) {
        final tokens = w.tokens;
        final head = tokens
            .take(3)
            .map((e) => e.trim())
            .where((e) => e.isNotEmpty)
            .toList();
        log(
          'walletSnap data tokensLen=${tokens.length} '
          'head=${head.isEmpty ? "[]" : head}',
        );
      }
      return null;
    },
    [
      walletSnap.connectionState,
      walletSnap.hasError,
      walletSnap.data?.tokens.length,
    ],
  );

  // ---------------------------
  // ✅ Resolve tokens by mintAddress
  // ---------------------------
  final tokensFromWallet = walletSnap.data?.tokens ?? const <String>[];

  useEffect(() {
    final t = tokensFromWallet
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();
    final head = t.take(3).toList();
    log('tokensFromWallet changed len=${t.length} head=$head');
    return null;
  }, [tokensFromWallet.length]);

  final resolveFuture = useMemoized(() async {
    final mints = tokensFromWallet
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();

    if (mints.isEmpty) {
      log('resolveFuture skipped (no mints)');
      return <String, TokenResolveDTO>{};
    }

    final seen = <String>{};
    final uniq = <String>[];
    for (final m in mints) {
      if (seen.add(m)) uniq.add(m);
    }

    log('resolveFuture start uniqLen=${uniq.length}');
    final results = await Future.wait<TokenResolveDTO?>(
      uniq.map((m) async {
        try {
          return await walletRepo.resolveTokenByMintAddress(m);
        } catch (_) {
          return null;
        }
      }),
    );

    final out = <String, TokenResolveDTO>{};
    for (var i = 0; i < uniq.length; i++) {
      final dto = results[i];
      if (dto == null) continue;
      out[uniq[i]] = dto;
    }

    log('resolveFuture done okLen=${out.length}');
    return out;
  }, [walletRepo, user?.uid, tokensFromWallet.join(',')]);
  final resolvedSnap = useFuture<Map<String, TokenResolveDTO>>(resolveFuture);

  useEffect(
    () {
      log(
        'resolvedSnap state=${resolvedSnap.connectionState} '
        'hasData=${resolvedSnap.data != null} '
        'len=${resolvedSnap.data?.length ?? 0} '
        'hasError=${resolvedSnap.hasError}',
      );
      if (resolvedSnap.hasError) {
        log('resolvedSnap error="${resolvedSnap.error}"');
      }
      return null;
    },
    [
      resolvedSnap.connectionState,
      resolvedSnap.hasError,
      resolvedSnap.data?.length,
    ],
  );

  // ---------------------------
  // ✅ Fetch token metadata via proxy (CORS avoid)
  // ---------------------------
  final metadataFuture = useMemoized(() async {
    final resolved = await resolveFuture;
    if (resolved.isEmpty) {
      log('metadataFuture skipped (resolved empty)');
      return <String, TokenMetadataDTO>{};
    }

    final entries = resolved.entries
        .map((e) => MapEntry(e.key.trim(), e.value))
        .where((e) => e.key.isNotEmpty && e.value.metadataUri.trim().isNotEmpty)
        .toList();

    if (entries.isEmpty) {
      log('metadataFuture skipped (no metadataUri)');
      return <String, TokenMetadataDTO>{};
    }

    log('metadataFuture start entriesLen=${entries.length}');
    final results = await Future.wait<TokenMetadataDTO?>(
      entries.map((e) async {
        try {
          return await walletRepo.fetchTokenMetadata(e.value.metadataUri);
        } catch (_) {
          return null;
        }
      }),
    );

    final out = <String, TokenMetadataDTO>{};
    for (var i = 0; i < entries.length; i++) {
      final dto = results[i];
      if (dto == null) continue;
      out[entries[i].key] = dto;
    }

    log('metadataFuture done okLen=${out.length}');
    return out;
  }, [walletRepo, user?.uid, resolveFuture]);
  final metadataSnap = useFuture<Map<String, TokenMetadataDTO>>(metadataFuture);

  useEffect(
    () {
      log(
        'metadataSnap state=${metadataSnap.connectionState} '
        'len=${metadataSnap.data?.length ?? 0} '
        'hasError=${metadataSnap.hasError}',
      );
      if (metadataSnap.hasError) {
        log('metadataSnap error="${metadataSnap.error}"');
      }
      return null;
    },
    [
      metadataSnap.connectionState,
      metadataSnap.hasError,
      metadataSnap.data?.length,
    ],
  );

  // ---------------------------
  // Navigation handlers
  // ---------------------------
  void goToAvatarEditHandler() => goToAvatarEdit(context);

  // ---------------------------
  // View-derived fields
  // ---------------------------
  final tokens = tokensFromWallet;

  final resolvedTokens = resolvedSnap.data ?? const <String, TokenResolveDTO>{};
  final tokenMetadatas =
      metadataSnap.data ?? const <String, TokenMetadataDTO>{};

  final isTokensLoading =
      walletSnap.connectionState == ConnectionState.waiting ||
      resolvedSnap.connectionState == ConnectionState.waiting ||
      metadataSnap.connectionState == ConnectionState.waiting;

  final tokenLoadingByMint = <String, bool>{};
  for (final raw in tokens) {
    final m = raw.trim();
    if (m.isEmpty) continue;

    final hasResolved = resolvedTokens.containsKey(m);
    final hasMeta = tokenMetadatas.containsKey(m);

    tokenLoadingByMint[m] = isTokensLoading || !hasResolved || !hasMeta;
  }

  final counts = ProfileCounts(
    postCount: 0,
    followerCount: 0,
    followingCount: 0,
    tokenCount: tokens.length,
  );

  final me = meAvatarSnap.data;
  final avatarIcon = me?.avatarIcon;
  final profile = me?.profile;

  useEffect(() {
    log(
      'AvatarVm summary '
      'uid="${s(user?.uid)}" '
      'meAvatarId="$meAvatarId" '
      'avatarName="${s(me?.avatarName)}" '
      'store.avatarName="${AvatarNameStore.I.avatarName}" '
      'profile="${short(s(profile))}" '
      'tokensLen=${tokens.length}',
    );
    return null;
  }, [user?.uid, meAvatarId, me?.avatarName, profile, tokens.length]);

  return AvatarVm(
    authSnap: authSnap,
    user: user,
    meAvatarSnap: meAvatarSnap,
    walletSnap: walletSnap,
    tokens: tokens,
    resolvedTokens: resolvedTokens,
    tokenMetadatas: tokenMetadatas,
    isTokensLoading: isTokensLoading,
    tokenLoadingByMint: tokenLoadingByMint,
    counts: counts,
    tab: tabState.value,
    setTab: (next) => tabState.value = next,
    avatarIcon: avatarIcon,
    profile: profile,
    goToAvatarEdit: goToAvatarEditHandler,
  );
}
