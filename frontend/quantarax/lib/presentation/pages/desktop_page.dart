import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import '../../config/app_theme.dart';
import '../widgets/sidebar/sidebar_widget.dart';
import '../widgets/chat/chat_panel.dart';
import '../widgets/monitoring/monitoring_panel.dart';
import 'mobile_pages.dart';

class _DesktopScaffold extends StatefulWidget {
  const _DesktopScaffold({super.key});

  @override
  State<_DesktopScaffold> createState() => _DesktopScaffoldState();
}

class _DesktopScaffoldState extends State<_DesktopScaffold> {
  bool showMonitor = true;

  void _toggleMonitor() => setState(() => showMonitor = !showMonitor);

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        // Left Sidebar
        const Sidebar(),

        // Main Chat Panel
        const Expanded(
          child: ChatPanel(),
        ),

        // Right Monitoring Panel (toggleable)
        if (showMonitor)
          MonitoringPanel(onClose: _toggleMonitor),
      ],
    );
  }
}

class _TabletScaffold extends StatefulWidget {
  const _TabletScaffold({super.key});

  @override
  State<_TabletScaffold> createState() => _TabletScaffoldState();
}

class _TabletScaffoldState extends State<_TabletScaffold> {
  bool showMonitor = false;

  void _toggleMonitor() => setState(() => showMonitor = !showMonitor);

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        // Sidebar (hidden when monitor is shown)
        if (!showMonitor)
          const SizedBox(width: 320, child: Sidebar()),
        
        // Chat Panel (always visible)
        const Expanded(
          child: ChatPanel(),
        ),
        
        // Monitoring Panel (shown when toggled)
        if (showMonitor)
          MonitoringPanel(onClose: _toggleMonitor),
      ],
    );
  }
}

class DesktopScreen extends StatelessWidget {
  const DesktopScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppTheme.background,
      body: LayoutBuilder(
        builder: (context, constraints) {
          // Responsive layout logic
          if (constraints.maxWidth < 768) {
            return _buildMobileLayout();
          } else if (constraints.maxWidth < 1200) {
            return _buildTabletLayout(); // Tablet behavior up to 1200px
          } else {
            return _buildDesktopLayout(); // Full desktop with 3 panels at 1200px+
          }
        },
      ),
    );
  }

  Widget _buildDesktopLayout() {
    return const _DesktopScaffold();
  }

  Widget _old_buildDesktopLayout() {
    return const Row(
      children: [
        // Left Sidebar
        Sidebar(),
        
        // Main Chat Panel
        Expanded(
          child: ChatPanel(),
        ),
        
        // Right Monitoring Panel
        MonitoringPanel.named(),
      ],
    );
  }

  Widget _buildTabletLayout() {
    return const _TabletScaffold();
  }

  // Legacy layout preserved for tablet path
  Widget _old_buildTabletLayout() {
    return Row(
      children: [
        // Sidebar (collapsed)
        Container(
          width: 60,
          color: AppTheme.sidebarBg,
          child: Column(
            children: [
              const SizedBox(height: 20),
              SvgPicture.asset(
                'assets/quantarax-logo.svg',
                height: 28,
              ),
              const Spacer(),
              IconButton(
                onPressed: () {},
                icon: const Icon(
                  Icons.menu,
                  color: AppTheme.textSecondary,
                ),
              ),
              const SizedBox(height: 20),
            ],
          ),
        ),
        
        // Main content
        Expanded(
          child: Container(
            color: AppTheme.background,
            child: const Center(
              child: Text(
                'Chat Panel (Tablet View)',
                style: TextStyle(color: AppTheme.textPrimary),
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildMobileLayout() {
    return Navigator(
      initialRoute: '/',
      onGenerateRoute: (settings) {
        switch (settings.name) {
          case '/':
            return MaterialPageRoute(
              builder: (context) => MobileChatListScreen(
                onChatTap: (chatTitle) {
                  Navigator.pushNamed(context, '/chat', arguments: chatTitle);
                },
              ),
            );
          case '/chat':
            final chatTitle = settings.arguments as String? ?? 'Chat';
            return MaterialPageRoute(
              builder: (context) => MobileChatScreen(
                chatTitle: chatTitle,
                onMonitorTap: () => Navigator.pushNamed(context, '/monitor'),
              ),
            );
          case '/monitor':
            return MaterialPageRoute(
              builder: (context) => const MobileMonitorScreen(),
            );
          default:
            return MaterialPageRoute(
              builder: (context) => MobileChatListScreen(
                onChatTap: (chatTitle) {
                  Navigator.pushNamed(context, '/chat', arguments: chatTitle);
                },
              ),
            );
        }
      },
    );
  }

  Widget _old_buildMobileLayout() {
    return Column(
      children: [
        // Top app bar
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          decoration: const BoxDecoration(
            color: AppTheme.sidebarBg,
            border: Border(
              bottom: BorderSide(color: AppTheme.border),
            ),
          ),
          child: SafeArea(
            bottom: false,
            child: Row(
              children: [
                SvgPicture.asset(
                  'assets/quantarax-logo.svg',
                  height: 28,
                ),
                const SizedBox(width: 12),
                Text(
                  'QuantaraX',
                  style: AppTheme.titleStyle,
                ),
                const Spacer(),
                IconButton(
                  onPressed: () {},
                  icon: const Icon(
                    Icons.menu,
                    color: AppTheme.textSecondary,
                  ),
                ),
              ],
            ),
          ),
        ),
        
        // Main chat area
        const Expanded(
          child: ChatPanel(),
        ),
        
        // Bottom navigation
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
          decoration: const BoxDecoration(
            color: AppTheme.sidebarBg,
            border: Border(
              top: BorderSide(color: AppTheme.border),
            ),
          ),
          child: SafeArea(
            top: false,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceAround,
              children: [
                _buildBottomNavItem(Icons.chat_bubble_outline, 'Chats', true),
                _buildBottomNavItem(Icons.analytics_outlined, 'Monitor', false),
                _buildBottomNavItem(Icons.qr_code_scanner, 'Scan', false),
                _buildBottomNavItem(Icons.person_outline, 'Profile', false),
              ],
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildBottomNavItem(IconData icon, String label, bool isActive) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(
          icon,
          color: isActive ? AppTheme.primary : AppTheme.textSecondary,
          size: 24,
        ),
        const SizedBox(height: 4),
        Text(
          label,
          style: AppTheme.smallStyle.copyWith(
            color: isActive ? AppTheme.primary : AppTheme.textSecondary,
          ),
        ),
      ],
    );
  }
}