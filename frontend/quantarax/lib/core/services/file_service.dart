import 'dart:io';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import '../constants/app_constants.dart';
import '../errors/exceptions.dart';

abstract class FileService {
  Future<File?> pickSingleFile({List<String>? allowedExtensions});
  Future<List<File>?> pickMultipleFiles({List<String>? allowedExtensions});
  Future<bool> validateFile(File file);
  String formatFileSize(int bytes);
  String getFileExtension(String fileName);
  bool isFileTypeAllowed(String extension);
}

class FileServiceImpl implements FileService {
  @override
  Future<File?> pickSingleFile({List<String>? allowedExtensions}) async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: allowedExtensions ?? AppConstants.allowedFileExtensions,
        allowMultiple: false,
      );

      if (result != null && result.files.isNotEmpty) {
        final platformFile = result.files.first;
        if (platformFile.path != null) {
          final file = File(platformFile.path!);
          if (await validateFile(file)) {
            return file;
          }
        }
      }
      return null;
    } catch (e) {
      throw FilePickerException('Failed to pick file: ${e.toString()}');
    }
  }

  @override
  Future<List<File>?> pickMultipleFiles({List<String>? allowedExtensions}) async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: allowedExtensions ?? AppConstants.allowedFileExtensions,
        allowMultiple: true,
      );

      if (result != null && result.files.isNotEmpty) {
        final files = <File>[];
        for (final platformFile in result.files) {
          if (platformFile.path != null) {
            final file = File(platformFile.path!);
            if (await validateFile(file)) {
              files.add(file);
            }
          }
        }
        return files.isNotEmpty ? files : null;
      }
      return null;
    } catch (e) {
      throw FilePickerException('Failed to pick files: ${e.toString()}');
    }
  }

  @override
  Future<bool> validateFile(File file) async {
    try {
      // Check if file exists
      if (!await file.exists()) {
        throw FileValidationException('File does not exist');
      }

      // Check file size
      final fileSize = await file.length();
      if (fileSize > AppConstants.maxFileSize) {
        throw FileValidationException('File size exceeds maximum allowed size');
      }

      // Check file extension
      final extension = getFileExtension(file.path);
      if (!isFileTypeAllowed(extension)) {
        throw FileValidationException('File type not allowed');
      }

      return true;
    } catch (e) {
      if (kDebugMode) {
        print('File validation failed: ${e.toString()}');
      }
      return false;
    }
  }

  @override
  String formatFileSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }

  @override
  String getFileExtension(String fileName) {
    return fileName.split('.').last.toLowerCase();
  }

  @override
  bool isFileTypeAllowed(String extension) {
    return AppConstants.allowedFileExtensions.contains(extension.toLowerCase());
  }
}