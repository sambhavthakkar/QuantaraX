import 'package:flutter/material.dart';
import 'package:gradient_borders/box_borders/gradient_box_border.dart';
import 'package:shimmer/shimmer.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:simple_gradient_text/simple_gradient_text.dart';
import '../../../config/app_theme.dart';
import '../../../core/services/transfer_store.dart';


class ChatPanel extends StatelessWidget {
  final VoidCallback? onToggleMonitor;
  final bool isMonitorOpen;
  final bool isMobile;
  final String? chatTitle;
  
  const ChatPanel({
    super.key, 
    this.onToggleMonitor, 
    this.isMonitorOpen = false, 
    this.isMobile = false,
    this.chatTitle,
  });
  const ChatPanel.simple({super.key}) : onToggleMonitor = null, isMonitorOpen = false, isMobile = false, chatTitle = null;

  Future<void> _pickFile(BuildContext context) async {
    final result = await FilePicker.platform.pickFiles(allowMultiple: false);
    if (result == null || result.files.isEmpty) return;
    final file = result.files.first;

    // Wire to API controls: show panel at top
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text('Selected: ${file.name} (${file.size} bytes)'),
        behavior: SnackBarBehavior.floating,
        duration: const Duration(seconds: 2),
      ),
    );

    // TODO: Wire this to create a transfer card or start a session
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      color: AppTheme.background,
      child: Column(
        children: [
          // Header
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            decoration: BoxDecoration(
              color: AppTheme.sidebarBg,
              border: Border(bottom: BorderSide(color: AppTheme.border)),
            ),
            child: Row(
              children: [
                // Back Button (mobile only)
                if (isMobile) ...[
                  InkWell(
                    borderRadius: BorderRadius.circular(20),
                    onTap: () => Navigator.pop(context),
                    child: const Padding(
                      padding: EdgeInsets.all(4),
                      child: Icon(
                        Icons.arrow_back_ios,
                        color: AppTheme.textSecondary,
                        size: 20,
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                ],
                // Chat Title
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        chatTitle ?? 'Project Trackshift', 
                        style: AppTheme.titleStyle,
                      ),
                      const SizedBox(height: 2),
                      Text('@css.quantara', style: AppTheme.smallStyle),
                    ],
                  ),
                ),
                Material(
                  color: Colors.transparent,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(8),
                    onTap: onToggleMonitor,
                    child: Ink(
                      padding: const EdgeInsets.all(8),
                      decoration: BoxDecoration(
                        color: AppTheme.surface,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: SvgPicture.asset(
                        'assets/icons/monitoring-icon.svg',
                        width: 20,
                        height: 20,
                        colorFilter: const ColorFilter.mode(
                          AppTheme.primary,
                          BlendMode.srcIn,
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),

          // Chat Messages Area
          Expanded(
            child: Stack(
              children: [
                // Background illustration
                Center(
                  child: Opacity(
                    opacity: 0.1,
                    child: SvgPicture.asset(
                      'assets/quantarax-logo.svg',
                      height: 340,
                    ),
                  ),
                ),

                // Messages (scrollable)
                Positioned.fill(
                  child: SingleChildScrollView(
                    reverse: true,
                    padding: const EdgeInsets.all(20),
                    physics: const BouncingScrollPhysics(),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        _buildMessage('Hi', isFromPeer: true),
                        const SizedBox(height: 16),
                        _buildMessage('Please send the file!', isFromPeer: true),
                        const SizedBox(height: 16),
                        _buildMessage('hnji, sending!', isFromPeer: false),
                        const SizedBox(height: 24),
                        // Transfer Cards (live)
                        StreamBuilder<List<TransferInfo>>(
                          stream: TransferStore.instance.observe(),
                          initialData: TransferStore.instance.snapshot,
                          builder: (context, snap) {
                            final items = snap.data ?? const [];
                            if (items.isEmpty) return const SizedBox.shrink();
                            return Column(
                              children: [
                                for (final t in items) ...[
                                  TransferCard(info: t),
                                  const SizedBox(height: 8),
                                ]
                              ],
                            );
                          },
                        ),
                        const SizedBox(height: 8),
                      ],
                    ),
                  ),
                ),
              ],
            ),
          ),

          // Input Bar
          Container(
            padding: const EdgeInsets.all(20),
            decoration: const BoxDecoration(
              border: Border(top: BorderSide(color: AppTheme.border)),
            ),
            child: Row(
              children: [
                Material(
                  color: Colors.transparent,
                  child: InkWell(
                    borderRadius: BorderRadius.circular(12),
                    onTap: () => _pickFile(context),
                    child: Ink(
                      padding: const EdgeInsets.all(12),
                      decoration: BoxDecoration(
                        color: AppTheme.surface,
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: AppTheme.border),
                      ),
                      child: Icon(
                        Icons.attach_file,
                        color: AppTheme.textSecondary,
                        size: 20,
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 2,
                    ),
                    decoration: BoxDecoration(
                      color: AppTheme.surface,
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(color: AppTheme.border),
                    ),
                    child: TextField(
                      decoration: InputDecoration(
                        contentPadding: const EdgeInsets.symmetric(horizontal: 8, vertical: 12),
                        hintText: 'Type a message...',
                        hintStyle: AppTheme.bodyStyle.copyWith(color: AppTheme.textMuted),
                        border: InputBorder.none,
                      ),
                      style: AppTheme.bodyStyle.copyWith(color: AppTheme.textPrimary),
                      minLines: 1,
                      maxLines: 4,
                      onSubmitted: (text) {
                        // TODO: send message
                      },
                    ),
                  ),
                ),
                const SizedBox(width: 12),
                Container(
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [const CriticalToggle()],
                  ),
                ),
                const SizedBox(width: 12),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    gradient: AppTheme.buttonGlow,
                    borderRadius: BorderRadius.circular(32),
                    boxShadow: [AppTheme.blueGlow],
                  ),
                  child: Icon(Icons.send, color: Colors.black, size: 20),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMessage(String text, {required bool isFromPeer}) {
    return Row(
      mainAxisAlignment: isFromPeer
          ? MainAxisAlignment.start
          : MainAxisAlignment.end,
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          constraints: const BoxConstraints(maxWidth: 300),
          decoration: BoxDecoration(
            color: AppTheme.surface,
            borderRadius: BorderRadius.circular(12),
            border: isFromPeer
                ? Border.all(color: AppTheme.border)
                : GradientBoxBorder(
                    gradient: AppTheme.primaryGradient,
                    width: 1,
                  ),
          ),
          child: Text(
            text,
            style: AppTheme.bodyStyle.copyWith(color: AppTheme.textPrimary),
          ),
        ),
      ],
    );
  }

  Widget _buildTransferCard() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
        boxShadow: [
          BoxShadow(
            color: AppTheme.primary.withOpacity(0.1),
            blurRadius: 20,
            spreadRadius: 0,
          ),
        ],
      ),
      child: Column(
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  gradient: AppTheme.primaryGradient,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: SvgPicture.asset(
                  'assets/icons/file-icon.svg',
                  width: 24,
                  height: 24,
                  colorFilter: const ColorFilter.mode(
                    Colors.black,
                    BlendMode.srcIn,
                  ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'design-system-v2.zip',
                      style: AppTheme.bodyStyle.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text('1.86 GB', style: AppTheme.smallStyle),
                  ],
                ),
              ),
              SvgPicture.asset(
                'assets/icons/pause-icon.svg',
                width: 24,
                height: 24,
                colorFilter: const ColorFilter.mode(
                  AppTheme.textSecondary,
                  BlendMode.srcIn,
                ),
              ),
            ],
          ),

          const SizedBox(height: 16),

          // Progress Bar
          Column(
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Text('Uploading', style: AppTheme.smallStyle),
                      SizedBox(width: 10),
                      Icon(Icons.circle, color: AppTheme.primary, size: 7),
                      SizedBox(width: 10),
                      GradientText(
                        '23.8 MB/s',
                        style: AppTheme.bodyStyle.copyWith(
                          fontWeight: FontWeight.w500,
                        ), //
                        colors: AppTheme.primaryGradient.colors,
                      ),
                    ],
                  ),
                  Row(
                    children: [
                      const SizedBox(width: 8),
                      GradientText(
                        '27%',
                        style: AppTheme.bodyStyle.copyWith(
                          fontWeight: FontWeight.w500,
                        ), //
                        colors: AppTheme.primaryGradient.colors,
                      ),
                    ],
                  ),
                ],
              ),

              const SizedBox(height: 8),

              Container(
                height: 4,
                decoration: BoxDecoration(
                  color: AppTheme.border,
                  borderRadius: BorderRadius.circular(2),
                ),
                child: Stack(
                  children: [
                    Container(
                      decoration: BoxDecoration(
                        color: AppTheme.border,
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                    FractionallySizedBox(
                      widthFactor: 0.27,
                      child: Shimmer.fromColors(
                        baseColor: AppTheme.primary,
                        highlightColor: AppTheme.primary.withOpacity(0.5),
                        child: Container(
                          decoration: BoxDecoration(
                            gradient: AppTheme.primaryGradient,
                            borderRadius: BorderRadius.circular(2),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 12),

              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text('Time remaining: 00:58', style: AppTheme.smallStyle),
                  Text('Time taken: 00:34', style: AppTheme.smallStyle),
                ],
              ),

              const SizedBox(height: 12),

              GestureDetector(
                onTap: () {},
                child: Text(
                  'Show details',
                  style: AppTheme.smallStyle.copyWith(
                    color: AppTheme.textSecondary,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class TransferCard extends StatefulWidget {
  final TransferInfo? info;
  const TransferCard({super.key, this.info});

  @override
  State<TransferCard> createState() => _TransferCardState();
}

class _TransferCardState extends State<TransferCard>
    with TickerProviderStateMixin {
  bool _showDetails = false;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
        boxShadow: [
          BoxShadow(
            color: AppTheme.primary.withOpacity(0.1),
            blurRadius: 20,
            spreadRadius: 0,
          ),
        ],
      ),
      child: Column(
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  gradient: AppTheme.primaryGradient,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: SvgPicture.asset(
                  'assets/icons/file-icon.svg',
                  width: 24,
                  height: 24,
                  colorFilter: const ColorFilter.mode(
                    Colors.black,
                    BlendMode.srcIn,
                  ),
                ),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'design-system-v2.zip',
                      style: AppTheme.bodyStyle.copyWith(
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text('1.86 GB', style: AppTheme.smallStyle),
                  ],
                ),
              ),
              SvgPicture.asset(
                'assets/icons/pause-icon.svg',
                width: 24,
                height: 24,
                colorFilter: const ColorFilter.mode(
                  AppTheme.textSecondary,
                  BlendMode.srcIn,
                ),
              ),
            ],
          ),

          const SizedBox(height: 16),

          // Progress Bar
          Column(
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Text('Uploading', style: AppTheme.smallStyle),
                      const SizedBox(width: 10),
                      const Icon(Icons.circle, color: AppTheme.primary, size: 7),
                      const SizedBox(width: 10),
                      GradientText(
                        '23.8 MB/s',
                        style: AppTheme.bodyStyle.copyWith(
                          fontWeight: FontWeight.w500,
                        ), //
                        colors: AppTheme.primaryGradient.colors,
                      ),
                    ],
                  ),
                  Row(
                    children: [
                      const SizedBox(width: 8),
                      GradientText(
                        '27%',
                        style: AppTheme.bodyStyle.copyWith(
                          fontWeight: FontWeight.w500,
                        ), //
                        colors: AppTheme.primaryGradient.colors,
                      ),
                    ],
                  ),
                ],
              ),

              const SizedBox(height: 8),

              Container(
                height: 4,
                decoration: BoxDecoration(
                  color: AppTheme.border,
                  borderRadius: BorderRadius.circular(2),
                ),
                child: Stack(
                  children: [
                    Container(
                      decoration: BoxDecoration(
                        color: AppTheme.border,
                        borderRadius: BorderRadius.circular(2),
                      ),
                    ),
                    FractionallySizedBox(
                      widthFactor: 0.27,
                      child: Shimmer.fromColors(
                        baseColor: AppTheme.primary,
                        highlightColor: AppTheme.primary.withOpacity(0.5),
                        child: Container(
                          decoration: BoxDecoration(
                            gradient: AppTheme.primaryGradient,
                            borderRadius: BorderRadius.circular(2),
                          ),
                        ),
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 12),

              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text('Time remaining: 00:58', style: AppTheme.smallStyle),
                  Text('Time taken: 00:34', style: AppTheme.smallStyle),
                ],
              ),

              const SizedBox(height: 12),

              GestureDetector(
                onTap: () => setState(() => _showDetails = !_showDetails),
                child: Text(
                  _showDetails ? 'Hide details' : 'Show details',
                  style: AppTheme.smallStyle.copyWith(
                    color: AppTheme.textSecondary,
                  ),
                ),
              ),

              AnimatedSize(
                duration: const Duration(milliseconds: 250),
                curve: Curves.easeInOut,
                child: _showDetails
                    ? Padding(
                        padding: const EdgeInsets.only(top: 12),
                        child: _TransferDetails(info: widget.info),
                      )
                    : const SizedBox.shrink(),
              ),
            ],
          ),
        ],
      ),
    );
  }
  String _formatBytes(int bytes) {
   const units = ['B','KB','MB','GB','TB'];
   double v = bytes.toDouble();
   int i = 0;
   while (v >= 1024 && i < units.length-1) { v /= 1024; i++; }
   return '${v.toStringAsFixed(2)} ${units[i]}';
 }

 String _formatEta(int seconds) {
   final m = (seconds ~/ 60).toString().padLeft(2,'0');
   final s = (seconds % 60).toString().padLeft(2,'0');
   return '$m:$s';
 }
}

class _TransferDetails extends StatelessWidget {
  final TransferInfo? info;
  const _TransferDetails({super.key, this.info});

  String _formatBytes(int bytes) {
    const units = ['B','KB','MB','GB','TB'];
    double v = bytes.toDouble();
    int i = 0;
    while (v >= 1024 && i < units.length-1) { v /= 1024; i++; }
    return '${v.toStringAsFixed(2)} ${units[i]}';
  }

  Widget _kv(String label, String value) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(width: 160, child: Text(label, style: AppTheme.smallStyle)),
        Expanded(
          child: Text(
            value,
            style: AppTheme.bodyStyle.copyWith(color: AppTheme.textPrimary),
            overflow: TextOverflow.ellipsis,
            maxLines: 2,
          ),
        ),
      ],
    );
  }


  @override
  Widget build(BuildContext context) {
    Widget row(String label, String value) {
      return Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 140,
            child: Text(label, style: AppTheme.smallStyle),
          ),
          Expanded(
            child: Text(
              value,
              style: AppTheme.bodyStyle.copyWith(color: AppTheme.textPrimary),
              overflow: TextOverflow.ellipsis,
              maxLines: 2,
            ),
          ),
        ],
      );
    }

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppTheme.surfaceAlt,
        borderRadius: BorderRadius.circular(12),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 0.6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Essentials – always visible
          _kv('File name', info?.fileName ?? '—'),
          const SizedBox(height: 8),
          _kv('File size', info?.fileSize != null ? _formatBytes(info!.fileSize!) : '—'),
          const SizedBox(height: 8),
          _kv('File type', 'ZIP'),
          const SizedBox(height: 8),
          _kv('Path', '/Users/john/Downloads/design-system-v2.zip'),

          const Divider(height: 20, color: AppTheme.border),

          // Transfer Identity
          _kv('Transfer ID', info?.sessionId ?? '—'),
          const SizedBox(height: 8),
          _kv('Sender', 'peer-local'),

          const Divider(height: 20, color: AppTheme.border),

          // Transfer Structure (QUIC + FEC)
          _kv('Chunks', info?.totalChunks != null ? '${info!.totalChunks} total' : '—'),
          const SizedBox(height: 8),
          _kv('FEC', 'Adaptive, 15% overhead'),
          const SizedBox(height: 8),
          _kv('Streams', '3'),
        ],
      ),
    );
  }
}

class CriticalToggle extends StatefulWidget {
  const CriticalToggle({super.key});

  @override
  State<CriticalToggle> createState() => _CriticalToggleState();
}

class _CriticalToggleState extends State<CriticalToggle>
    with SingleTickerProviderStateMixin {
  bool isOn = true;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => setState(() => isOn = !isOn),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          // "Critical" Text
          GradientText(
            'Critical',
            style: AppTheme.bodyStyle.copyWith(fontWeight: FontWeight.w500), //
            colors: AppTheme.primaryGradient.colors,
          ),
          const SizedBox(width: 10),

          // Toggle Button
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            width: 45,
            height: 25,
            padding: const EdgeInsets.all(3),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: AppTheme.textMuted, width: 1.2),
              color: Colors.transparent,
            ),
            child: AnimatedAlign(
              duration: const Duration(milliseconds: 300),
              alignment: isOn ? Alignment.centerRight : Alignment.centerLeft,
              curve: Curves.easeInOut,
              child: Container(
                width: 16,
                height: 16,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  gradient: LinearGradient(
                    colors: isOn
                        ? AppTheme.primaryGradient.colors
                        : [Colors.grey.shade700, Colors.grey.shade800],
                  ),
                  boxShadow: [
                    if (isOn)
                      BoxShadow(
                        color: Colors.blueAccent.withOpacity(0.7),
                        blurRadius: 10,
                        spreadRadius: 1,
                      ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
