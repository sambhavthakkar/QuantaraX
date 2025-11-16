class AppConstants {
  // App Info
  static const String appName = 'QuantaraX';
  static const String appVersion = '1.0.0';
  
  // Responsive Breakpoints
  static const double mobileBreakpoint = 768;
  static const double tabletBreakpoint = 1200;
  
  // Animation Durations
  static const Duration defaultAnimationDuration = Duration(milliseconds: 250);
  static const Duration fastAnimationDuration = Duration(milliseconds: 150);
  static const Duration slowAnimationDuration = Duration(milliseconds: 350);
  
  // UI Constants
  static const double defaultPadding = 20.0;
  static const double smallPadding = 8.0;
  static const double largePadding = 32.0;
  static const double defaultBorderRadius = 12.0;
  
  // Transfer Constants
  static const int defaultChunkSize = 4 * 1024 * 1024; // 4MB
  static const double defaultFecOverhead = 0.15; // 15%
  static const int defaultStreamCount = 3;
  
  // File Picker Constants
  static const List<String> allowedFileExtensions = [
    'zip', 'rar', '7z', 'tar', 'gz',
    'pdf', 'doc', 'docx', 'txt',
    'jpg', 'jpeg', 'png', 'gif', 'bmp',
    'mp4', 'avi', 'mov', 'wmv', 'flv',
    'mp3', 'wav', 'aac', 'flac'
  ];
  
  static const int maxFileSize = 50 * 1024 * 1024 * 1024; // 50GB
}