import 'dart:convert';
import 'package:http/http.dart' as http;
import '../models/summary.dart';

class ApiService {

  final String baseUrl;

  const ApiService({this.baseUrl = "http://127.0.0.1:8080"});

  Future<Summary> getSummary({int days = 7, String? serviceType}) async {
    final Map<String, String> queryParams = {
      'days': days.toString(),
    };
    if (serviceType != null && serviceType.isNotEmpty) {
      queryParams['service_type'] = serviceType;
    }

    final uri = Uri.parse('$baseUrl/api/stats/summary').replace(
        queryParameters: queryParams);

    final response = await http.get(uri);
    if (response.statusCode >= 300) {
      throw Exception('Server error: ${response.statusCode}');
    }
    final data = json.decode(response.body);
    return Summary.fromJson(data);
  }
}
