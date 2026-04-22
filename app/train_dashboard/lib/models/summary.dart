import 'package:flutter/cupertino.dart';
import 'package:intl/intl.dart';

class ServiceTypeStats {
  final ServiceType serviceType;
  final int totalTrips;
  final double avgDelay;
  final int onTimeCount;
  final int delayedCount;
  final int cancelledCount;

  ServiceTypeStats({
    required this.serviceType,
    this.totalTrips = 0,
    this.avgDelay = 0,
    this.onTimeCount = 0,
    this.delayedCount = 0,
    this.cancelledCount = 0,
  });

  factory ServiceTypeStats.fromJson(Map<String, dynamic> json) {
    return ServiceTypeStats(
      serviceType: ServiceTypeExtension.fromString(json['service_type'] ?? ''),
      totalTrips: json['total_trips'] ?? 0,
      avgDelay: (json['avg_delay'] as num?)?.toDouble() ?? 0.0,
      onTimeCount: json['on_time_count'] ?? 0,
      delayedCount: json['delayed_count'] ?? 0,
      cancelledCount: json['cancelled_count'] ?? 0,
    );
  }

  double get delayedPercentage {
    if (totalTrips == 0) return 0.0;
    return (delayedCount / totalTrips) * 100;
  }

  double get onTimePercentage {
    if (totalTrips == 0) return 0.0;
    return (onTimeCount / totalTrips) * 100;
  }

  double get cancelledPercentage {
    if (totalTrips == 0) return 0.0;
    return (cancelledCount / totalTrips) * 100;
  }
}

class Summary {
  final Map<ServiceType, ServiceTypeStats> breakdown;

  Summary({required this.breakdown});

  // Get the TOTAL_SYSTEM stats
  ServiceTypeStats get totalSystem {
    return breakdown[ServiceType.Total] ??
        ServiceTypeStats(serviceType: ServiceType.Total);
  }

  // Get stats by service type excluding TOTAL_SYSTEM
  List<ServiceTypeStats> get serviceStats {
    return breakdown.values
        .where((item) => item.serviceType != ServiceType.Total)
        .toList();
  }

  // Factory to create a Summary from the Go JSON response
  factory Summary.fromJson(Map<String, dynamic> json) {
    final list = json['breakdown'] as List<dynamic>?;

    if (list == null) {
      return Summary(breakdown: {});
    }

    Map<ServiceType, ServiceTypeStats> result = {};

    for (var e in list) {
      final k = ServiceTypeExtension.fromString(e['service_type'] ?? "");
      final v = ServiceTypeStats.fromJson(e);
      result[k] = v;
    }
    return Summary(breakdown: result);
  }
}

enum ServiceType {
  Urban,
  Regional,
  InterCidades,
  AlfaPendular,
  International,
  Total,
  Other,
}

extension ServiceTypeExtension on ServiceType {
  static ServiceType fromString(String s) {
    if (s == "U") return ServiceType.Urban;
    if (s == "R") return ServiceType.Regional;
    if (s == "IC") return ServiceType.InterCidades;
    if (s == "AP") return ServiceType.AlfaPendular;
    if (s == "IN") return ServiceType.International;
    if (s == "TOTAL_SYSTEM") return ServiceType.Total;

    return ServiceType.Other;
  }

  String toLocalizedString(BuildContext context) {
    switch (this) {
      case ServiceType.Urban:
        return "Urbano";
      case ServiceType.Regional:
        return "Regional";
      case ServiceType.InterCidades:
        return "Intercidades";
      case ServiceType.AlfaPendular:
        return "Alfa Pendular";
      case ServiceType.International:
        return "Internacional";
      case ServiceType.Total:
        return "Total";
      default:
        return "Outro";
    }
  }
}

class Trip {
  final String id;
  final String trainNumber;
  final ServiceType serviceType;
  final String originStation;
  final String originStationName;
  final String destinationStation;
  final String destinationStationName;

  final String? scheduledDeparture;
  final String? scheduledArrival;
  final String? actualDeparture;
  final String? actualArrival;

  final int? delayMinutes;
  final bool? isCancelled;
  final String createdAt;
  final String updatedAt;

  Trip({
    required this.id,
    required this.trainNumber,
    required this.serviceType,
    required this.originStation,
    required this.originStationName,
    required this.destinationStation,
    required this.destinationStationName,
    this.scheduledDeparture,
    this.scheduledArrival,
    this.actualDeparture,
    this.actualArrival,
    this.delayMinutes,
    this.isCancelled,
    required this.createdAt,
    required this.updatedAt,
  });

  factory Trip.fromJson(Map<String, dynamic> json) {
    return Trip(
      id: json['id'] ?? '',
      trainNumber: json['train_number'] ?? '',
      serviceType: ServiceTypeExtension.fromString(json['service_type'] ?? ''),
      originStation: json['origin_station'] ?? '',
      originStationName: json['origin_station_name'] ?? '<em falta>',
      destinationStation: json['destination_station'] ?? '',
      destinationStationName: json['destination_station_name'] ?? '<em falta>',
      scheduledDeparture: json['scheduled_departure'],
      scheduledArrival: json['scheduled_arrival'],
      actualDeparture: json['actual_departure'],
      actualArrival: json['actual_arrival'],
      delayMinutes: json['delay_minutes'],
      isCancelled: json['is_cancelled'],
      createdAt: json['created_at'] ?? '',
      updatedAt: json['updated_at'] ?? '',
    );
  }

  String get departureDate {
    DateTime dateTime = DateTime.parse(createdAt);
    return DateFormat("dd-MM-yyyy").format(dateTime);
  }

  bool get isDelayed {
    if (isCancelled == true) {
      return false;
    }
    final delayMinutes = this.delayMinutes;
    if (delayMinutes == null) {
      return false;
    }

    // For urban trains, >3 min is considered delay
    // For regional and above, >5 min is considered delay
    return (serviceType == ServiceType.Urban && delayMinutes > 3) ||
        delayMinutes > 5;
  }
}

class LeaderboardEntry {
  final String trainNumber;
  final ServiceType serviceType;
  final String originStation;
  final String originStationName;
  final String destinationStation;
  final String destinationStationName;
  final double value; // avg delay, % of cancelled, etc
  final int count; // number of trips considered for the Value

  LeaderboardEntry({
    required this.trainNumber,
    required this.serviceType,
    required this.originStation,
    required this.originStationName,
    required this.destinationStation,
    required this.destinationStationName,
    required this.value,
    required this.count,
  });

  factory LeaderboardEntry.fromJson(Map<String, dynamic> json) {
    return LeaderboardEntry(
      trainNumber: json['train_number'] ?? '',
      serviceType: ServiceTypeExtension.fromString(json['service_type'] ?? ''),
      originStation: json['origin_station'] ?? '',
      originStationName: json['origin_station_name'] ?? '<em falta>',
      destinationStation: json['destination_station'] ?? '',
      destinationStationName: json['destination_station_name'] ?? '<em falta>',
      value: (json['value'] as num?)?.toDouble() ?? 0.0,
      count: json['count'] ?? 0,
    );
  }
}
