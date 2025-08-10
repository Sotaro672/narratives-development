import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:provider/provider.dart';
import 'package:graphql_flutter/graphql_flutter.dart';

// Services
class AuthService {
  bool get isAuthenticated => false;
  
  Future<void> signOut() async {
    // TODO: Implement sign out
  }
  
  Future<bool> signInWithEmailAndPassword(String email, String password) async {
    // TODO: Implement sign in
    return false;
  }
}

class GraphQLService {
  late ValueNotifier<GraphQLClient> client;
  
  GraphQLService() {
    final HttpLink httpLink = HttpLink('http://localhost:8080/graphql');
    client = ValueNotifier(
      GraphQLClient(
        cache: GraphQLCache(store: HiveStore()),
        link: httpLink,
      ),
    );
  }
}

// Providers
class UserProvider extends ChangeNotifier {
  bool _isLoading = false;
  Map<String, dynamic>? _currentUser;
  
  bool get isLoading => _isLoading;
  Map<String, dynamic>? get currentUser => _currentUser;
  
  Future<void> loadCurrentUser() async {
    _isLoading = true;
    notifyListeners();
    
    // Simulate loading
    await Future.delayed(const Duration(seconds: 1));
    
    _currentUser = {
      'id': '1',
      'name': 'Demo User',
      'email': 'demo@example.com',
    };
    
    _isLoading = false;
    notifyListeners();
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
    
    // Simulate loading posts
    await Future.delayed(const Duration(seconds: 1));
    
    _posts = [
      {
        'id': '1',
        'content': 'Welcome to Narratives SNS!',
        'author': {'name': 'System', 'avatar': null},
        'createdAt': DateTime.now().toIso8601String(),
        'likesCount': 5,
        'commentsCount': 2,
      },
      {
        'id': '2',
        'content': 'This is a demo post to show the app is working.',
        'author': {'name': 'Demo User', 'avatar': null},
        'createdAt': DateTime.now().subtract(const Duration(hours: 1)).toIso8601String(),
        'likesCount': 3,
        'commentsCount': 1,
      },
    ];
    
    _isLoading = false;
    notifyListeners();
  }
  
  Future<void> refreshPosts() async {
    await loadPosts();
  }
}

// Widgets
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
                      Text(
                        post['author']['name'],
                        style: const TextStyle(fontWeight: FontWeight.bold),
                      ),
                      Text(
                        DateTime.parse(post['createdAt']).toString().substring(0, 16),
                        style: const TextStyle(color: Colors.grey, fontSize: 12),
                      ),
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
                Icon(Icons.favorite_border, size: 20),
                Text(' ${post['likesCount']}'),
                const SizedBox(width: 16),
                Icon(Icons.comment_outlined, size: 20),
                Text(' ${post['commentsCount']}'),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class CreatePostFAB extends StatelessWidget {
  const CreatePostFAB({Key? key}) : super(key: key);
  
  @override
  Widget build(BuildContext context) {
    return FloatingActionButton(
      onPressed: () {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Create post feature coming soon!')),
        );
      },
      child: const Icon(Icons.add),
    );
  }
}

// Screens
class SplashScreen extends StatelessWidget {
  const SplashScreen({Key? key}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const FlutterLogo(size: 100),
            const SizedBox(height: 24),
            const Text(
              'Narratives SNS',
              style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () {
                Navigator.pushReplacementNamed(context, '/home');
              },
              child: const Text('Enter App'),
            ),
          ],
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
      body: const Center(
        child: Text('Login Screen - Coming Soon'),
      ),
    );
  }
}

class ProfileScreen extends StatelessWidget {
  const ProfileScreen({Key? key}) : super(key: key);
  
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Profile')),
      body: const Center(
        child: Text('Profile Screen - Coming Soon'),
      ),
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
    
    // Load initial data
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<PostProvider>().loadPosts();
      context.read<UserProvider>().loadCurrentUser();
    });
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
          ProfileScreen(),
        ],
      ),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _currentIndex,
        onTap: (index) {
          setState(() {
            _currentIndex = index;
          });
          _pageController.animateToPage(
            index,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeInOut,
          );
        },
        items: const [
          BottomNavigationBarItem(
            icon: Icon(Icons.home),
            label: 'ホーム',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.explore),
            label: '発見',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.notifications),
            label: '通知',
          ),
          BottomNavigationBarItem(
            icon: Icon(Icons.person),
            label: 'プロフィール',
          ),
        ],
      ),
      floatingActionButton: _currentIndex == 0 ? const CreatePostFAB() : null,
    );
  }
}

class _FeedTab extends StatelessWidget {
  const _FeedTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Narratives'),
        elevation: 0,
      ),
      body: Consumer<PostProvider>(
        builder: (context, postProvider, child) {
          if (postProvider.isLoading && postProvider.posts.isEmpty) {
            return const Center(child: CircularProgressIndicator());
          }

          if (postProvider.posts.isEmpty) {
            return const Center(
              child: Text('まだ投稿がありません'),
            );
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
      appBar: AppBar(
        title: const Text('発見'),
        elevation: 0,
      ),
      body: const Center(
        child: Text('発見タブ（実装予定）'),
      ),
    );
  }
}

class _NotificationsTab extends StatelessWidget {
  const _NotificationsTab();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('通知'),
        elevation: 0,
      ),
      body: const Center(
        child: Text('通知タブ（実装予定）'),
      ),
    );
  }
}

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  
  // Initialize Firebase only if not on web or with proper options
  try {
    if (kIsWeb) {
      // Web-specific Firebase configuration
      await Firebase.initializeApp(
        options: const FirebaseOptions(
          apiKey: "demo-api-key",
          authDomain: "narratives-development-26c2d.firebaseapp.com",
          projectId: "narratives-development-26c2d",
          storageBucket: "narratives-development-26c2d.appspot.com",
          messagingSenderId: "229613581466",
          appId: "demo-app-id",
        ),
      );
    } else {
      // Native platforms - use default initialization
      await Firebase.initializeApp();
    }
    print('Firebase initialized successfully');
  } catch (e) {
    print('Firebase initialization failed: $e');
  }
  
  // Initialize Hive for GraphQL caching
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
