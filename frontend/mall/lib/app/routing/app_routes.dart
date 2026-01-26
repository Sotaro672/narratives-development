import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'navigation.dart';
import 'routes.dart';

import '../shell/presentation/layout/app_shell.dart';
import '../shell/presentation/components/footer.dart';

// pages
import '../../features/list/presentation/page/list.dart';
import '../../features/list/presentation/page/catalog.dart';
import '../../features/avatar/presentation/page/avatar.dart';
import '../../features/avatar/presentation/page/avatar_edit.dart';
import '../../features/user/presentation/page/user_edit.dart';
import '../../features/cart/presentation/page/cart.dart';
import '../../features/preview/presentation/page/preview.dart';
import '../../features/payment/presentation/page/payment.dart';
import '../../features/wallet/presentation/page/contents.dart';

// types
import '../../features/list/infrastructure/list_repository_http.dart';

// auth pages（alias で確実に解決させる）
import '../../features/auth/presentation/page/login_page.dart' as auth_login;
import '../../features/auth/presentation/page/create_account.dart'
    as auth_create;
import '../../features/auth/presentation/page/shipping_address.dart'
    as auth_ship;
import '../../features/auth/presentation/page/billing_address.dart'
    as auth_bill;
import '../../features/auth/presentation/page/avatar_create.dart'
    as auth_avatar;

// header meta/actions（router.dart から分離済み前提）
import 'app_scaffold_meta.dart';
import 'header/header_actions.dart';

/// ------------------------------------------------------------
/// ✅ /:productId が “固定パス” と衝突した時の安全弁（念のため）
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

    // ✅ wallet contents
    'wallet',
  };
  return reserved.contains(seg);
}

List<RouteBase> buildAppRoutes({required bool firebaseReady}) {
  return [
    // -------------------------
    // Auth / settings routes (ShellRoute 外)
    // -------------------------
    GoRoute(
      path: AppRoutePath.login,
      name: AppRouteName.login,
      pageBuilder: (context, state) {
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(child: auth_login.LoginPage(intent: intent));
      },
    ),
    GoRoute(
      path: AppRoutePath.createAccount,
      name: AppRouteName.createAccount,
      pageBuilder: (context, state) {
        final intent = state.uri.queryParameters[AppQueryKey.intent];
        return NoTransitionPage(
          child: auth_create.CreateAccountPage(intent: intent),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.shippingAddress,
      name: AppRouteName.shippingAddress,
      pageBuilder: (context, state) {
        final qp = state.uri.queryParameters;
        return NoTransitionPage(
          child: auth_ship.ShippingAddressPage(
            mode: qp[AppQueryKey.mode],
            oobCode: qp[AppQueryKey.oobCode],
            continueUrl: qp[AppQueryKey.continueUrl],
            lang: qp[AppQueryKey.lang],
            intent: qp[AppQueryKey.intent],
          ),
        );
      },
    ),
    GoRoute(
      path: AppRoutePath.billingAddress,
      name: AppRouteName.billingAddress,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: auth_bill.BillingAddressPage());
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarCreate,
      name: AppRouteName.avatarCreate,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: auth_avatar.AvatarCreatePage());
      },
    ),
    GoRoute(
      path: AppRoutePath.avatarEdit,
      name: AppRouteName.avatarEdit,
      pageBuilder: (context, state) {
        return const NoTransitionPage(child: AvatarEditPage());
      },
    ),
    GoRoute(
      path: AppRoutePath.userEdit,
      name: AppRouteName.userEdit,
      pageBuilder: (context, state) {
        return NoTransitionPage(
          child: UserEditPage(tab: state.uri.queryParameters[AppQueryKey.tab]),
        );
      },
    ),

    // -------------------------
    // Shell routes (header/footer 常時表示)
    // -------------------------
    ShellRoute(
      builder: (context, state, child) {
        return AppShell(
          title: resolveTitleFor(state),
          showBack: resolveShowBackFor(state),
          actions: buildHeaderActionsFor(
            state,
            allowLogin: true,
            firebaseReady: firebaseReady,
          ),
          footer: const SignedInFooter(),
          child: child,
        );
      },
      routes: [
        GoRoute(
          path: AppRoutePath.home,
          name: AppRouteName.home,
          pageBuilder: (context, state) =>
              const NoTransitionPage(child: HomePage()),
        ),
        GoRoute(
          path: AppRoutePath.catalog,
          name: AppRouteName.catalog,
          builder: (context, state) {
            final listId = state.pathParameters['listId'] ?? '';
            final extra = state.extra;
            final initialItem = extra is MallListItem ? extra : null;
            return CatalogPage(listId: listId, initialItem: initialItem);
          },
        ),
        GoRoute(
          path: AppRoutePath.avatar,
          name: AppRouteName.avatar,
          pageBuilder: (context, state) {
            return const NoTransitionPage(child: AvatarPage());
          },
        ),

        // ✅ wallet token contents（Shell内）
        GoRoute(
          path: AppRoutePath.walletContents,
          name: AppRouteName.walletContents,
          pageBuilder: (context, state) {
            String s(String? v) => (v ?? '').trim();
            String? asOpt(String v) => v.trim().isEmpty ? null : v.trim();

            final qp = state.uri.queryParameters;

            // ✅ primary: query param, fallback: state.extra(String)
            final mintFromQP = s(qp['mintAddress']);
            final mintFromExtra = s(
              state.extra is String ? state.extra as String : null,
            );
            final mint = mintFromQP.isNotEmpty ? mintFromQP : mintFromExtra;

            if (mint.isEmpty) {
              return const NoTransitionPage(
                child: Center(child: Text('mintAddress is required.')),
              );
            }

            return NoTransitionPage(
              key: ValueKey('wallet-contents-$mint'),
              child: WalletContentsPage(
                mintAddress: mint,

                // ✅ query から補完（prefill用）
                productId: asOpt(s(qp['productId'])),
                brandId: asOpt(s(qp['brandId'])),
                brandName: asOpt(s(qp['brandName'])),
                productName: asOpt(s(qp['productName'])),
                tokenName: asOpt(s(qp['tokenName'])),

                // 互換
                imageUrl: asOpt(s(qp['imageUrl'])),

                // ✅ 追加: icon/contents（prefill）
                iconUrl: asOpt(s(qp['iconUrl'])),
                contentsUrl: asOpt(s(qp['contentsUrl'])),

                // ✅ 戻り先
                from: asOpt(s(qp['from'])),
              ),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.cart,
          name: AppRouteName.cart,
          pageBuilder: (context, state) {
            return const NoTransitionPage(
              key: ValueKey('cart'),
              child: CartPage(),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.preview,
          name: AppRouteName.preview,
          pageBuilder: (context, state) {
            final qp = state.uri.queryParameters;
            final productId = (qp[AppQueryKey.productId] ?? '').trim();

            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: ValueKey('preview-$productId'),
              child: PreviewPage(
                avatarId: avatarId,
                productId: productId.isEmpty ? null : productId,
              ),
            );
          },
        ),

        GoRoute(
          path: AppRoutePath.payment,
          name: AppRouteName.payment,
          pageBuilder: (context, state) {
            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: const ValueKey('payment'),
              child: PaymentPage(avatarId: avatarId),
            );
          },
        ),

        // ✅ QR入口（/:productId）
        GoRoute(
          path: AppRoutePath.qrProduct,
          name: AppRouteName.qrProduct,
          pageBuilder: (context, state) {
            final productId = (state.pathParameters['productId'] ?? '').trim();

            if (productId.isEmpty || _isReservedTopSegment(productId)) {
              return const NoTransitionPage(child: HomePage());
            }

            // ✅ avatarId は URL から取らない（store から使う）
            final avatarId = AvatarIdStore.I.avatarId;

            return NoTransitionPage(
              key: ValueKey('qr-preview-$productId'),
              child: PreviewPage(avatarId: avatarId, productId: productId),
            );
          },
        ),
      ],
    ),
  ];
}
