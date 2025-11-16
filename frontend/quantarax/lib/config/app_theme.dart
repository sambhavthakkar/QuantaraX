import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class AppTheme {
  // Color Palette
  static const Color primary = Color(0xFF29DDFD);
  static const Color primaryDark = Color(0xFF0062DF);
  static const Color secondary = Color(0xFF3AAED5);
  static const Color accent = Color(0xFF3D85FD);
  static const Color background = Color(0xFF000000);
  static const Color surface = Color(0xFF141414);
  static const Color surfaceAlt = Color(0xFF171717);
  static const Color border = Color(0xFF212121);
  static const Color textPrimary = Color(0xFFFFFFFF);
  static const Color textSecondary = Color.fromARGB(255, 150, 150, 150);
  static const Color textMuted = Color.fromARGB(255, 106, 106, 106);
  static const Color highlight = Color(0xFF0166E0);
  static const Color sidebarBg = Color(0xFF171717);

  // Gradients
  static const LinearGradient primaryGradient = LinearGradient(
    begin: Alignment.topLeft,
    end: Alignment.bottomRight,
    colors: [primary, primaryDark],
  );

  static const LinearGradient buttonGlow = LinearGradient(
    begin: Alignment.bottomRight,
    end: Alignment.topLeft,
    colors: [primary, highlight],
  );

  static const RadialGradient highlightGlow = RadialGradient(
    colors: [Color(0x6629DDFD), Color(0x000062DF)],
  );

  // Shadows
  static const BoxShadow blueGlow = BoxShadow(
    color: Color(0x6629DDFD),
    blurRadius: 18,
    spreadRadius: 0,
  );

  static const BoxShadow softDark = BoxShadow(
    color: Color(0x73000000),
    blurRadius: 20,
    offset: Offset(0, 8),
  );

  // Typography
  static TextStyle get titleStyle => GoogleFonts.inter(
    fontSize: 18,
    fontWeight: FontWeight.w600,
    color: textPrimary,
  );

  static TextStyle get bodyStyle => GoogleFonts.inter(
    fontSize: 14,
    fontWeight: FontWeight.w400,
    color: textPrimary,
  );

  static TextStyle get smallStyle => GoogleFonts.inter(
    fontSize: 12,
    fontWeight: FontWeight.w400,
    color: textSecondary,
  );

  static TextStyle get mediumStyle => GoogleFonts.inter(
    fontSize: 14,
    fontWeight: FontWeight.w500,
    color: textPrimary,
  );

  // Theme Data
  static ThemeData get darkTheme => ThemeData(
    brightness: Brightness.dark,
    scaffoldBackgroundColor: background,
    primaryColor: primary,
    textTheme: GoogleFonts.interTextTheme().apply(
      bodyColor: textPrimary,
      displayColor: textPrimary,
    ),
    colorScheme: const ColorScheme.dark(
      primary: primary,
      secondary: secondary,
      surface: surface,
      background: background,
      onPrimary: Colors.black,
      onSecondary: Colors.white,
      onSurface: textPrimary,
      onBackground: textPrimary,
    ),
  );
}