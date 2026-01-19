// frontend/mall/lib/features/avatar/presentation/hook/use_avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

import '../../../../app/routing/routes.dart';
import '../../../wallet/infrastructure/repository_http.dart';
import '../../../wallet/infrastructure/token_resolve_dto.dart';
import '../../../wallet/infrastructure/wallet_dto.dart';
import '../../infrastructure/avatar_api_client.dart';
import '../model/avatar_vm.dart';
import '../navigation/avatar_navigation.dart';
import '../model/me_avatar.dart';

AvatarVm useAvatarVm(BuildContext context, {String? from}) {
  String s(String? v) => (v ?? '').trim();

  // ---------------------------
  // Repository / Client lifecycle
  // ---------------------------
  final walletRepo = useMemoized(() => WalletRepositoryHttp());
  useEffect(() {
    return () => walletRepo.dispose();
  }, [walletRepo]);

  final apiClient = useMemoized(() => const AvatarApiClient(), const []);

  // ---------------------------
  // Auth stream
  // ---------------------------
  final authSnap = useStream<User?>(
    FirebaseAuth.instance.userChanges(),
    initialData: FirebaseAuth.instance.currentUser,
  );
  final user = FirebaseAuth.instance.currentUser ?? authSnap.data;

  // ---------------------------
  // Tab state
  // ---------------------------
  final tabState = useState<ProfileTab>(ProfileTab.tokens);

  // ---------------------------
  // Data loads
  // ---------------------------
  final urlAvatarId = resolveAvatarIdFromUrl(context);

  final Future<MeAvatar?>? meAvatarFuture = useMemoized(
    () => apiClient.fetchMeAvatar(),
    const [],
  );
  final meAvatarSnap = useFuture<MeAvatar?>(meAvatarFuture);

  final Future<WalletDTO?>? walletFuture = useMemoized(() async {
    final me = await meAvatarFuture;
    if (me == null) return null;

    final urlAid = urlAvatarId.trim();
    final effectiveAid = urlAid.isNotEmpty ? urlAid : me.avatarId;
    if (effectiveAid.trim().isEmpty) return null;

    // ✅ ログと同じ挙動：sync → fetch で最新を表示
    return walletRepo.syncAndFetchMeWallet();
  }, [urlAvatarId, meAvatarFuture, walletRepo]);
  final walletSnap = useFuture<WalletDTO?>(walletFuture);

  // ---------------------------
  // ✅ Resolve tokens by mintAddress (Firestore tokens lookup)
  // ---------------------------
  final Future<Map<String, TokenResolveDTO>>? resolveFuture = useMemoized(
    () async {
      final w = await walletFuture;
      final mints = (w?.tokens ?? const <String>[])
          .map((e) => e.trim())
          .where((e) => e.isNotEmpty)
          .toList();

      if (mints.isEmpty) return <String, TokenResolveDTO>{};

      // 重複排除（順序維持）
      final seen = <String>{};
      final uniq = <String>[];
      for (final m in mints) {
        if (seen.add(m)) uniq.add(m);
      }

      // 並列 resolve（失敗は握りつぶし、取れたものだけ map に入れる）
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

        // DTO 側の mintAddress とキーがズレても困るので、キーは request mint を優先
        out[uniq[i]] = dto;
      }
      return out;
    },
    [walletFuture, walletRepo],
  );
  final resolvedSnap = useFuture<Map<String, TokenResolveDTO>>(resolveFuture);

  // ---------------------------
  // URL normalize + auto-return
  // ---------------------------
  final normalizedUrlOnce = useRef(false);
  final returnedToFromOnce = useRef(false);

  useEffect(() {
    final me = meAvatarSnap.data;
    if (me == null || me.avatarId.trim().isEmpty) return null;

    final effectiveAid = urlAvatarId.trim().isNotEmpty
        ? urlAvatarId.trim()
        : me.avatarId;

    ensureAvatarIdInUrl(
      context: context,
      avatarId: effectiveAid,
      alreadyNormalized: normalizedUrlOnce.value,
      markNormalized: () => normalizedUrlOnce.value = true,
    );

    maybeReturnToFrom(
      context: context,
      avatarId: effectiveAid,
      from: from,
      alreadyReturned: returnedToFromOnce.value,
      markReturned: () => returnedToFromOnce.value = true,
    );

    return null;
  }, [meAvatarSnap.data?.avatarId, urlAvatarId, from]);

  // ---------------------------
  // Navigation handlers
  // ---------------------------
  void goToAvatarEditHandler() => goToAvatarEdit(context);

  // ---------------------------
  // View-derived fields
  // ---------------------------
  final tokens = walletSnap.data?.tokens ?? const <String>[];
  final resolvedTokens = resolvedSnap.data ?? const <String, TokenResolveDTO>{};

  final counts = ProfileCounts(
    postCount: 0,
    followerCount: 0,
    followingCount: 0,
    tokenCount: tokens.length,
  );

  final backTo = effectiveFrom(context, from: from);

  // NOTE:
  // router.dart は /login の from を「生URL」で受け取る前提になっているため
  // ここでは encodeFrom は使わず、従来どおり backTo をそのまま渡す。
  final loginUri = Uri(
    path: '/login',
    queryParameters: {AppQueryKey.from: backTo},
  );

  final photoUrl = user == null ? '' : s(user.photoURL);

  String profileBioFor(User u) {
    return '（プロフィール未設定）';
  }

  final bio = user == null ? '（プロフィール未設定）' : profileBioFor(user);

  return AvatarVm(
    authSnap: authSnap,
    user: user,
    meAvatarSnap: meAvatarSnap,
    walletSnap: walletSnap,
    tokens: tokens,
    resolvedTokens: resolvedTokens,
    counts: counts,
    tab: tabState.value,
    setTab: (next) => tabState.value = next,
    backTo: backTo,
    loginUri: loginUri,
    photoUrl: photoUrl,
    bio: bio,
    goToAvatarEdit: goToAvatarEditHandler,
  );
}
