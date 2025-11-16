import 'package:flutter/material.dart';
import '../constants/app_constants.dart';

enum DeviceType { mobile, tablet, desktop }

class ResponsiveUtils {
  static DeviceType getDeviceType(double width) {
    if (width < AppConstants.mobileBreakpoint) {
      return DeviceType.mobile;
    } else if (width < AppConstants.tabletBreakpoint) {
      return DeviceType.tablet;
    } else {
      return DeviceType.desktop;
    }
  }

  static bool isMobile(BuildContext context) {
    return MediaQuery.of(context).size.width < AppConstants.mobileBreakpoint;
  }

  static bool isTablet(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    return width >= AppConstants.mobileBreakpoint && width < AppConstants.tabletBreakpoint;
  }

  static bool isDesktop(BuildContext context) {
    return MediaQuery.of(context).size.width >= AppConstants.tabletBreakpoint;
  }

  static double getResponsivePadding(BuildContext context) {
    final deviceType = getDeviceType(MediaQuery.of(context).size.width);
    switch (deviceType) {
      case DeviceType.mobile:
        return AppConstants.smallPadding;
      case DeviceType.tablet:
        return AppConstants.defaultPadding;
      case DeviceType.desktop:
        return AppConstants.largePadding;
    }
  }

  static double getResponsiveFontSize(BuildContext context, double baseFontSize) {
    final width = MediaQuery.of(context).size.width;
    if (width < AppConstants.mobileBreakpoint) {
      return baseFontSize * 0.9; // Slightly smaller on mobile
    } else if (width < AppConstants.tabletBreakpoint) {
      return baseFontSize; // Base size on tablet
    } else {
      return baseFontSize * 1.1; // Slightly larger on desktop
    }
  }

  static EdgeInsets getResponsiveMargin(BuildContext context) {
    final deviceType = getDeviceType(MediaQuery.of(context).size.width);
    switch (deviceType) {
      case DeviceType.mobile:
        return const EdgeInsets.all(8.0);
      case DeviceType.tablet:
        return const EdgeInsets.all(16.0);
      case DeviceType.desktop:
        return const EdgeInsets.all(24.0);
    }
  }

  static int getResponsiveColumns(BuildContext context, int maxColumns) {
    final width = MediaQuery.of(context).size.width;
    if (width < 600) return 1;
    if (width < 900) return 2;
    if (width < 1200) return 3;
    return maxColumns;
  }
}