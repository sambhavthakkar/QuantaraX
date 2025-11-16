import 'package:equatable/equatable.dart';

enum TransferStatus {
  pending,
  uploading,
  downloading,
  completed,
  paused,
  failed,
  cancelled
}

enum TransferDirection { upload, download }

class TransferModel extends Equatable {
  final String id;
  final String fileName;
  final String fileType;
  final int fileSize;
  final String filePath;
  final TransferStatus status;
  final TransferDirection direction;
  final double progress;
  final double speed; // bytes per second
  final int timeRemaining; // seconds
  final int timeElapsed; // seconds
  final String peerId;
  final String peerName;
  final String? relayId;
  final TransferDetails details;
  final DateTime createdAt;
  final DateTime? completedAt;

  const TransferModel({
    required this.id,
    required this.fileName,
    required this.fileType,
    required this.fileSize,
    required this.filePath,
    required this.status,
    required this.direction,
    required this.progress,
    required this.speed,
    required this.timeRemaining,
    required this.timeElapsed,
    required this.peerId,
    required this.peerName,
    this.relayId,
    required this.details,
    required this.createdAt,
    this.completedAt,
  });

  factory TransferModel.fromJson(Map<String, dynamic> json) {
    return TransferModel(
      id: json['id'] as String,
      fileName: json['fileName'] as String,
      fileType: json['fileType'] as String,
      fileSize: json['fileSize'] as int,
      filePath: json['filePath'] as String,
      status: TransferStatus.values.firstWhere(
        (e) => e.name == json['status'],
        orElse: () => TransferStatus.pending,
      ),
      direction: TransferDirection.values.firstWhere(
        (e) => e.name == json['direction'],
        orElse: () => TransferDirection.upload,
      ),
      progress: (json['progress'] as num).toDouble(),
      speed: (json['speed'] as num).toDouble(),
      timeRemaining: json['timeRemaining'] as int,
      timeElapsed: json['timeElapsed'] as int,
      peerId: json['peerId'] as String,
      peerName: json['peerName'] as String,
      relayId: json['relayId'] as String?,
      details: TransferDetails.fromJson(json['details'] as Map<String, dynamic>),
      createdAt: DateTime.parse(json['createdAt'] as String),
      completedAt: json['completedAt'] != null 
          ? DateTime.parse(json['completedAt'] as String) 
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'fileName': fileName,
      'fileType': fileType,
      'fileSize': fileSize,
      'filePath': filePath,
      'status': status.name,
      'direction': direction.name,
      'progress': progress,
      'speed': speed,
      'timeRemaining': timeRemaining,
      'timeElapsed': timeElapsed,
      'peerId': peerId,
      'peerName': peerName,
      'relayId': relayId,
      'details': details.toJson(),
      'createdAt': createdAt.toIso8601String(),
      'completedAt': completedAt?.toIso8601String(),
    };
  }

  @override
  List<Object?> get props => [
        id,
        fileName,
        fileType,
        fileSize,
        filePath,
        status,
        direction,
        progress,
        speed,
        timeRemaining,
        timeElapsed,
        peerId,
        peerName,
        relayId,
        details,
        createdAt,
        completedAt,
      ];
}

class TransferDetails extends Equatable {
  final int chunkCount;
  final int chunkSize;
  final String fecType;
  final double fecOverhead;
  final int streamCount;
  final String encryptionType;
  final bool isE2EEnabled;
  final String identityKey;

  const TransferDetails({
    required this.chunkCount,
    required this.chunkSize,
    required this.fecType,
    required this.fecOverhead,
    required this.streamCount,
    required this.encryptionType,
    required this.isE2EEnabled,
    required this.identityKey,
  });

  factory TransferDetails.fromJson(Map<String, dynamic> json) {
    return TransferDetails(
      chunkCount: json['chunkCount'] as int,
      chunkSize: json['chunkSize'] as int,
      fecType: json['fecType'] as String,
      fecOverhead: (json['fecOverhead'] as num).toDouble(),
      streamCount: json['streamCount'] as int,
      encryptionType: json['encryptionType'] as String,
      isE2EEnabled: json['isE2EEnabled'] as bool,
      identityKey: json['identityKey'] as String,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'chunkCount': chunkCount,
      'chunkSize': chunkSize,
      'fecType': fecType,
      'fecOverhead': fecOverhead,
      'streamCount': streamCount,
      'encryptionType': encryptionType,
      'isE2EEnabled': isE2EEnabled,
      'identityKey': identityKey,
    };
  }

  @override
  List<Object?> get props => [
        chunkCount,
        chunkSize,
        fecType,
        fecOverhead,
        streamCount,
        encryptionType,
        isE2EEnabled,
        identityKey,
      ];
}