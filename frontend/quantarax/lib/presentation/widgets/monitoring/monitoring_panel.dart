import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter_svg/flutter_svg.dart';
import '../../../config/app_theme.dart';
import 'package:gradient_borders/gradient_borders.dart';
import 'package:simple_gradient_text/simple_gradient_text.dart';

class MonitoringPanel extends StatelessWidget {
  final VoidCallback? onClose;
  final bool isMobile;
  const MonitoringPanel({super.key, this.onClose, this.isMobile = false});
  const MonitoringPanel.named({super.key}) : onClose = null, isMobile = false;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: isMobile ? double.infinity : 360,
      color: AppTheme.sidebarBg,
      child: Column(
        children: [
          // Header
          Container(
            padding: const EdgeInsets.all(28),
            child: Row(
              children: [
                // Back Button (mobile only)
                if (isMobile) ...[
                  InkWell(
                    borderRadius: BorderRadius.circular(20),
                    onTap: onClose,
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
                // Title
                Expanded(
                  child: Text('Transferring Monitor', style: AppTheme.titleStyle),
                ),
                // Close Button (desktop only)
                if (!isMobile)
                  InkWell(
                    borderRadius: BorderRadius.circular(16),
                    onTap: onClose,
                    child: const Padding(
                      padding: EdgeInsets.all(4.0),
                      child: Icon(Icons.close, color: AppTheme.textSecondary, size: 20),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 20),
          // Illustration
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 20),
            child: Container(
              height: 200,
              decoration: BoxDecoration(
                gradient: RadialGradient(
                  colors: [
                    AppTheme.primary.withOpacity(0.1),
                    Colors.transparent,
                  ],
                ),
                borderRadius: BorderRadius.circular(16),
              ),
              child: Center(
                child: SvgPicture.asset('assets/illustration.svg', width: 280),
              ),
            ),
          ),

          const SizedBox(height: 18),

          // Metrics
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: SingleChildScrollView(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildThroughputCard(),
                    const SizedBox(height: 12),
                    _buildMetricRow('RTT', '121 ms'),
                    const SizedBox(height: 12),
                    _buildMetricRow('Streams', '16'),
                    const SizedBox(height: 12),
                    _buildMetricRow('Loss Rate', '0.03%'),
                    const SizedBox(height: 12),
                    _buildToggleMetric('Auto-FEC', true),
                    const SizedBox(height: 12),
                    _buildToggleMetric('Encryption', true),
                    const SizedBox(height: 16),
                    _buildSpeedChart(),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildThroughputCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                'Throughput',
                style: AppTheme.bodyStyle,
              ),
              GradientText(
                '23.8 MB/s',
                style: AppTheme.bodyStyle.copyWith(
                  fontWeight: FontWeight.w600,
                ), //
                colors: AppTheme.primaryGradient.colors,
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              _buildThroughputDetail('Current', '23.8 MB/s'),
              _buildThroughputDetail('Average', '27.6 MB/s'),
              _buildThroughputDetail('Peak', '34.6 MB/s'),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildThroughputDetail(String label, String value) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label,
          style: AppTheme.smallStyle.copyWith(color: AppTheme.textSecondary),
        ),
        const SizedBox(height: 2),
        Text(
          value,
          style: AppTheme.smallStyle.copyWith(color: AppTheme.textSecondary),
        ),
      ],
    );
  }

  Widget _buildMetricRow(String label, String value) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: AppTheme.bodyStyle),
          GradientText(
            value,
            style: AppTheme.bodyStyle, //
            colors: AppTheme.primaryGradient.colors,
          ),
        ],
      ),
    );
  }

  Widget _buildToggleMetric(String label, bool isEnabled) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(8),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(label, style: AppTheme.bodyStyle),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: isEnabled
                  ? AppTheme.secondary.withOpacity(0.05)
                  : AppTheme.textMuted.withOpacity(0.1),
              borderRadius: BorderRadius.circular(4),
            ),
            child: GradientText(
              isEnabled ? 'On' : 'Off',
              style: AppTheme.smallStyle.copyWith(fontWeight: FontWeight.w500),
              colors: isEnabled ? AppTheme.primaryGradient.colors : [AppTheme.textMuted, AppTheme.textMuted],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSpeedChart() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(12),
        border: GradientBoxBorder(gradient: AppTheme.primaryGradient, width: 1),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text('Speed', style: AppTheme.bodyStyle),
              GradientText(
                '23.8 MB/s',
                style: AppTheme.bodyStyle,//
                colors: AppTheme.primaryGradient.colors,
              ),
            ],
          ),
          const SizedBox(height: 16),
          SizedBox(
            height: 60,
            child: LineChart(
              LineChartData(
                gridData: const FlGridData(show: false),
                titlesData: const FlTitlesData(show: false),
                borderData: FlBorderData(show: false),
                lineBarsData: [
                  LineChartBarData(
                    spots: [
                      const FlSpot(0, 12.3),
                      const FlSpot(1, 18.6),
                      const FlSpot(2, 22.4),
                      const FlSpot(3, 25.0),
                      const FlSpot(4, 23.8),
                      const FlSpot(5, 28.6),
                      const FlSpot(6, 26.2),
                      const FlSpot(7, 23.8),
                    ],
                    isCurved: true,
                    gradient: AppTheme.primaryGradient,
                    barWidth: 2,
                    isStrokeCapRound: true,
                    dotData: const FlDotData(show: false),
                    belowBarData: BarAreaData(
                      show: true,
                      gradient: LinearGradient(
                        begin: Alignment.topCenter,
                        end: Alignment.bottomCenter,
                        colors: [
                          AppTheme.primary.withOpacity(0.3),
                          AppTheme.primary.withOpacity(0.0),
                        ],
                      ),
                    ),
                  ),
                ],
                minX: 0,
                maxX: 7,
                minY: 10,
                maxY: 35,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
