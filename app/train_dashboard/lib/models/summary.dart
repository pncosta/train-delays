class Summary {
  final double totalTrains;
  final double avgDelay;
  final double delayedPercentage;
  final double periodDays;

  Summary({
    required this.totalTrains,
    required this.avgDelay,
    required this.delayedPercentage,
    required this.periodDays,
  });

  // Factory to create a Summary from the Go JSON response
  factory Summary.fromJson(Map<String, dynamic> json) {
    return Summary(
      totalTrains: (json['total_trains'] as num?)?.toDouble() ?? 0.0,
      avgDelay: (json['avg_delay'] as num?)?.toDouble() ?? 0.0,
      delayedPercentage: (json['delayed_percentage'] as num?)?.toDouble() ?? 0.0,
      periodDays: json['period_days'] ?? 7,
    );
  }
}
