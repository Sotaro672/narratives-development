// frontend/mall/lib/features/wallet/presentation/hook/use_contents.dart
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart'; // ✅ AvatarIdStore
import '../../infrastructure/repository_http.dart'; // ✅ WalletRepositoryHttp
import '../../infrastructure/token_metadata_dto.dart';
import '../../infrastructure/token_resolve_dto.dart';

class WalletContentsViewModel {
  const WalletContentsViewModel({
    required this.loading,
    required this.error,
    required this.mintAddress,
    required this.productId,
    required this.brandName,
    required this.productName,
    required this.tokenName,
    required this.imageUrl,
    required this.goPreviewByProductId,
    required this.openContents,
  });

  final bool loading;
  final String error;

  final String mintAddress;

  /// NOTE: 画面表示は productName 側で行うが、遷移に必要
  final String productId;

  final String brandName;
  final String productName;
  final String tokenName;
  final String imageUrl;

  final void Function(BuildContext context, String productId)
  goPreviewByProductId;

  /// ✅ tokenName 押下で contents.dart へ遷移（Preview埋め込み用）
  final void Function(BuildContext context) openContents;
}

/// WalletContentsPage 用のロジック集約 hook.
WalletContentsViewModel useWalletContentsViewModel({
  required String mintAddress,
  String? productId,
  String? brandId,
  String? brandName,
  String? productName,
  String? tokenName,
  String? imageUrl,
  String? from,
}) {
  final repo = useMemoized(() => WalletRepositoryHttp());
  useEffect(() => repo.dispose, [repo]);

  final loading = useState<bool>(false);
  final error = useState<String>('');

  final resolved = useState<TokenResolveDTO?>(null);
  final metadata = useState<TokenMetadataDTO?>(null);

  String s(String? v) => (v ?? '').trim();

  String firstNonEmpty(List<String?> xs) {
    for (final v in xs) {
      final t = s(v);
      if (t.isNotEmpty) return t;
    }
    return '';
  }

  Future<void> setErr(String msg) async {
    error.value = msg.trim();
  }

  Future<void> load() async {
    if (loading.value) return;

    loading.value = true;
    error.value = '';

    try {
      final mint = mintAddress.trim();
      if (mint.isEmpty) {
        await setErr('mintAddress is required.');
        return;
      }

      // ✅ avatarId は URL ではなく store から解決（期待値）
      final avatarId = (await AvatarIdStore.I.resolveMyAvatarId() ?? '').trim();
      if (avatarId.isEmpty) {
        await setErr('avatarId could not be resolved.');
        return;
      }

      // 1) resolve（product/brand/metadataUri 等）
      final r = await repo.resolveTokenByMintAddress(mint);
      if (r == null) {
        await setErr('Failed to resolve token by mintAddress.');
        return;
      }

      // 2) metadata（proxy 経由は repository が担保）
      TokenMetadataDTO? m;
      final metaUri = r.metadataUri.trim();
      if (metaUri.isNotEmpty) {
        m = await repo.fetchTokenMetadata(metaUri);
      }

      resolved.value = r;
      metadata.value = m;
    } catch (e) {
      await setErr(e.toString());
    } finally {
      loading.value = false;
    }
  }

  // 初期ロード判定（prefill がある場合は必要時のみ補完）
  useEffect(() {
    final mint = mintAddress.trim();
    if (mint.isEmpty) {
      error.value = 'mintAddress is required.';
      return null;
    }

    final hasPrefill =
        s(productId).isNotEmpty ||
        s(brandId).isNotEmpty ||
        s(brandName).isNotEmpty ||
        s(productName).isNotEmpty ||
        s(tokenName).isNotEmpty ||
        s(imageUrl).isNotEmpty;

    if (!hasPrefill) {
      load();
      return null;
    }

    final missing =
        s(brandName).isEmpty ||
        s(productName).isEmpty ||
        s(tokenName).isEmpty ||
        s(imageUrl).isEmpty;

    if (missing) load();

    return null;
    // ignore: exhaustive_keys
  }, [mintAddress]);

  final pid = firstNonEmpty([productId, resolved.value?.productId]);

  final bname = firstNonEmpty([brandName, resolved.value?.brandName]);
  final pname = firstNonEmpty([productName, resolved.value?.productName]);

  final tname = firstNonEmpty([tokenName, metadata.value?.name]);
  final img = firstNonEmpty([imageUrl, metadata.value?.image]);

  void goPreviewByProductId(BuildContext context, String productId) {
    final pid = productId.trim();
    if (pid.isEmpty) return;
    context.go('/$pid');
  }

  // ✅ tokenName 押下時に contents.dart へ遷移
  // ルーティングのパスはプロジェクト側の実装に合わせて変更してください。
  // ここでは一般的な想定として '/wallet/contents' を使用します。
  void openContents(BuildContext context) {
    final mint = mintAddress.trim();
    if (mint.isEmpty) return;

    const path = '/wallet/contents';

    final qp = <String, String>{'mintAddress': mint};

    void putIf(String k, String v) {
      final t = v.trim();
      if (t.isNotEmpty) qp[k] = t;
    }

    putIf('productId', pid);
    putIf('brandId', firstNonEmpty([brandId, resolved.value?.brandId]));
    putIf('brandName', bname);
    putIf('productName', pname);
    putIf('tokenName', tname);
    putIf('imageUrl', img);
    putIf('from', s(from));

    final loc = Uri(path: path, queryParameters: qp).toString();
    context.push(loc);
  }

  return WalletContentsViewModel(
    loading: loading.value,
    error: error.value,
    mintAddress: mintAddress.trim(),
    productId: pid,
    brandName: bname,
    productName: pname,
    tokenName: tname,
    imageUrl: img,
    goPreviewByProductId: goPreviewByProductId,
    openContents: openContents,
  );
}
