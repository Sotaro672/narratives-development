// frontend\mall\lib\app\shell\presentation\components\footer.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../state/catalog_selection_store.dart';

// ✅ routing constants
import '../../../routing/routes.dart';

// ✅ Pattern B stores (no URL-based navigation state)
import '../../../routing/navigation.dart';

// ✅ NEW: 3 buttons extracted
import 'footer_buttons.dart';

// ✅ Cart (Pattern A stats): use_cart controller/result
import 'package:mall/features/cart/presentation/hook/use_cart.dart';

// ✅ Avatar API client (absolute schema)
import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';

// ✅ MeAvatar model (now carries avatar patch fields)
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

/// Minimal footer widget (layout primitive).
class AppFooter extends StatelessWidget {
  const AppFooter({
    super.key,
    this.left,
    this.center,
    this.right,
    this.padding = const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
  });

  final Widget? left;
  final Widget? center;
  final Widget? right;

  final EdgeInsets padding;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Theme.of(context).cardColor,
      elevation: 0,
      child: SafeArea(
        top: false,
        child: Padding(
          padding: padding,
          child: Row(
            children: [
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerLeft, child: left),
              ),
              Expanded(child: Center(child: center)),
              SizedBox(
                width: 96,
                child: Align(alignment: Alignment.centerRight, child: right),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Signed-in footer (Shop / Scan(or AddToCart) / AvatarIcon)
/// - Shows ONLY when FirebaseAuth.currentUser != null
///
/// Pattern B:
/// - `from` を URL query に入れない
/// - `avatarId` を URL query に入れない／URL から読まない
/// - 戻り先は NavStore に保持する
/// - avatarId は AvatarIdStore（= app state）を参照する
///
/// ✅ 方針B：/cart のときだけ UseCartController を参照して「購入する」押下可否を決める
class SignedInFooter extends StatefulWidget {
  const SignedInFooter({super.key});

  @override
  State<SignedInFooter> createState() => _SignedInFooterState();
}

class _SignedInFooterState extends State<SignedInFooter> {
  UseCartController? _cartUc;

  bool _isCatalogPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path.startsWith('/catalog/');
  }

  bool _isCartPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path == AppRoutePath.cart;
  }

  bool _isPaymentPath(BuildContext context) {
    final path = GoRouterState.of(context).uri.path;
    return path == AppRoutePath.payment;
  }

  String _currentLocationForReturnTo(BuildContext context) {
    // ✅ Pattern B: URL query を前提にしない。
    // 戻り先は「現在の location」を store に保持するだけ（URLへは出さない）。
    // 既に query が付いている場合でも、ここで保持するのは store 内だけなのでOK。
    return GoRouterState.of(context).uri.toString();
  }

  String _catalogListIdOrEmpty(BuildContext context) {
    final path = GoRouterState.of(context).uri.path; // /catalog/:listId
    final parts = path.split('/');
    if (parts.length >= 3 && parts[1] == 'catalog') return parts[2];
    return '';
  }

  /// ------------------------------------------------------------
  /// ✅ /:productId が固定パスと衝突しないように除外（scan時の安全弁）
  bool _isReservedTopSegment(String seg) {
    const reserved = <String>{
      'login',
      'create-account',
      'shipping-address',
      'billing-address',
      'avatar-create',
      'avatar-edit',
      'avatar',
      'user-edit',
      'cart',
      'preview',
      'payment',
      'catalog',
      'wallet',
    };
    return reserved.contains(seg);
  }

  /// ------------------------------------------------------------
  /// ✅ QRでスキャンした文字列を「アプリ内遷移先URI」に正規化して返す
  ///
  /// Pattern B:
  /// - `from` / `avatarId` を query に付与しない
  /// - 既に含まれていた場合も削除する
  Uri? _normalizeScannedToAppUri(String raw) {
    final s = raw.trim();
    if (s.isEmpty) return null;

    Uri? u;

    // 1) まず Uri として解釈
    try {
      u = Uri.parse(s);
    } catch (_) {
      u = null;
    }

    // 2) scheme が無い & / でもない場合は「生 productId」とみなす
    if (u == null || (u.scheme.isEmpty && !s.startsWith('/'))) {
      final pid = s.trim();
      if (pid.isEmpty) return null;
      if (_isReservedTopSegment(pid)) return null;
      return Uri(path: '/$pid');
    }

    // 3) http(s) の場合は path/query だけ抽出してアプリ内遷移にする
    final isHttp = u.scheme == 'http' || u.scheme == 'https';
    final extracted = Uri(
      path: (u.path.trim().isEmpty
          ? '/'
          : (u.path.startsWith('/') ? u.path : '/${u.path}')),
      queryParameters: isHttp
          ? (u.queryParameters.isEmpty ? null : u.queryParameters)
          : (u.queryParameters.isEmpty ? null : u.queryParameters),
      fragment: null, // ✅ fragment は捨てる（ルーティング破壊回避）
    );

    // 4) パスがトップ1階層（= /{something}）の場合、reserved は弾く
    final segs = extracted.pathSegments;
    if (segs.length == 1) {
      final top = segs.first.trim();
      if (top.isNotEmpty && _isReservedTopSegment(top)) {
        return null;
      }
    }

    // 5) Pattern B: URL state を持ち込まない（from/avatarId/mintAddress 等を除去）
    final merged = <String, String>{...extracted.queryParameters};

    merged.remove(AppQueryKey.from);
    merged.remove(AppQueryKey.avatarId);
    merged.remove(AppQueryKey.mintAddress);

    return extracted.replace(queryParameters: merged.isEmpty ? null : merged);
  }

  void _syncCartControllerIfNeeded(bool isCart) {
    // /cart に入ったら init、離れたら dispose（このファイルで完結）
    if (isCart && _cartUc == null) {
      _cartUc = UseCartController(context: context);
      _cartUc!.init();
    } else if (!isCart && _cartUc != null) {
      _cartUc!.dispose();
      _cartUc = null;
    }
  }

  @override
  void dispose() {
    _cartUc?.dispose();
    _cartUc = null;
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      // ✅ auth state だけを見る（avatarIcon は backend の正規キーを参照する）
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;
        if (user == null) return const SizedBox.shrink();

        final isCatalog = _isCatalogPath(context);
        final isCart = _isCartPath(context);
        final isPayment = _isPaymentPath(context);

        // ✅ /cart のときだけ cart controller を生かす
        _syncCartControllerIfNeeded(isCart);

        final listId = isCatalog ? _catalogListIdOrEmpty(context) : '';

        // ✅ Pattern B: avatarId は URL から読まない（store を参照）
        final avatarId = AvatarIdStore.I.avatarId.trim();

        // ✅ Pattern B: 戻り先は store に保持（URLへは出さない）
        final returnTo = _currentLocationForReturnTo(context);

        // ✅ Absolute schema: /mall/me/avatar は MeAvatar(=patch全体) を返す前提
        final avatarProfileFuture = Future<MeAvatar?>.microtask(() async {
          final api = AvatarApiClient();
          try {
            return await api.fetchMyAvatarProfile(); // => MeAvatar?
          } finally {
            api.dispose();
          }
        });

        return FutureBuilder<MeAvatar?>(
          future: avatarProfileFuture,
          builder: (context, profileSnap) {
            final avatarIcon = profileSnap.data?.avatarIcon;

            return Material(
              color: Theme.of(context).cardColor,
              elevation: 0,
              child: SafeArea(
                top: false,
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 18,
                    vertical: 10,
                  ),
                  child: Row(
                    children: [
                      // ✅ Shop (Home)
                      _FooterItem(
                        icon: Icons.storefront_outlined,
                        onTap: () {
                          // Home は戻り先にする必要は基本無いが、念のため現在地は保持しておく
                          NavStore.I.setReturnTo(returnTo);
                          context.go(AppRoutePath.home);
                        },
                      ),
                      const SizedBox(width: 8),

                      // ✅ 中央：
                      // - catalog: カートに入れる
                      // - cart: 購入する（paymentへ）※ cart=0 のときは押下不可
                      // - payment: 支払を確定する
                      // - other: Scan（スキャンしたURLへ遷移）
                      Expanded(
                        child: Center(
                          child: isCatalog
                              ? ValueListenableBuilder<CatalogSelection>(
                                  valueListenable:
                                      CatalogSelectionStore.notifier,
                                  builder: (context, sel, _) {
                                    final sameList =
                                        sel.listId.trim() == listId.trim();

                                    final mid = (sel.modelId ?? '').trim();
                                    final stock = sel.stockCount ?? 0;

                                    // ✅ backend が必須にした inventoryId/listId を揃える
                                    final invId = sel.inventoryId.trim();

                                    // ✅ 在庫 0 は押下不可 + 必須IDが揃っていること
                                    final enabled =
                                        sameList &&
                                        invId.isNotEmpty &&
                                        listId.trim().isNotEmpty &&
                                        mid.isNotEmpty &&
                                        stock > 0;

                                    return AddToCartButton(
                                      inventoryId: invId,
                                      listId: listId,
                                      avatarId: avatarId,
                                      enabled: enabled,
                                      modelId: sel.modelId,
                                      stockCount: sel.stockCount,
                                    );
                                  },
                                )
                              : isCart
                              // ✅ /cart のときだけ cart を参照して enabled を決める
                              ? _CartAwareGoToPaymentButton(
                                  avatarId: avatarId,
                                  controller: _cartUc,
                                )
                              : isPayment
                              ? ConfirmPaymentButton(
                                  avatarId: avatarId,
                                  enabled: avatarId.isNotEmpty,
                                )
                              : _FooterItem(
                                  icon: Icons.qr_code_scanner,
                                  onTap: () async {
                                    final code =
                                        await showModalBottomSheet<String>(
                                          context: context,
                                          isScrollControlled: true,
                                          backgroundColor: Colors.black,
                                          builder: (_) => const _QrScanSheet(),
                                        );

                                    if (!context.mounted) return;
                                    if (code == null || code.trim().isEmpty) {
                                      return;
                                    }

                                    final target = _normalizeScannedToAppUri(
                                      code,
                                    );
                                    if (target == null) {
                                      ScaffoldMessenger.of(
                                        context,
                                      ).showSnackBar(
                                        const SnackBar(
                                          content: Text('スキャン結果が無効です（遷移できません）'),
                                        ),
                                      );
                                      return;
                                    }

                                    // ✅ Pattern B: 戻り先は store へ
                                    NavStore.I.setReturnTo(returnTo);

                                    context.go(target.toString());
                                  },
                                ),
                        ),
                      ),

                      const SizedBox(width: 8),

                      // ✅ Avatar: /avatar へ（query なし）
                      _AvatarIconButton(
                        avatarIcon: avatarIcon,
                        fallbackText:
                            (user.displayName ?? user.email ?? user.uid).trim(),
                        onTap: () {
                          NavStore.I.setReturnTo(returnTo);
                          context.go(AppRoutePath.avatar);
                        },
                      ),
                    ],
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

/// ✅ /cart 専用：UseCartController を参照して「購入する」押下可否を決める
class _CartAwareGoToPaymentButton extends StatelessWidget {
  const _CartAwareGoToPaymentButton({
    required this.avatarId,
    required this.controller,
  });

  final String avatarId;
  final UseCartController? controller;

  @override
  Widget build(BuildContext context) {
    final aid = avatarId.trim();

    // controller が無い（= /cart 以外）ケースは防御的に disabled
    final uc = controller;
    if (uc == null) {
      return GoToPaymentButton(avatarId: aid, enabled: false);
    }

    // ✅ FutureBuilder で初回 fetch 完了を拾い、cart=0 のときは disabled にする
    return FutureBuilder<CartDTO>(
      future: uc.future,
      builder: (context, snap) {
        final _ = snap.data; // ignore: unused_local_variable

        // Pattern A: buildResult() から派生値を取り出して押下可否を決める
        final res = uc.buildResult((fn) {});
        final cartNotEmpty = !res.isEmpty;

        final enabled = aid.isNotEmpty && cartNotEmpty;

        return GoToPaymentButton(avatarId: aid, enabled: enabled);
      },
    );
  }
}

class _FooterItem extends StatelessWidget {
  const _FooterItem({required this.icon, required this.onTap});

  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: Icon(icon, size: 22),
      ),
    );
  }
}

class _AvatarIconButton extends StatelessWidget {
  const _AvatarIconButton({
    required this.avatarIcon,
    required this.fallbackText,
    required this.onTap,
  });

  /// ✅ absolute schema: avatarIcon
  final String? avatarIcon;

  final String fallbackText;
  final VoidCallback onTap;

  bool _isHttpUrl(String? v) {
    final s = (v ?? '').trim();
    if (s.isEmpty) return false;
    return s.startsWith('http://') || s.startsWith('https://');
  }

  @override
  Widget build(BuildContext context) {
    final icon = (avatarIcon ?? '').trim();
    final fallback = fallbackText.trim();
    final initial = fallback.isNotEmpty ? fallback[0].toUpperCase() : '?';

    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        child: CircleAvatar(
          radius: 14,
          // ✅ NetworkImage は http(s) のみ（gs:// や相対pathは表示不可なので fallback）
          backgroundImage: _isHttpUrl(icon) ? NetworkImage(icon) : null,
          child: !_isHttpUrl(icon)
              ? Text(initial, style: const TextStyle(fontSize: 12))
              : null,
        ),
      ),
    );
  }
}

/// Bottom sheet QR scanner
class _QrScanSheet extends StatefulWidget {
  const _QrScanSheet();

  @override
  State<_QrScanSheet> createState() => _QrScanSheetState();
}

class _QrScanSheetState extends State<_QrScanSheet> {
  final MobileScannerController _controller = MobileScannerController();
  bool _handled = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onDetect(BarcodeCapture capture) {
    if (_handled) return;
    final codes = capture.barcodes;
    if (codes.isEmpty) return;

    final raw = codes.first.rawValue;
    if (raw == null || raw.trim().isEmpty) return;

    _handled = true;
    Navigator.pop(context, raw.trim());
  }

  @override
  Widget build(BuildContext context) {
    final mq = MediaQuery.of(context);
    final h = mq.size.height;

    return SafeArea(
      top: false,
      child: SizedBox(
        height: h * 0.72,
        child: Stack(
          children: [
            MobileScanner(controller: _controller, onDetect: _onDetect),
            Positioned(
              left: 0,
              right: 0,
              top: 0,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    IconButton(
                      onPressed: () => Navigator.pop(context),
                      icon: const Icon(Icons.close, color: Colors.white),
                    ),
                    const SizedBox(width: 8),
                    const Text(
                      'Scan QR',
                      style: TextStyle(color: Colors.white, fontSize: 16),
                    ),
                    const Spacer(),
                    IconButton(
                      onPressed: () => _controller.toggleTorch(),
                      icon: const Icon(Icons.flash_on, color: Colors.white),
                    ),
                  ],
                ),
              ),
            ),
            Positioned(
              left: 0,
              right: 0,
              bottom: 18,
              child: Center(
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.black.withValues(alpha: 0.55),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: const Text(
                    'Point the camera at a QR code',
                    style: TextStyle(color: Colors.white),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
