// frontend/sns/lib/app/routing/routes.dart
class AppRouteName {
  static const home = 'home';
  static const catalog = 'catalog';
  static const avatar = 'avatar';
  static const cart = 'cart';

  // ✅ NEW
  static const payment = 'payment';

  static const login = 'login';
  static const createAccount = 'createAccount';
  static const shippingAddress = 'shippingAddress';
  static const billingAddress = 'billingAddress';
  static const avatarCreate = 'avatarCreate';
  static const avatarEdit = 'avatarEdit';
  static const userEdit = 'userEdit';
}

class AppRoutePath {
  static const home = '/';
  static const catalog = '/catalog/:listId';
  static const avatar = '/avatar';
  static const cart = '/cart';

  // ✅ NEW
  static const payment = '/payment';

  static const login = '/login';
  static const createAccount = '/create-account';
  static const shippingAddress = '/shipping-address';
  static const billingAddress = '/billing-address';
  static const avatarCreate = '/avatar-create';
  static const avatarEdit = '/avatar-edit';
  static const userEdit = '/user-edit';
}

class AppQueryKey {
  static const from = 'from';
  static const intent = 'intent';
  static const avatarId = 'avatarId';
  static const tab = 'tab';
  static const mode = 'mode';
  static const oobCode = 'oobCode';
  static const continueUrl = 'continueUrl';
  static const lang = 'lang';
}
