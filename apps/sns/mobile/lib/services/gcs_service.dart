import 'dart:io';
import 'dart:typed_data';
import 'package:firebase_storage/firebase_storage.dart';
import 'package:flutter/foundation.dart';
import 'package:image_picker/image_picker.dart';
import 'package:uuid/uuid.dart';
import 'package:graphql_flutter/graphql_flutter.dart';

class GCSService {
  final FirebaseStorage _storage = FirebaseStorage.instance;
  final Uuid _uuid = const Uuid();
  GraphQLClient? _graphqlClient;
  
  // Set GraphQL client for API communication
  void setGraphQLClient(GraphQLClient client) {
    _graphqlClient = client;
  }
  
  // GraphQL mutations for image upload
  static const String uploadImageMutation = '''
    mutation UploadImage(\$type: String!, \$url: String!, \$userId: String!, \$metadata: JSON) {
      uploadImage(input: {
        type: \$type
        url: \$url
        userId: \$userId
        metadata: \$metadata
      }) {
        id
        url
        type
        uploadedAt
      }
    }
  ''';
  
  // Bucket configuration
  static const String bucketName = 'narratives-development.appspot.com';
  static const String avatarsPath = 'avatars';
  static const String postsPath = 'posts';
  
  // Upload avatar image
  Future<String?> uploadAvatarImage({
    required String userId,
    required dynamic imageSource, // File for mobile, Uint8List for web
    String? fileName,
  }) async {
    try {
      final imageId = fileName ?? '${_uuid.v4()}.jpg';
      final ref = _storage.ref().child('$avatarsPath/$userId/$imageId');
      
      UploadTask uploadTask;
      
      if (kIsWeb) {
        // Web upload
        uploadTask = ref.putData(imageSource as Uint8List);
      } else {
        // Mobile upload
        uploadTask = ref.putFile(imageSource as File);
      }
      
      final snapshot = await uploadTask;
      final downloadUrl = await snapshot.ref.getDownloadURL();
      
      // Register upload in backend via GraphQL
      if (_graphqlClient != null) {
        await _registerImageUpload(
          type: 'avatar',
          url: downloadUrl,
          userId: userId,
          metadata: {'fileName': fileName, 'imageId': imageId},
        );
      }
      
      if (kDebugMode) {
        print('Avatar uploaded successfully: $downloadUrl');
      }
      
      return downloadUrl;
    } catch (e) {
      if (kDebugMode) {
        print('Error uploading avatar: $e');
      }
      throw Exception('Failed to upload avatar: $e');
    }
  }
  
  // Upload post image
  Future<String?> uploadPostImage({
    required String userId,
    required dynamic imageSource, // File for mobile, Uint8List for web
    String? fileName,
    String? postId,
  }) async {
    try {
      final imageId = fileName ?? '${_uuid.v4()}.jpg';
      final ref = _storage.ref().child('$postsPath/$userId/$imageId');
      
      UploadTask uploadTask;
      
      if (kIsWeb) {
        // Web upload
        uploadTask = ref.putData(imageSource as Uint8List);
      } else {
        // Mobile upload
        uploadTask = ref.putFile(imageSource as File);
      }
      
      final snapshot = await uploadTask;
      final downloadUrl = await snapshot.ref.getDownloadURL();
      
      // Register upload in backend via GraphQL
      if (_graphqlClient != null) {
        await _registerImageUpload(
          type: 'post',
          url: downloadUrl,
          userId: userId,
          metadata: {
            'fileName': fileName,
            'imageId': imageId,
            'postId': postId,
          },
        );
      }
      
      if (kDebugMode) {
        print('Post image uploaded successfully: $downloadUrl');
      }
      
      return downloadUrl;
    } catch (e) {
      if (kDebugMode) {
        print('Error uploading post image: $e');
      }
      throw Exception('Failed to upload post image: $e');
    }
  }
  
  // Register image upload with GraphQL
  Future<void> _registerImageUpload({
    required String type,
    required String url,
    required String userId,
    Map<String, dynamic>? metadata,
  }) async {
    try {
      if (_graphqlClient == null) return;
      
      final MutationOptions options = MutationOptions(
        document: gql(uploadImageMutation),
        variables: {
          'type': type,
          'url': url,
          'userId': userId,
          'metadata': metadata ?? {},
        },
      );
      
      final QueryResult result = await _graphqlClient!.mutate(options);
      
      if (result.hasException) {
        if (kDebugMode) {
          print('GraphQL upload registration failed: ${result.exception}');
        }
      } else {
        if (kDebugMode) {
          print('Image upload registered successfully in backend');
        }
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error registering image upload: $e');
      }
    }
  }
  
  // Pick and upload avatar image
  Future<String?> pickAndUploadAvatarImage(String userId) async {
    try {
      final ImagePicker picker = ImagePicker();
      final XFile? image = await picker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 512,
        maxHeight: 512,
        imageQuality: 80,
      );
      
      if (image == null) return null;
      
      if (kIsWeb) {
        final bytes = await image.readAsBytes();
        return await uploadAvatarImage(
          userId: userId,
          imageSource: bytes,
          fileName: image.name,
        );
      } else {
        final file = File(image.path);
        return await uploadAvatarImage(
          userId: userId,
          imageSource: file,
          fileName: image.name,
        );
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error picking and uploading avatar: $e');
      }
      throw Exception('Failed to pick and upload avatar: $e');
    }
  }
  
  // Pick and upload post image
  Future<String?> pickAndUploadPostImage(String userId, {String? postId}) async {
    try {
      final ImagePicker picker = ImagePicker();
      final XFile? image = await picker.pickImage(
        source: ImageSource.gallery,
        maxWidth: 1080,
        maxHeight: 1080,
        imageQuality: 85,
      );
      
      if (image == null) return null;
      
      if (kIsWeb) {
        final bytes = await image.readAsBytes();
        return await uploadPostImage(
          userId: userId,
          imageSource: bytes,
          fileName: image.name,
          postId: postId,
        );
      } else {
        final file = File(image.path);
        return await uploadPostImage(
          userId: userId,
          imageSource: file,
          fileName: image.name,
          postId: postId,
        );
      }
    } catch (e) {
      if (kDebugMode) {
        print('Error picking and uploading post image: $e');
      }
      throw Exception('Failed to pick and upload post image: $e');
    }
  }
  
  // Delete image from GCS
  Future<bool> deleteImage(String imageUrl) async {
    try {
      final ref = _storage.refFromURL(imageUrl);
      await ref.delete();
      
      if (kDebugMode) {
        print('Image deleted successfully: $imageUrl');
      }
      
      return true;
    } catch (e) {
      if (kDebugMode) {
        print('Error deleting image: $e');
      }
      return false;
    }
  }
  
  // Get optimized image URL for different sizes
  String getOptimizedImageUrl(String originalUrl, {int? width, int? height}) {
    // Firebase Storage automatic image optimization
    if (width != null || height != null) {
      final uri = Uri.parse(originalUrl);
      final params = <String, String>{...uri.queryParameters};
      
      if (width != null) params['w'] = width.toString();
      if (height != null) params['h'] = height.toString();
      
      return uri.replace(queryParameters: params).toString();
    }
    
    return originalUrl;
  }
}
