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

    /// 互換性維持: 従来の imageUrl（= iconUrl と同義）
    required this.imageUrl,

    /// 追加: token icon / token contents を画面へ渡す
    required this.iconUrl,
    required this.contentsUrl,

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

  /// 従来フィールド（互換）
  final String imageUrl;

  /// 追加フィールド
  final String iconUrl;
  final String contentsUrl;

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

  /// 互換: 既存呼び出し側が imageUrl を渡している場合に対応
  String? imageUrl,

  /// 追加: 新しく iconUrl / contentsUrl を明示的に prefill できるようにする
  String? iconUrl,
  String? contentsUrl,

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

  String pickBestContentsViewUrl(TokenResolveDTO? r) {
    if (r == null) return '';

    // tokenContentsFiles は non-nullable 前提（lint がそう言っている）
    final files = r.tokenContentsFiles;
    if (files.isEmpty) return '';

    // 念のため .keep は除外（バックエンドでも除外されている想定）
    final filtered = files
        .where((f) {
          final u = f.viewUri.trim();
          if (u.isEmpty) return false;
          return !u.endsWith('/.keep') && !u.endsWith('.keep');
        })
        .toList(growable: false);

    final target = filtered.isNotEmpty ? filtered : files;

    // application/octet-stream を優先
    for (final f in target) {
      final t = f.type.trim();
      final u = f.viewUri.trim();
      if (u.isEmpty) continue;
      if (t == 'application/octet-stream') return u;
    }

    // それ以外は先頭の viewUri
    for (final f in target) {
      final u = f.viewUri.trim();
      if (u.isNotEmpty) return u;
    }

    return '';
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

      // NOTE:
      // backend は middleware から avatarId を取得する設計なら、このチェックは本質的に不要。
      // UX要件として「自分のavatarIdが解決できないなら止める」なら残してよい。
      final avatarId = (await AvatarIdStore.I.resolveMyAvatarId() ?? '').trim();
      if (avatarId.isEmpty) {
        await setErr('avatarId could not be resolved.');
        return;
      }

      // 1) resolve（product/brand/metadataUri + tokenBlueprintId/tokenContentsFiles 等）
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
        s(imageUrl).isNotEmpty ||
        s(iconUrl).isNotEmpty ||
        s(contentsUrl).isNotEmpty;

    if (!hasPrefill) {
      load();
      return null;
    }

    final missing =
        s(brandName).isEmpty ||
        s(productName).isEmpty ||
        s(tokenName).isEmpty ||
        // icon は imageUrl 互換もあるため、両方 empty なら不足扱い
        (s(imageUrl).isEmpty && s(iconUrl).isEmpty) ||
        // contents が空なら resolve を走らせる
        s(contentsUrl).isEmpty;

    if (missing) load();

    return null;
    // ignore: exhaustive_keys
  }, [mintAddress]);

  final pid = firstNonEmpty([productId, resolved.value?.productId]);

  final bname = firstNonEmpty([brandName, resolved.value?.brandName]);
  final pname = firstNonEmpty([productName, resolved.value?.productName]);

  // tokenName: prefill -> metadata.name
  final tname = firstNonEmpty([tokenName, metadata.value?.name]);

  // icon: prefill(iconUrl or imageUrl) -> metadata.tokenIconUri -> metadata.image
  final icon = firstNonEmpty([
    iconUrl,
    imageUrl,
    metadata.value?.tokenIconUri,
    metadata.value?.image,
  ]);

  // contents:
  // prefill(contentsUrl) -> resolve.tokenContentsFiles[*].viewUri (SIGNED, expected) -> metadata.tokenContentsUri (fallback)
  final resolvedContentsView = pickBestContentsViewUrl(resolved.value);
  final contents = firstNonEmpty([
    contentsUrl,
    resolvedContentsView,
    metadata.value?.tokenContentsUri, // fallback（素URLで403になり得る）
  ]);

  // 互換: 従来の imageUrl は icon と同義で返す
  final img = icon;

  void goPreviewByProductId(BuildContext context, String productId) {
    final pid = productId.trim();
    if (pid.isEmpty) return;
    context.go('/$pid');
  }

  // ✅ tokenName 押下時に contents.dart へ遷移
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

    // 互換: 既存の imageUrl
    putIf('imageUrl', img);

    // 追加: icon / contents（contents は署名付き viewUrl を優先）
    putIf('iconUrl', icon);
    putIf('contentsUrl', contents);

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

    imageUrl: img, // 互換
    iconUrl: icon, // 追加
    contentsUrl: contents, // 追加（署名付き viewUrl を優先）

    goPreviewByProductId: goPreviewByProductId,
    openContents: openContents,
  );
}
