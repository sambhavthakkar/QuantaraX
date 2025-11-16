import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'config/app_theme.dart';
import 'presentation/pages/desktop_page.dart';

void main() {
  runApp(const QuantaraXApp());
}

class QuantaraXApp extends StatelessWidget {
  const QuantaraXApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'QuantaraX - Resilient File Transfer',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.darkTheme,
      home: const DesktopScreen(),
      builder: (context, child) {
        return AnnotatedRegion<SystemUiOverlayStyle>(
          value: const SystemUiOverlayStyle(
            statusBarColor: Colors.transparent,
            statusBarIconBrightness: Brightness.light,
            systemNavigationBarColor: AppTheme.background,
            systemNavigationBarIconBrightness: Brightness.light,
          ),
          child: child!,
        );
      },
    );
  }
}
