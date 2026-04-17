import 'package:flutter/cupertino.dart';

class ServiceTypeStats {
  final ServiceTypes serviceType;
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
  final Map<ServiceTypes, ServiceTypeStats> breakdown;

  Summary({required this.breakdown});

  // Get the TOTAL_SYSTEM stats
  ServiceTypeStats get totalSystem {
    return breakdown[ServiceTypes.Total] ??
        ServiceTypeStats(serviceType: ServiceTypes.Total);
  }

  // Get stats by service type excluding TOTAL_SYSTEM
  List<ServiceTypeStats> get serviceStats {
    return breakdown.values
        .where((item) => item.serviceType != ServiceTypes.Total)
        .toList();
  }

  // Factory to create a Summary from the Go JSON response
  factory Summary.fromJson(Map<String, dynamic> json) {
    final list = json['breakdown'] as List<dynamic>?;

    if (list == null) {
      return Summary(breakdown: {});
    }

    Map<ServiceTypes, ServiceTypeStats> result = {};

    for (var e in list) {
      final k =   ServiceTypeExtension.fromString(e['service_type'] ?? "");
      final v = ServiceTypeStats.fromJson(e);
      result[k] =  v;
    }
    return Summary(breakdown: result);
  }
}

enum ServiceTypes {
  Urban,
  Regional,
  InterCidades,
  AlfaPendular,
  International,
  Total,
  Other,
}

extension ServiceTypeExtension on ServiceTypes {

  static ServiceTypes fromString(String s) {
    if (s == "U")
      return ServiceTypes.Urban;
    if (s == "R")
      return ServiceTypes.Regional;
    if (s == "IC")
      return ServiceTypes.InterCidades;
    if (s == "AP")
      return ServiceTypes.AlfaPendular;
    if (s == "IN")
      return ServiceTypes.International;
    if (s == "TOTAL_SYSTEM")
      return ServiceTypes.Total;

    return ServiceTypes.Other;
  }


String toLocalizedString(BuildContext context) {
  switch (this) {
    case ServiceTypes.Urban:
      return "Urbano";
    case ServiceTypes.Regional:
      return "Regional";
    case ServiceTypes.InterCidades:
      return "Intercidades";
    case ServiceTypes.AlfaPendular:
      return "Alfa Pendular";
    case ServiceTypes.International:
      return "Internacional";
    case ServiceTypes.Total:
      return "Total";
    default:
      return "Outro";
  }
}}
