import 'package:flutter_test/flutter_test.dart';
import 'package:train_dashboard/models/summary.dart';
// Import your models and extension here

void main() {
  group('Summary.fromJson', () {

    test('should create a Summary with an empty map when breakdown is null', () {
      final json = <String, dynamic>{'breakdown': null};

      final result = Summary.fromJson(json);

      expect(result.breakdown, isA<Map<ServiceType, ServiceTypeStats>>());
      expect(result.breakdown.isEmpty, true);
    });

    test('should correctly map valid JSON list to ServiceTypes enum keys', () {
      final json = {
        'breakdown': [
          {
            'service_type': 'U',
            'total_trips': 1,
            'delayed_count': 2,
            'cancelled_count': 3,
            'on_time_count': 4,
            'avg_delay': 5.0,
          },
          {
            'service_type': 'TOTAL_SYSTEM',
            'total_trips': 1000,
            'delayed_count': 50,
          }
        ]
      };

      final result = Summary.fromJson(json);

      // Verify keys are correctly mapped via your Extension
      expect(result.breakdown.containsKey(ServiceType.Urban), true);
      expect(result.breakdown.containsKey(ServiceType.Total), true);

      // Verify data is preserved
      expect(result.breakdown[ServiceType.Urban]?.totalTrips, 1);
      expect(result.breakdown[ServiceType.Urban]?.delayedCount, 2);
      expect(result.breakdown[ServiceType.Urban]?.cancelledCount, 3);
      expect(result.breakdown[ServiceType.Urban]?.onTimeCount, 4);
      expect(result.breakdown[ServiceType.Urban]?.avgDelay, 5.0);
      expect(result.breakdown[ServiceType.Total]?.totalTrips, 1000);
    });

    test('should map unknown types to ServiceTypes.Other', () {
      final json = {
        'breakdown': [
          {
            'type': 'UNKNOWN_CODE',
            'total': 10,
          }
        ]
      };

      final result = Summary.fromJson(json);

      expect(result.breakdown.containsKey(ServiceType.Other), true);
    });
  });
}
