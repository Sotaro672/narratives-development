// frontend/sns/lib/firebase_options.dart
//
// Generated-like template for FlutterFire.
// Project: narratives-development-26c2d
//
// IMPORTANT:
// - Replace the placeholder values (YOUR_...) with actual values from Firebase Console.
// - Firebase Console -> Project settings -> Your apps -> (Web/Android/iOS/macOS/Windows)

import 'package:firebase_core/firebase_core.dart' show FirebaseOptions;
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, kIsWeb, TargetPlatform;

class DefaultFirebaseOptions {
  static FirebaseOptions get currentPlatform {
    if (kIsWeb) return web;

    switch (defaultTargetPlatform) {
      case TargetPlatform.android:
        return android;
      case TargetPlatform.iOS:
        return ios;
      case TargetPlatform.macOS:
        return macos;
      case TargetPlatform.windows:
        return windows;
      case TargetPlatform.linux:
        return linux;
      default:
        return web;
    }
  }

  // ------------------------------------------------------------
  // Web (registered: sns (web))

  static const FirebaseOptions web = FirebaseOptions(
    apiKey: 'AIzaSyDTetB8PcVlSHhXbItMZv2thd5lY4d5nIQ',
    appId: '1:871263659099:web:0d4bbdc36e59d7ed8d4b7e',
    messagingSenderId: '871263659099',
    projectId: 'narratives-development-26c2d',
    authDomain: 'narratives-development-26c2d.firebaseapp.com',
    storageBucket: 'narratives-development-26c2d.firebasestorage.app',
    measurementId: 'G-T77JW1DF4V',
  );

  // ------------------------------------------------------------

  // ------------------------------------------------------------
  // Android (registered: com.example.sns)

  static const FirebaseOptions android = FirebaseOptions(
    apiKey: 'AIzaSyAYGIzM85qj1k-hQ2sdFGyaLNr2zENJ42w',
    appId: '1:871263659099:android:172407550c4ba2748d4b7e',
    messagingSenderId: '871263659099',
    projectId: 'narratives-development-26c2d',
    storageBucket: 'narratives-development-26c2d.firebasestorage.app',
  );

  // ------------------------------------------------------------

  // ------------------------------------------------------------
  // iOS (registered: com.example.sns)

  static const FirebaseOptions ios = FirebaseOptions(
    apiKey: 'AIzaSyDbhBM2zKBEdK4kqbveBc05VMOKZEdB5WU',
    appId: '1:871263659099:ios:374ac95b4af4ff3e8d4b7e',
    messagingSenderId: '871263659099',
    projectId: 'narratives-development-26c2d',
    storageBucket: 'narratives-development-26c2d.firebasestorage.app',
    iosClientId: '871263659099-q23c2p66nmlqgho9r0sfm6bjnjbi5msa.apps.googleusercontent.com',
    iosBundleId: 'com.example.sns',
  );

  // ------------------------------------------------------------

  // ------------------------------------------------------------
  // macOS (registered)

  static const FirebaseOptions macos = FirebaseOptions(
    apiKey: 'AIzaSyDbhBM2zKBEdK4kqbveBc05VMOKZEdB5WU',
    appId: '1:871263659099:ios:374ac95b4af4ff3e8d4b7e',
    messagingSenderId: '871263659099',
    projectId: 'narratives-development-26c2d',
    storageBucket: 'narratives-development-26c2d.firebasestorage.app',
    iosClientId: '871263659099-q23c2p66nmlqgho9r0sfm6bjnjbi5msa.apps.googleusercontent.com',
    iosBundleId: 'com.example.sns',
  );

  // ------------------------------------------------------------

  // ------------------------------------------------------------
  // Windows (registered: sns (windows))

  static const FirebaseOptions windows = FirebaseOptions(
    apiKey: 'AIzaSyDTetB8PcVlSHhXbItMZv2thd5lY4d5nIQ',
    appId: '1:871263659099:web:45aeea3a8aa8bda28d4b7e',
    messagingSenderId: '871263659099',
    projectId: 'narratives-development-26c2d',
    authDomain: 'narratives-development-26c2d.firebaseapp.com',
    storageBucket: 'narratives-development-26c2d.firebasestorage.app',
    measurementId: 'G-F8ZRBC6LFN',
  );

  // ------------------------------------------------------------

  // Not used for now (keep to satisfy currentPlatform switch)
  static const FirebaseOptions linux = FirebaseOptions(
    apiKey: 'YOUR_LINUX_API_KEY',
    appId: 'YOUR_LINUX_APP_ID',
    messagingSenderId: 'YOUR_MESSAGING_SENDER_ID',
    projectId: 'narratives-development-26c2d',
    storageBucket: 'narratives-development-26c2d.appspot.com',
  );
}