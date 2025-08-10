import 'package:flutter/material.dart';
import '../services/firestore_service.dart';

class UserProvider extends ChangeNotifier {
  final FirestoreService _firestoreService = FirestoreService();
  
  bool _isLoading = false;
  Map<String, dynamic>? _currentUser;
  String? _error;
  
  bool get isLoading => _isLoading;
  Map<String, dynamic>? get currentUser => _currentUser;
  String? get error => _error;
  
  Future<void> loadCurrentUser() async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      final userData = await _firestoreService.getUserProfile();
      
      if (userData == null) {
        // Create default profile if none exists
        _currentUser = {
          'firstName': '',
          'firstNameKatakana': '',
          'lastName': '',
          'lastNameKatakana': '',
          'emailAddress': '',
          'role': 'user',
          'avatarName': '',
          'iconUrl': '',
          'bio': '',
          'link': '',
        };
      } else {
        // Map Firestore field names to our field names
        _currentUser = {
          'firstName': userData['first_name'] ?? '',
          'firstNameKatakana': userData['first_name_katakana'] ?? '',
          'lastName': userData['last_name'] ?? '',
          'lastNameKatakana': userData['last_name_katakana'] ?? '',
          'emailAddress': userData['email_address'] ?? '',
          'role': userData['role'] ?? 'user',
          'avatarName': userData['avatarName'] ?? '',
          'iconUrl': userData['iconUrl'] ?? '',
          'bio': userData['bio'] ?? '',
          'link': userData['link'] ?? '',
          'createdAt': userData['created_at'],
          'updatedAt': userData['updated_at'],
        };
      }
    } catch (e) {
      _error = e.toString();
      _currentUser = null;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  Future<void> updateProfile(Map<String, dynamic> profileData) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    
    try {
      await _firestoreService.saveUserProfile(
        firstName: profileData['firstName'] ?? '',
        firstNameKatakana: profileData['firstNameKatakana'] ?? '',
        lastName: profileData['lastName'] ?? '',
        lastNameKatakana: profileData['lastNameKatakana'] ?? '',
        emailAddress: profileData['emailAddress'] ?? '',
        role: profileData['role'] ?? 'user',
        avatarName: profileData['avatarName'] ?? '',
        iconUrl: profileData['iconUrl'] ?? '',
        bio: profileData['bio'] ?? '',
        link: profileData['link'] ?? '',
      );
      
      // Update local data
      _currentUser = {
        ..._currentUser ?? {},
        ...profileData,
        'updatedAt': DateTime.now().toIso8601String(),
      };
      
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
  
  void clearUser() {
    _currentUser = null;
    _error = null;
    notifyListeners();
  }
}
