import 'package:cloud_firestore/cloud_firestore.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

class FirestoreService {
  final FirebaseFirestore _firestore = FirebaseFirestore.instance;
  final FirebaseAuth _auth = FirebaseAuth.instance;
  final Uuid _uuid = const Uuid();

  // Collections references
  CollectionReference get _usersCollection => _firestore.collection('users');
  CollectionReference get _avatarsCollection => _firestore.collection('avatars');
  CollectionReference get _postsCollection => _firestore.collection('posts');

  String? get _currentUserId => _auth.currentUser?.uid;

  // Create or update user profile with avatar
  Future<void> saveUserProfile({
    required String firstName,
    required String firstNameKatakana,
    required String lastName,
    required String lastNameKatakana,
    required String emailAddress,
    required String role,
    required String avatarName,
    required String iconUrl,
    required String bio,
    required String link,
  }) async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      // Start a batch write
      final batch = _firestore.batch();
      
      // User data
      final userData = {
        'user_id': _currentUserId,
        'first_name': firstName,
        'first_name_katakana': firstNameKatakana,
        'last_name': lastName,
        'last_name_katakana': lastNameKatakana,
        'email_address': emailAddress,
        'role': role,
        'updated_at': FieldValue.serverTimestamp(),
      };

      // Check if user document exists
      final userDoc = await _usersCollection.doc(_currentUserId).get();
      
      if (userDoc.exists) {
        batch.update(_usersCollection.doc(_currentUserId), userData);
      } else {
        userData['created_at'] = FieldValue.serverTimestamp();
        batch.set(_usersCollection.doc(_currentUserId), userData);
      }

      // Avatar data
      final avatarData = {
        'user_id': _currentUserId,
        'avatar_name': avatarName,
        'icon_url': iconUrl,
        'bio': bio,
        'link': link,
        'updated_at': FieldValue.serverTimestamp(),
      };

      // Check if avatar document exists
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      if (avatarQuery.docs.isNotEmpty) {
        // Update existing avatar
        final avatarDocId = avatarQuery.docs.first.id;
        batch.update(_avatarsCollection.doc(avatarDocId), avatarData);
      } else {
        // Create new avatar
        final avatarId = _uuid.v4();
        avatarData['avatar_id'] = avatarId;
        avatarData['created_at'] = FieldValue.serverTimestamp();
        batch.set(_avatarsCollection.doc(avatarId), avatarData);
      }

      // Commit the batch
      await batch.commit();

      if (kDebugMode) {
        print('User profile and avatar saved successfully');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error saving user profile and avatar: $e');
      }
      throw Exception('Failed to save user profile and avatar: $e');
    }
  }

  // Get user profile with avatar
  Future<Map<String, dynamic>?> getUserProfile() async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      // Get user data
      final userDoc = await _usersCollection.doc(_currentUserId).get();
      
      if (!userDoc.exists) {
        return null;
      }

      final userData = userDoc.data() as Map<String, dynamic>;

      // Get avatar data
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      Map<String, dynamic>? avatarData;
      if (avatarQuery.docs.isNotEmpty) {
        avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
      }

      // Combine user and avatar data
      final combinedData = <String, dynamic>{
        ...userData,
        'avatarName': avatarData?['avatar_name'] ?? '',
        'iconUrl': avatarData?['icon_url'] ?? '',
        'bio': avatarData?['bio'] ?? '',
        'link': avatarData?['link'] ?? '',
      };

      // Convert timestamps
      if (combinedData['created_at'] != null) {
        combinedData['created_at'] = (combinedData['created_at'] as Timestamp).toDate().toIso8601String();
      }
      if (combinedData['updated_at'] != null) {
        combinedData['updated_at'] = (combinedData['updated_at'] as Timestamp).toDate().toIso8601String();
      }

      return combinedData;
    } catch (e) {
      if (kDebugMode) {
        print('Error getting user profile: $e');
      }
      throw Exception('Failed to get user profile: $e');
    }
  }

  // Stream user profile with avatar for real-time updates
  Stream<Map<String, dynamic>?> getUserProfileStream() {
    if (_currentUserId == null) {
      return Stream.value(null);
    }

    return _usersCollection.doc(_currentUserId).snapshots().asyncMap((userDoc) async {
      if (!userDoc.exists) {
        return null;
      }

      final userData = userDoc.data() as Map<String, dynamic>;

      // Get avatar data
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      Map<String, dynamic>? avatarData;
      if (avatarQuery.docs.isNotEmpty) {
        avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
      }

      // Combine data
      final combinedData = <String, dynamic>{
        ...userData,
        'avatarName': avatarData?['avatar_name'] ?? '',
        'iconUrl': avatarData?['icon_url'] ?? '',
        'bio': avatarData?['bio'] ?? '',
        'link': avatarData?['link'] ?? '',
      };

      // Convert timestamps
      if (combinedData['created_at'] != null) {
        combinedData['created_at'] = (combinedData['created_at'] as Timestamp).toDate().toIso8601String();
      }
      if (combinedData['updated_at'] != null) {
        combinedData['updated_at'] = (combinedData['updated_at'] as Timestamp).toDate().toIso8601String();
      }

      return combinedData;
    });
  }

  // Delete user profile and avatar
  Future<void> deleteUserProfile() async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      final batch = _firestore.batch();

      // Delete user document
      batch.delete(_usersCollection.doc(_currentUserId));

      // Delete avatar documents
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .get();

      for (final doc in avatarQuery.docs) {
        batch.delete(doc.reference);
      }

      await batch.commit();

      if (kDebugMode) {
        print('User profile and avatar deleted successfully');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error deleting user profile and avatar: $e');
      }
      throw Exception('Failed to delete user profile and avatar: $e');
    }
  }

  // Get user avatar by user ID (for displaying in posts, etc.)
  Future<Map<String, dynamic>?> getUserAvatar(String userId) async {
    try {
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: userId)
          .limit(1)
          .get();

      if (avatarQuery.docs.isNotEmpty) {
        final avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
        
        // Convert timestamps
        if (avatarData['created_at'] != null) {
          avatarData['created_at'] = (avatarData['created_at'] as Timestamp).toDate().toIso8601String();
        }
        if (avatarData['updated_at'] != null) {
          avatarData['updated_at'] = (avatarData['updated_at'] as Timestamp).toDate().toIso8601String();
        }
        
        return avatarData;
      }
      return null;
    } catch (e) {
      if (kDebugMode) {
        print('Error getting user avatar: $e');
      }
      return null;
    }
  }

  // Posts management
  Future<void> createPost({
    required String text,
    String? mediaUrl,
  }) async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      // Get current user's avatar_id
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      String? avatarId;
      if (avatarQuery.docs.isNotEmpty) {
        final avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
        avatarId = avatarData['avatar_id'] as String?;
      }

      // Create post data
      final postId = _uuid.v4();
      final postData = {
        'post_id': postId,
        'avatar_id': avatarId ?? _currentUserId, // Use user_id as fallback
        'text': text,
        'media_url': mediaUrl ?? '',
        'created_at': FieldValue.serverTimestamp(),
        'updated_at': FieldValue.serverTimestamp(),
      };

      await _postsCollection.doc(postId).set(postData);

      if (kDebugMode) {
        print('Post created successfully: $postId');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error creating post: $e');
      }
      throw Exception('Failed to create post: $e');
    }
  }

  Future<List<Map<String, dynamic>>> getPosts({int limit = 20}) async {
    try {
      final querySnapshot = await _postsCollection
          .orderBy('created_at', descending: true)
          .limit(limit)
          .get();

      final List<Map<String, dynamic>> posts = [];

      for (final doc in querySnapshot.docs) {
        final postData = doc.data() as Map<String, dynamic>;
        postData['id'] = doc.id;

        // Convert timestamps
        if (postData['created_at'] != null) {
          postData['createdAt'] = (postData['created_at'] as Timestamp).toDate().toIso8601String();
        }
        if (postData['updated_at'] != null) {
          postData['updatedAt'] = (postData['updated_at'] as Timestamp).toDate().toIso8601String();
        }

        // Get author information from avatar
        final avatarId = postData['avatar_id'];
        if (avatarId != null) {
          try {
            final avatarDoc = await _avatarsCollection.doc(avatarId).get();
            if (avatarDoc.exists) {
              final avatarData = avatarDoc.data() as Map<String, dynamic>;
              postData['author'] = {
                'name': avatarData['avatar_name'] ?? 'Unknown User',
                'avatar': avatarData['avatar_id'],
                'iconUrl': avatarData['icon_url'] ?? '',
                'bio': avatarData['bio'] ?? '',
              };
            } else {
              // Try to get user data as fallback
              final userDoc = await _usersCollection.doc(avatarId).get();
              if (userDoc.exists) {
                final userData = userDoc.data() as Map<String, dynamic>;
                postData['author'] = {
                  'name': '${userData['last_name']} ${userData['first_name']}'.trim(),
                  'avatar': null,
                  'iconUrl': '',
                  'bio': '',
                };
              }
            }
          } catch (e) {
            if (kDebugMode) {
              print('Error getting author info: $e');
            }
          }
        }

        // Set default author if not found
        if (postData['author'] == null) {
          postData['author'] = {
            'name': 'Unknown User',
            'avatar': null,
            'iconUrl': '',
            'bio': '',
          };
        }

        // Map fields for Flutter app
        postData['content'] = postData['text'];
        postData['mediaUrl'] = postData['media_url'] ?? '';
        postData['likesCount'] = 0; // TODO: Implement likes functionality with Firestore
        postData['commentsCount'] = 0; // TODO: Implement comments functionality with Firestore
        postData['isLiked'] = false; // TODO: Check if current user liked this post
        postData['isBookmarked'] = false; // TODO: Check if current user bookmarked this post

        posts.add(postData);
      }

      return posts;
    } catch (e) {
      if (kDebugMode) {
        print('Error getting posts: $e');
      }
      throw Exception('Failed to get posts: $e');
    }
  }

  Future<void> updatePost({
    required String postId,
    required String text,
    String? mediaUrl,
  }) async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      // Check if user owns this post
      final postDoc = await _postsCollection.doc(postId).get();
      if (!postDoc.exists) {
        throw Exception('Post not found');
      }

      final postData = postDoc.data() as Map<String, dynamic>;
      final postAvatarId = postData['avatar_id'];

      // Get current user's avatar_id
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      String? currentUserAvatarId;
      if (avatarQuery.docs.isNotEmpty) {
        final avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
        currentUserAvatarId = avatarData['avatar_id'] as String?;
      }

      // Check ownership
      if (postAvatarId != currentUserAvatarId && postAvatarId != _currentUserId) {
        throw Exception('You can only edit your own posts');
      }

      // Update post
      await _postsCollection.doc(postId).update({
        'text': text,
        'media_url': mediaUrl ?? '',
        'updated_at': FieldValue.serverTimestamp(),
      });

      if (kDebugMode) {
        print('Post updated successfully: $postId');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error updating post: $e');
      }
      throw Exception('Failed to update post: $e');
    }
  }

  Future<void> deletePost(String postId) async {
    if (_currentUserId == null) {
      throw Exception('User not authenticated');
    }

    try {
      // Check if user owns this post
      final postDoc = await _postsCollection.doc(postId).get();
      if (!postDoc.exists) {
        throw Exception('Post not found');
      }

      final postData = postDoc.data() as Map<String, dynamic>;
      final postAvatarId = postData['avatar_id'];

      // Get current user's avatar_id
      final avatarQuery = await _avatarsCollection
          .where('user_id', isEqualTo: _currentUserId)
          .limit(1)
          .get();

      String? currentUserAvatarId;
      if (avatarQuery.docs.isNotEmpty) {
        final avatarData = avatarQuery.docs.first.data() as Map<String, dynamic>;
        currentUserAvatarId = avatarData['avatar_id'] as String?;
      }

      // Check ownership
      if (postAvatarId != currentUserAvatarId && postAvatarId != _currentUserId) {
        throw Exception('You can only delete your own posts');
      }

      // Delete post
      await _postsCollection.doc(postId).delete();

      if (kDebugMode) {
        print('Post deleted successfully: $postId');
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error deleting post: $e');
      }
      throw Exception('Failed to delete post: $e');
    }
  }

  // Stream posts for real-time updates
  Stream<List<Map<String, dynamic>>> getPostsStream({int limit = 20}) {
    return _postsCollection
        .orderBy('created_at', descending: true)
        .limit(limit)
        .snapshots()
        .asyncMap((snapshot) async {
      final List<Map<String, dynamic>> posts = [];

      for (final doc in snapshot.docs) {
        final postData = doc.data() as Map<String, dynamic>;
        postData['id'] = doc.id;

        // Convert timestamps
        if (postData['created_at'] != null) {
          postData['createdAt'] = (postData['created_at'] as Timestamp).toDate().toIso8601String();
        }
        if (postData['updated_at'] != null) {
          postData['updatedAt'] = (postData['updated_at'] as Timestamp).toDate().toIso8601String();
        }

        // Get author information
        final avatarId = postData['avatar_id'];
        if (avatarId != null) {
          try {
            final avatarDoc = await _avatarsCollection.doc(avatarId).get();
            if (avatarDoc.exists) {
              final avatarData = avatarDoc.data() as Map<String, dynamic>;
              postData['author'] = {
                'name': avatarData['avatar_name'] ?? 'Unknown User',
                'avatar': avatarData['avatar_id'],
                'iconUrl': avatarData['icon_url'] ?? '',
                'bio': avatarData['bio'] ?? '',
              };
            }
          } catch (e) {
            if (kDebugMode) {
              print('Error getting author info: $e');
            }
          }
        }

        // Set default author if not found
        if (postData['author'] == null) {
          postData['author'] = {
            'name': 'Unknown User',
            'avatar': null,
            'iconUrl': '',
            'bio': '',
          };
        }

        // Map fields for Flutter app
        postData['content'] = postData['text'];
        postData['mediaUrl'] = postData['media_url'] ?? '';
        postData['likesCount'] = 0; // TODO: Implement likes functionality with Firestore
        postData['commentsCount'] = 0; // TODO: Implement comments functionality with Firestore
        postData['isLiked'] = false; // TODO: Check if current user liked this post
        postData['isBookmarked'] = false; // TODO: Check if current user bookmarked this post

        posts.add(postData);
      }

      return posts;
    });
  }
}
