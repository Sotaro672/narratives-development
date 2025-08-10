import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:provider/provider.dart';
import 'package:graphql_flutter/graphql_flutter.dart';

// Simple Services without external dependencies
class AuthService {
  bool get isAuthenticated => false;
  Future<void> signOut() async {}
  Future<bool> signInWithEmailAndPassword(String email, String password) async => false;
}

// Simplified Providers without external dependencies
class UserProvider extends ChangeNotifier {
  bool _isLoading = false;
  Map<String, dynamic>? _currentUser;
  String? _error;
  
  bool get isLoading => _isLoading;
  Map<String, dynamic>? get currentUser => _currentUser;
  String? get error => _error;
  
  Future<void> loadCurrentUser() async {
    _isLoading = true;
    notifyListeners();
    
    try {
      await Future.delayed(const Duration(seconds: 1));
      _currentUser = {
        'firstName': 'Demo',
        'firstNameKatakana': 'Demo',
        'lastName': 'User',
        'lastNameKatakana': 'User',
        'emailAddress': 'demo@narratives.com',
        'role': 'user',
        'avatarName': 'Demo User',
        'iconUrl': '',
        'bio': 'Welcome to Narratives SNS',
        'link': '',
        'createdAt': DateTime.now().toIso8601String(),
        'updatedAt': DateTime.now().toIso8601String(),
      };
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
      await Future.delayed(const Duration(seconds: 1));
      _currentUser = {
        ..._currentUser ?? {},
        ...profileData,
        'updatedAt': DateTime.now().toIso8601String(),
      };
      print('Profile updated: $profileData');
    } catch (e) {
      _error = e.toString();
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
}

class PostProvider extends ChangeNotifier {
  bool _isLoading = false;
  List<Map<String, dynamic>> _posts = [];
  
  bool get isLoading => _isLoading;
  List<Map<String, dynamic>> get posts => _posts;
  
  Future<void> loadPosts() async {
    _isLoading = true;
    notifyListeners();
    
    await Future.delayed(const Duration(seconds: 1));
    
    _posts = [
      {
        'id': '1',
        'content': 'Welcome to Narratives SNS! This is your social network for stories.',
        'author': {'name': 'System', 'iconUrl': '', 'bio': ''},
        'createdAt': DateTime.now().toIso8601String(),
        'mediaUrl': '',
        'likesCount': 12,
        'commentsCount': 3,
        'isLiked': false,
        'isBookmarked': false,
      },
      {
        'id': '2',
        'content': 'This is a demo post showing the SNS functionality. You can create posts, manage your profile, and interact with other users!',
        'author': {'name': 'Demo User', 'iconUrl': '', 'bio': 'SNS enthusiast'},
        'createdAt': DateTime.now().subtract(const Duration(hours: 2)).toIso8601String(),
        'mediaUrl': '',
        'likesCount': 8,
        'commentsCount': 1,
        'isLiked': true,
        'isBookmarked': false,
      },
      {
        'id': '3',
        'content': 'Firebase integration is working perfectly! Real-time updates and cloud storage ready.',
        'author': {'name': 'Firebase Admin', 'iconUrl': '', 'bio': 'Cloud services'},
        'createdAt': DateTime.now().subtract(const Duration(hours: 5)).toIso8601String(),
        'mediaUrl': '',
        'likesCount': 15,
        'commentsCount': 5,
        'isLiked': false,
        'isBookmarked': true,
      }
    ];
    
    _isLoading = false;
    notifyListeners();
  }
  
  Future<void> refreshPosts() async {
    await loadPosts();
  }
  
  Future<void> createPost({required String text, String? mediaUrl}) async {
    _isLoading = true;
    notifyListeners();
    
    try {
      await Future.delayed(const Duration(seconds: 1));
      
      final newPost = {
        'id': DateTime.now().millisecondsSinceEpoch.toString(),
        'content': text,
        'mediaUrl': mediaUrl ?? '',
        'author': {'name': 'Current User', 'iconUrl': '', 'bio': ''},
        'createdAt': DateTime.now().toIso8601String(),
        'likesCount': 0,
        'commentsCount': 0,
        'isLiked': false,
        'isBookmarked': false,
      };
      
      _posts.insert(0, newPost);
    } catch (e) {
      throw e;
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }
}

// GraphQL Service with proper implementation
class GraphQLService {
  late ValueNotifier<GraphQLClient> client;
  
  GraphQLService() {
    final HttpLink httpLink = HttpLink('http://localhost:8080/graphql');
    
    client = ValueNotifier(GraphQLClient(
      cache: GraphQLCache(store: HiveStore()),
      link: httpLink,
    ));
  }
  
  // Add GraphQL operations
  Future<QueryResult> query(String query, {Map<String, dynamic>? variables}) async {
    final QueryOptions options = QueryOptions(
      document: gql(query),
      variables: variables ?? {},
    );
    return await client.value.query(options);
  }
  
  Future<QueryResult> mutate(String mutation, {Map<String, dynamic>? variables}) async {
    final MutationOptions options = MutationOptions(
      document: gql(mutation),
      variables: variables ?? {},
    );
    return await client.value.mutate(options);
  }
}

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  try {
    if (kIsWeb) {
      await Firebase.initializeApp(
        options: const FirebaseOptions(
          apiKey: "AIzaSyC8qL9XQw5_-qQGGpXHBZJGBSgOzjGvhxA",
          authDomain: "narratives-development-26c2d.firebaseapp.com",
          projectId: "narratives-development-26c2d",
          storageBucket: "narratives-development-26c2d.appspot.com",
          messagingSenderId: "229613581466",
          appId: "1:229613581466:web:8f0f88901cc5cdec123456",
        ),
      );
    } else {
      await Firebase.initializeApp();
    }
    print('Firebase initialized successfully');
  } catch (e) {
    print('Firebase initialization failed: $e');
  }
  
  try {
    await initHiveForFlutter();
    print('Hive initialized successfully');
  } catch (e) {
    print('Hive initialization failed: $e');
  }
  
  runApp(const NarrativesSNSApp());
}

class NarrativesSNSApp extends StatelessWidget {
  const NarrativesSNSApp({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return MultiProvider(
      providers: [
        ChangeNotifierProvider(create: (_) => UserProvider()),
        ChangeNotifierProvider(create: (_) => PostProvider()),
        Provider<AuthService>(create: (_) => AuthService()),
        Provider<GraphQLService>(create: (_) => GraphQLService()),
      ],
      child: Consumer<GraphQLService>(
        builder: (context, graphqlService, child) {
          return GraphQLProvider(
            client: graphqlService.client,
            child: MaterialApp(
              title: 'Narratives SNS',
              theme: ThemeData(
                primarySwatch: Colors.blue,
                visualDensity: VisualDensity.adaptivePlatformDensity,
              ),
              darkTheme: ThemeData.dark(),
              themeMode: ThemeMode.system,
              home: const SplashScreen(),
              routes: {
                '/login': (context) => const LoginScreen(),
                '/home': (context) => const HomeScreen(),
              },
            ),
          );
        },
      ),
    );
  }
}

class SplashScreen extends StatelessWidget {
  const SplashScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Container(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [Color(0xFF1976D2), Color(0xFF1E88E5)],
          ),
        ),
        child: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.chat_bubble_outline, size: 100, color: Colors.white),
              const SizedBox(height: 24),
              const Text(
                'Narratives SNS',
                style: TextStyle(fontSize: 32, fontWeight: FontWeight.bold, color: Colors.white),
              ),
              const SizedBox(height: 8),
              const Text(
                'Social Network for Stories',
                style: TextStyle(fontSize: 16, color: Colors.white70),
              ),
              const SizedBox(height: 48),
              ElevatedButton(
                onPressed: () {
                  Navigator.pushReplacementNamed(context, '/home');
                },
                style: ElevatedButton.styleFrom(
                  backgroundColor: Colors.white,
                  foregroundColor: Color(0xFF1976D2),
                  padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 16),
                ),
                child: const Text('Enter App', style: TextStyle(fontSize: 18)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class LoginScreen extends StatelessWidget {
  const LoginScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Login')),
      body: const Center(child: Text('Login Screen - Coming Soon')),
    );
  }
}

class HomeScreen extends StatefulWidget {
  const HomeScreen({Key? key}) : super(key: key);

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _currentIndex = 0;
  late PageController _pageController;

  @override
  void initState() {
    super.initState();
    _pageController = PageController();
    
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<PostProvider>().loadPosts();
      context.read<UserProvider>().loadCurrentUser();
    });
  }

  void _onTabTapped(int index) {
    if (index == 2) {
      _showCreatePostDialog();
    } else {
      final actualIndex = index > 2 ? index - 1 : index;
      setState(() {
        _currentIndex = actualIndex;
      });
      _pageController.animateToPage(
        actualIndex,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeInOut,
      );
    }
  }

  void _showCreatePostDialog() {
    showDialog(
      context: context,
      builder: (context) => const CreatePostDialog(),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: PageView(
        controller: _pageController,
        onPageChanged: (index) {
          setState(() {
            _currentIndex = index;
          });
        },
        children: const [
          _FeedTab(),
          _ExploreTab(),
          _NotificationsTab(),
          _ProfileTab(),
        ],
      ),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _currentIndex >= 2 ? _currentIndex + 1 : _currentIndex,
        onTap: _onTabTapped,
        selectedItemColor: Colors.blue,
        unselectedItemColor: Colors.grey,
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.home), label: 'Home'),
          BottomNavigationBarItem(icon: Icon(Icons.explore), label: 'Explore'),
          BottomNavigationBarItem(icon: Icon(Icons.add_circle, size: 32), label: 'Post'),
          BottomNavigationBarItem(icon: Icon(Icons.notifications), label: 'Notifications'),
          BottomNavigationBarItem(icon: Icon(Icons.person), label: 'Profile'),
        ],
      ),
    );
  }

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }
}

class _FeedTab extends StatelessWidget {
  const _FeedTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Narratives'), elevation: 0),
      body: Consumer<PostProvider>(
        builder: (context, postProvider, child) {
          if (postProvider.isLoading && postProvider.posts.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }
          if (postProvider.posts.isEmpty) {
            return const Center(child: Text('No posts yet'));
          }
          return RefreshIndicator(
            onRefresh: () => postProvider.refreshPosts(),
            child: ListView.builder(
              itemCount: postProvider.posts.length,
              itemBuilder: (context, index) {
                final post = postProvider.posts[index];
                return PostCard(post: post);
              },
            ),
          );
        },
      ),
    );
  }
}

class _ExploreTab extends StatelessWidget {
  const _ExploreTab();
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Explore'), elevation: 0),
      body: const Center(child: Text('Explore Tab - Coming Soon')),
    );
  }
}

class _NotificationsTab extends StatelessWidget {
  const _NotificationsTab();
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Notifications'), elevation: 0),
      body: const Center(child: Text('Notifications Tab - Coming Soon')),
    );
  }
}

class _ProfileTab extends StatelessWidget {
  const _ProfileTab();
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Profile'), elevation: 0),
      body: const ProfileEditForm(),
    );
  }
}

class ProfileEditForm extends StatefulWidget {
  const ProfileEditForm({Key? key}) : super(key: key);
  @override
  State<ProfileEditForm> createState() => _ProfileEditFormState();
}

class _ProfileEditFormState extends State<ProfileEditForm> {
  final _formKey = GlobalKey<FormState>();
  final _firstNameController = TextEditingController();
  final _lastNameController = TextEditingController();
  final _emailController = TextEditingController();
  final _avatarNameController = TextEditingController();
  final _bioController = TextEditingController();
  
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    _loadUserData();
  }

  void _loadUserData() async {
    final userProvider = context.read<UserProvider>();
    await userProvider.loadCurrentUser();
    
    final userData = userProvider.currentUser;
    if (userData != null) {
      setState(() {
        _firstNameController.text = userData['firstName'] ?? '';
        _lastNameController.text = userData['lastName'] ?? '';
        _emailController.text = userData['emailAddress'] ?? '';
        _avatarNameController.text = userData['avatarName'] ?? '';
        _bioController.text = userData['bio'] ?? '';
      });
    }
  }

  Future<void> _saveProfile() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() { _isLoading = true; });

    try {
      final userProvider = context.read<UserProvider>();
      await userProvider.updateProfile({
        'firstName': _firstNameController.text,
        'lastName': _lastNameController.text,
        'emailAddress': _emailController.text,
        'avatarName': _avatarNameController.text,
        'bio': _bioController.text,
      });

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Profile saved!'), backgroundColor: Colors.green),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to save: $e'), backgroundColor: Colors.red),
        );
      }
    } finally {
      setState(() { _isLoading = false; });
    }
  }

  @override
  Widget build(BuildContext context) {
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16.0),
      child: Form(
        key: _formKey,
        child: Column(
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16.0),
                child: Row(
                  children: [
                    CircleAvatar(
                      radius: 40,
                      child: Text(
                        '${_firstNameController.text.isNotEmpty ? _firstNameController.text[0] : 'U'}'
                        '${_lastNameController.text.isNotEmpty ? _lastNameController.text[0] : ''}',
                        style: const TextStyle(fontSize: 24),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            _avatarNameController.text.isNotEmpty
                                ? _avatarNameController.text
                                : '${_lastNameController.text} ${_firstNameController.text}',
                            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                          ),
                          Text(_emailController.text, style: const TextStyle(color: Colors.grey)),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 20),
            TextFormField(
              controller: _firstNameController,
              decoration: const InputDecoration(labelText: 'First Name', border: OutlineInputBorder()),
              validator: (value) => value?.isEmpty == true ? 'Please enter first name' : null,
              onChanged: (value) => setState(() {}),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _lastNameController,
              decoration: const InputDecoration(labelText: 'Last Name', border: OutlineInputBorder()),
              validator: (value) => value?.isEmpty == true ? 'Please enter last name' : null,
              onChanged: (value) => setState(() {}),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _emailController,
              decoration: const InputDecoration(labelText: 'Email', border: OutlineInputBorder()),
              keyboardType: TextInputType.emailAddress,
              validator: (value) {
                if (value?.isEmpty == true) return 'Please enter email';
                if (!RegExp(r'^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$').hasMatch(value!)) return 'Please enter valid email';
                return null;
              },
              onChanged: (value) => setState(() {}),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _avatarNameController,
              decoration: const InputDecoration(labelText: 'Display Name', border: OutlineInputBorder()),
              onChanged: (value) => setState(() {}),
            ),
            const SizedBox(height: 16),
            TextFormField(
              controller: _bioController,
              decoration: const InputDecoration(labelText: 'Bio', border: OutlineInputBorder()),
              maxLines: 3,
            ),
            const SizedBox(height: 32),
            SizedBox(
              width: double.infinity,
              child: ElevatedButton(
                onPressed: _isLoading ? null : _saveProfile,
                style: ElevatedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
                child: _isLoading 
                    ? const CircularProgressIndicator() 
                    : const Text('Save Profile'),
              ),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    _firstNameController.dispose();
    _lastNameController.dispose();
    _emailController.dispose();
    _avatarNameController.dispose();
    _bioController.dispose();
    super.dispose();
  }
}

class PostCard extends StatelessWidget {
  final Map<String, dynamic> post;
  const PostCard({Key? key, required this.post}) : super(key: key);
  
  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  child: Text(post['author']['name'][0]),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(post['author']['name'], style: const TextStyle(fontWeight: FontWeight.bold)),
                      Text(_formatDateTime(post['createdAt']), style: const TextStyle(color: Colors.grey, fontSize: 12)),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Text(post['content']),
            const SizedBox(height: 12),
            Row(
              children: [
                Icon(post['isLiked'] == true ? Icons.favorite : Icons.favorite_border, 
                     color: post['isLiked'] == true ? Colors.red : Colors.grey, size: 20),
                Text(' ${post['likesCount']}'),
                const SizedBox(width: 16),
                const Icon(Icons.comment_outlined, size: 20, color: Colors.grey),
                Text(' ${post['commentsCount']}'),
              ],
            ),
          ],
        ),
      ),
    );
  }
  
  String _formatDateTime(String? dateTimeString) {
    if (dateTimeString == null) return '';
    try {
      final dateTime = DateTime.parse(dateTimeString);
      final now = DateTime.now();
      final difference = now.difference(dateTime);
      
      if (difference.inMinutes < 1) return 'Just now';
      if (difference.inMinutes < 60) return '${difference.inMinutes}m ago';
      if (difference.inHours < 24) return '${difference.inHours}h ago';
      if (difference.inDays < 7) return '${difference.inDays}d ago';
      return '${dateTime.month}/${dateTime.day}';
    } catch (e) {
      return '';
    }
  }
}

class CreatePostDialog extends StatefulWidget {
  const CreatePostDialog({Key? key}) : super(key: key);
  @override
  State<CreatePostDialog> createState() => _CreatePostDialogState();
}

class _CreatePostDialogState extends State<CreatePostDialog> {
  final _textController = TextEditingController();
  final _formKey = GlobalKey<FormState>();
  bool _isLoading = false;

  @override
  Widget build(BuildContext context) {
    return Dialog(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      child: Container(
        width: MediaQuery.of(context).size.width * 0.9,
        padding: const EdgeInsets.all(20),
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Text('New Post', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
                  IconButton(onPressed: () => Navigator.pop(context), icon: const Icon(Icons.close)),
                ],
              ),
              const SizedBox(height: 16),
              TextFormField(
                controller: _textController,
                decoration: const InputDecoration(
                  labelText: 'What are you thinking?',
                  border: OutlineInputBorder(),
                ),
                maxLines: 4,
                maxLength: 300,
                validator: (value) {
                  if (value?.trim().isEmpty == true) return 'Please enter some content';
                  return null;
                },
              ),
              const SizedBox(height: 20),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: _isLoading ? null : () => Navigator.pop(context), 
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 12),
                  ElevatedButton(
                    onPressed: _isLoading ? null : _createPost,
                    style: ElevatedButton.styleFrom(
                      backgroundColor: Colors.blue, 
                      foregroundColor: Colors.white,
                    ),
                    child: _isLoading 
                        ? const SizedBox(
                            width: 20, height: 20,
                            child: CircularProgressIndicator(strokeWidth: 2, valueColor: AlwaysStoppedAnimation<Color>(Colors.white)),
                          ) 
                        : const Text('Post'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _createPost() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() { _isLoading = true; });

    try {
      final postProvider = context.read<PostProvider>();
      await postProvider.createPost(text: _textController.text);
      
      if (mounted) {
        Navigator.pop(context);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Post created!'), backgroundColor: Colors.green),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to create post: $e'), backgroundColor: Colors.red),
        );
      }
    } finally {
      setState(() { _isLoading = false; });
    }
  }

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }
}